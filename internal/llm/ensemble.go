package llm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// callLLM queries a specific LLM with a prompt variant
func (c *LLMClient) callLLM(articleID int64, modelName string, promptVariant PromptVariant, content string) (float64, string, float64, string, error) {
	maxRetries := 2
	var lastErr error
	var rawResp string
	var score, confidence float64
	var explanation string

	for attempt := 0; attempt <= maxRetries; attempt++ {
		prompt := promptVariant.FormatPrompt(content)

		// Compute prompt hash for logging
		h := sha256.Sum256([]byte(prompt))
		promptHash := fmt.Sprintf("%x", h[:8]) // first 8 bytes as hex string
		promptSnippet := prompt
		if len(promptSnippet) > 80 {
			promptSnippet = promptSnippet[:80] + "..."
		}
		log.Printf("Prompt snippet [%s] (attempt %d): %s", promptHash, attempt+1, promptSnippet)

		var err error
		// Use the generic LLM service stored in the client
		if c.llmService == nil {
			log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | LLM service not initialized", articleID, modelName, promptHash)
			return 0, "", 0, "", fmt.Errorf("LLM service not initialized")
		}

		// We need the raw response string and the parsed score/confidence/explanation.
		// The current HTTPLLMService.AnalyzeWithPrompt returns a *db.LLMScore which contains metadata,
		// but not necessarily the raw response string needed for parseLLMResponse here.
		// Let's adapt by calling the lower-level API call method directly.
		// Assuming c.llmService is *HTTPLLMService (might need type assertion)
		httpService, ok := c.llmService.(*HTTPLLMService)
		if !ok {
			log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | LLM service is not *HTTPLLMService", articleID, modelName, promptHash)
			return 0, "", 0, "", fmt.Errorf("LLM service is not *HTTPLLMService")
		}

		// Call the underlying API method
		apiResp, err := httpService.callLLMAPIWithKey(modelName, prompt, httpService.apiKey) // Use renamed function and pass primary key
		if err != nil {
			// Enhanced error handling for SSE/streaming errors
			if strings.Contains(err.Error(), "SSE") ||
				strings.Contains(err.Error(), "stream") ||
				strings.Contains(err.Error(), "PROCESSING") {
				// Convert to streaming-specific error
				log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | Streaming error: %v",
					articleID, modelName, promptHash, err)
				return 0, "", 0, "", LLMAPIError{
					Message:      "LLM streaming response failed",
					StatusCode:   503,
					ResponseBody: err.Error(),
					ErrorType:    ErrTypeStreaming,
				}
			}

			// Error is already logged within callLLMAPI
			lastErr = err
			// Try to get raw response body even on error for logging/parsing attempts
			if apiResp != nil {
				rawResp = apiResp.String()
			}
			continue // Retry
		}
		rawResp = apiResp.String() // Store raw response on success

		// --- BEGIN INSERTED: Check for embedded error structure ---
		var genericResponse map[string]interface{}
		if errUnmarshal := json.Unmarshal([]byte(rawResp), &genericResponse); errUnmarshal == nil {
			if errorField, ok := genericResponse["error"].(map[string]interface{}); ok {
				if message, msgOK := errorField["message"].(string); msgOK && message != "" {
					errType, _ := errorField["type"].(string)
					codeVal := errorField["code"]
					isRateLimit := strings.Contains(strings.ToLower(message), "rate limit exceeded") || fmt.Sprintf("%v", codeVal) == "429"

					if isRateLimit {
						log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | "+
							"Detected embedded rate limit: %s", articleID, modelName, promptHash, message)
						lastErr = ErrBothLLMKeysRateLimited // Use sentinel error
					} else {
						log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | "+
							"Detected embedded API error: %s", articleID, modelName, promptHash, message)
						lastErr = fmt.Errorf("API error: %s (type: %s, code: %v)", message, errType, codeVal)
					}
					continue // Skip parsing, retry loop
				}
			}
		}
		// --- END INSERTED: Check for embedded error structure ---

		var parseErr error
		// Use the renamed parser for nested JSON expected in this ensemble context
		score, explanation, confidence, parseErr = parseNestedLLMJSONResponse(rawResp)
		if parseErr != nil {
			rawSnippet := rawResp
			if len(rawSnippet) > 200 {
				rawSnippet = rawSnippet[:200] + "..."
			}
			log.Printf("[ERROR][LLM] ArticleID %d | Model %s | PromptHash %s | Parse error: %v | "+
				"Raw response: %s", articleID, modelName, promptHash, parseErr, rawSnippet)
			if articleID == 133 {
				log.Printf("[ERROR][Article 133] Parse error: %v", parseErr)
				log.Printf("[ERROR][Article 133] FULL raw response:\n%s", rawResp)
			}
			lastErr = parseErr
			continue
		}

		// Validate parsed values
		if confidence == 0 {
			log.Printf("[ERROR][LLM] ArticleID %d | Model %s | PromptHash %s | "+
				"Invalid zero confidence, retrying...", articleID, modelName, promptHash)
			lastErr = fmt.Errorf("invalid zero confidence")
			continue
		}

		log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | Success | Score: %.3f | "+
			"Confidence: %.3f", articleID, modelName, promptHash, score, confidence)
		return score, explanation, confidence, rawResp, nil
	}

	log.Printf("[ERROR][LLM] ArticleID %d | Model %s | Final failure after retries. Last error: %v", articleID, modelName, lastErr)
	return 0, "", 0, rawResp, lastErr
}

// Removed callOpenAIAPI as it's replaced by direct use of httpService.callLLMAPI

// parseNestedLLMJSONResponse extracts score, explanation, confidence from a raw response
// where the LLM is expected to return a JSON string *within* the main content field
// (e.g., {"choices":[{"message":{"content":"{\"score\":...}"}}]}), or in text format
// with patterns like "Score: X.X" and "Confidence: X.X".
func parseNestedLLMJSONResponse(rawResp string) (float64, string, float64, error) {
	log.Printf("[DEBUG][LLM] Starting to parse response: %s", rawResp)

	// Step 1: Parse the OpenAI API response JSON
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if !json.Valid([]byte(rawResp)) {
		log.Printf("[DEBUG][LLM] Outer response is not valid JSON: %s", rawResp)
	}
	err := json.Unmarshal([]byte(rawResp), &apiResp)
	if err != nil {
		log.Printf("[DEBUG][LLM] Error parsing outer JSON: %v", err)
		return 0, "", 0, fmt.Errorf("error parsing outer LLM API response JSON: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		log.Printf("[DEBUG][LLM] No choices in outer response")
		return 0, "", 0, fmt.Errorf("no choices in outer LLM API response")
	}

	// Step 2: Extract the content string
	contentStr := apiResp.Choices[0].Message.Content
	log.Printf("[DEBUG][LLM] Extracted content: %s", contentStr)

	// Step 3: First try to parse as JSON
	var innerResp struct {
		Score       float64 `json:"score"`
		Explanation string  `json:"explanation"`
		Confidence  float64 `json:"confidence"`
	}

	// Add robust backtick stripping
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```") // Matches ```json ... ``` or ``` ... ```
	matches := re.FindStringSubmatch(contentStr)
	if len(matches) >= 2 {
		contentStr = strings.TrimSpace(matches[1]) // Use the captured group
		log.Printf("[DEBUG][LLM] After backtick stripping: %s", contentStr)
	}

	// Try to parse as JSON
	if json.Valid([]byte(contentStr)) {
		if err = json.Unmarshal([]byte(contentStr), &innerResp); err == nil {
			log.Printf("[DEBUG][LLM] Successfully parsed as JSON: score=%.4f, confidence=%.4f",
				innerResp.Score, innerResp.Confidence)
			return innerResp.Score, innerResp.Explanation, innerResp.Confidence, nil
		}
	}
	log.Printf("[DEBUG][LLM] JSON parsing failed, raw content: %s, error: %v", contentStr, err)

	// Step 4: If JSON parsing fails, try to extract values using regex patterns
	// Extract score with regex
	scoreRegex := regexp.MustCompile(`Score: (-?\d+\.?\d*)`)
	scoreMatches := scoreRegex.FindStringSubmatch(contentStr)
	if len(scoreMatches) < 2 {
		log.Printf("[DEBUG][LLM] No score found in text")
		return 0, "", 0, fmt.Errorf("error parsing inner content JSON: %w", err)
	}
	score, err := strconv.ParseFloat(scoreMatches[1], 64)
	if err != nil {
		log.Printf("[DEBUG][LLM] Invalid score format: %v", err)
		return 0, "", 0, fmt.Errorf("invalid score format: %w", err)
	}

	// Extract confidence with regex
	confidenceRegex := regexp.MustCompile(`Confidence: (\d+\.?\d*)`)
	confidenceMatches := confidenceRegex.FindStringSubmatch(contentStr)
	if len(confidenceMatches) < 2 {
		log.Printf("[DEBUG][LLM] No confidence found, using default 0.5")
		return score, "Extracted from text response", 0.5, nil
	}
	confidence, err := strconv.ParseFloat(confidenceMatches[1], 64)
	if err != nil {
		log.Printf("[DEBUG][LLM] Invalid confidence format: %v", err)
		return 0, "", 0, fmt.Errorf("invalid confidence format: %w", err)
	}

	// Extract reasoning or explanation
	explanation := "Extracted from text response"
	reasoningRegex := regexp.MustCompile(`Reasoning: (.+)`)
	reasoningMatches := reasoningRegex.FindStringSubmatch(contentStr)
	if len(reasoningMatches) >= 2 {
		explanation = reasoningMatches[1]
	}

	log.Printf("[DEBUG][LLM] Successfully parsed with regex: score=%.4f, confidence=%.4f", score, confidence)
	return score, explanation, confidence, nil
}

// EnsembleAnalyze performs multi-model, multi-prompt ensemble analysis
func (c *LLMClient) EnsembleAnalyze(articleID int64, content string) (*db.LLMScore, error) {
	// Use models defined in the loaded configuration
	if c.config == nil || len(c.config.Models) == 0 {
		log.Printf("[Ensemble] ArticleID %d | Error: LLMClient config is nil or has no models defined.", articleID)
		return nil, fmt.Errorf("LLMClient config is nil or has no models defined")
	}
	// Extract model names from the config
	models := make([]string, 0, len(c.config.Models))
	for _, modelCfg := range c.config.Models {
		if modelCfg.ModelName == "" {
			log.Printf("[Ensemble] Warning: Skipping model config with empty name (Perspective: %s)", modelCfg.Perspective)
			continue
		}
		models = append(models, modelCfg.ModelName)
	}
	if len(models) == 0 {
		log.Printf("[Ensemble] ArticleID %d | Error: No valid models found in configuration after filtering.", articleID)
		return nil, fmt.Errorf("no valid models found in configuration")
	}
	log.Printf("[Ensemble] ArticleID %d | Using %d models from config: %v", articleID, len(models), models)
	promptVariants := loadPromptVariants()

	type SubResult struct {
		Model         string  `json:"model"`
		PromptVariant string  `json:"prompt_variant"`
		Score         float64 `json:"score"`
		Explanation   string  `json:"explanation"`
		Confidence    float64 `json:"confidence"`
		RawResponse   string  `json:"raw_response"`
	}

	allSubResults := make([]SubResult, 0)
	perModelResults := make(map[string][]SubResult)
	perModelAgg := make(map[string]map[string]float64)

	const minValid = 1
	const maxAttempts = 6
	const confidenceThreshold = 0.5

	// Collect all valid responses that pass the threshold
	allValidResponses := make([]SubResult, 0)

	for _, model := range models {
		validResponses := make([]SubResult, 0, minValid)
		attempts := 0
	outer:
		for attempts < maxAttempts && len(validResponses) < minValid {
			for _, pv := range promptVariants {
				for retry := 0; retry < 2 && attempts < maxAttempts && len(validResponses) < minValid; retry++ {
					attempts++
					score, explanation, confidence, rawResp, err := c.callLLM(articleID, model, pv, content)
					if err != nil {
						// Log error from callLLM but continue trying other prompts/models
						log.Printf("[Ensemble] ArticleID %d | Model %s | Prompt %s | callLLM Error: %v", articleID, model, pv.ID, err)
						continue // Don't count this as a valid response
					}
					sub := SubResult{
						Model: model, PromptVariant: pv.ID,
						Score: score, Explanation: explanation,
						Confidence: confidence, RawResponse: rawResp,
					}
					allSubResults = append(allSubResults, sub)
					if confidence >= confidenceThreshold {
						validResponses = append(validResponses, sub)
						// Also add to the overall list
						allValidResponses = append(allValidResponses, sub)
					}
					if len(validResponses) >= minValid || attempts >= maxAttempts {
						break outer // Break model loop once minValid is reached or maxAttempts
					}
				}
			}
		}

		if len(validResponses) == 0 {
			log.Printf("[Ensemble] Model %s: no valid high-confidence responses after %d attempts. Skipping model.", model, attempts)
			// Don't fail the whole ensemble here, just skip this model's contribution
			continue
		}

		var sum, weightedSum, sumWeights float64
		for _, r := range validResponses {
			sum += r.Score
			weightedSum += r.Score * r.Confidence
			sumWeights += r.Confidence
		}
		mean := sum / float64(len(validResponses))
		weightedMean := weightedSum / math.Max(sumWeights, 1e-6)

		var varianceSum float64
		for _, r := range validResponses {
			diff := r.Score - mean
			varianceSum += diff * diff
		}
		variance := varianceSum / float64(len(validResponses))

		perModelResults[model] = validResponses
		perModelAgg[model] = map[string]float64{
			"mean":           mean,
			"weighted_mean":  weightedMean,
			"variance":       variance,
			"count":          float64(len(validResponses)),
			"sum_confidence": sumWeights, // Store sum of confidence for this model
		}

		log.Printf("[Ensemble] Model %s: %d valid responses, weighted mean=%.3f, variance=%.3f, sum_confidence=%.3f",
			model, len(validResponses), weightedMean, variance, sumWeights)
	}

	if len(perModelAgg) == 0 {
		log.Printf("[Ensemble] ArticleID %d | No valid high-confidence LLM responses from any model after all attempts.", articleID)
		return nil, fmt.Errorf("no valid high-confidence LLM responses from any model")
	}

	// Aggregate across models that provided valid responses
	var totalWeightedSum, totalSumWeights float64
	for _, agg := range perModelAgg {
		// Use the sum of confidence from this model's valid responses as its weight
		// This gives more weight to models that were more confident more often
		weight := agg["sum_confidence"]
		totalWeightedSum += agg["weighted_mean"] * weight
		totalSumWeights += weight
	}
	// Avoid division by zero
	finalScore := totalWeightedSum / math.Max(totalSumWeights, 1e-9)

	// Compute overall variance (average of per-model variances weighted by sum_confidence)
	var totalVarianceSum float64
	for _, agg := range perModelAgg {
		totalVarianceSum += agg["variance"] * agg["sum_confidence"]
	}
	// Avoid division by zero
	totalVariance := totalVarianceSum / math.Max(totalSumWeights, 1e-9)

	// Calculate overall confidence based on inverse variance
	// Assumes variance is somewhat normalized (e.g., scores -1 to 1 mean variance likely 0 to ~1)
	ensembleConfidence := math.Max(0.0, 1.0-totalVariance)

	// Determine uncertainty based on variance
	uncertaintyFlag := totalVariance > 0.1 // Keep variance check for now

	log.Printf("[Ensemble] Final Score: %.4f | Variance-Based Confidence: %.4f | Variance: %.4f | Total Valid Sub-Results: %d",
		finalScore, ensembleConfidence, totalVariance, len(allValidResponses))

	meta := map[string]interface{}{
		"confidence":            ensembleConfidence, // Store variance-based confidence
		"all_sub_results":       allSubResults,
		"per_model_results":     perModelResults,
		"per_model_aggregation": perModelAgg,
		"final_aggregation": map[string]interface{}{
			"weighted_mean":    finalScore,
			"variance":         totalVariance,
			"uncertainty_flag": uncertaintyFlag,
			"total_weight":     totalSumWeights, // Include total weight used
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		log.Printf("[Ensemble] ArticleID %d | Error marshaling metadata: %v", articleID, err)
		// Proceed but log it clearly
		metaBytes = []byte(fmt.Sprintf(`{"error": "failed to marshal metadata: %v"}`, err))
	}

	return &db.LLMScore{
		Model:     "ensemble",
		Score:     finalScore,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
	}, nil
}

const promptScaleFragment = "on a scale from -1.0 (strongly left) to 1.0 (strongly right). Respond with a JSON object containing 'score', "
const promptJsonFieldsFragment = "'explanation', and 'confidence'."

// loadPromptVariants returns hardcoded prompt variants (replace with config later)
func loadPromptVariants() []PromptVariant {
	return []PromptVariant{
		{
			ID: "default",
			Template: "Please analyze the political bias of the following article on a scale from -1.0 (strongly left) " +
				"to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', " +
				"and 'confidence'. Do not include any other text or formatting.",
			Examples: []string{
				`{"score": -1.0, "explanation": "Strongly left-leaning language", "confidence": 0.9}`,
				`{"score": 0.0, "explanation": "Neutral reporting", "confidence": 0.95}`,
				`{"score": 1.0, "explanation": "Strongly right-leaning language", "confidence": 0.9}`,
			},
		},
		{
			ID: "left_focus",
			Template: "From a progressive or left-leaning perspective, analyze the political bias of the following article " +
				promptScaleFragment +
				promptJsonFieldsFragment + "\nRespond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'. Do not include any other text or formatting.",
			Examples: []string{
				`{"score": -1.0, "explanation": "Strongly aligns with progressive viewpoints", "confidence": 0.9}`,
				`{"score": 0.0, "explanation": "Balanced or neutral reporting", "confidence": 0.95}`,
				`{"score": 1.0, "explanation": "Strongly opposes progressive viewpoints", "confidence": 0.9}`,
			},
		},
		{
			ID: "center_focus",
			Template: "From a centrist or neutral perspective, analyze the political bias of the following article " +
				promptScaleFragment +
				promptJsonFieldsFragment + "\nRespond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'. Do not include any other text or formatting.",
			Examples: []string{
				`{"score": -1.0, "explanation": "Clearly favors left-leaning positions", "confidence": 0.9}`,
				`{"score": 0.0, "explanation": "Appears balanced without clear bias", "confidence": 0.95}`,
				`{"score": 1.0, "explanation": "Clearly favors right-leaning positions", "confidence": 0.9}`,
			},
		},
		{
			ID: "right_focus",
			Template: "From a conservative or right-leaning perspective, analyze the political bias of the following article " +
				promptScaleFragment +
				promptJsonFieldsFragment + "\nRespond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'. Do not include any other text or formatting.",
			Examples: []string{
				`{"score": -1.0, "explanation": "Strongly opposes conservative viewpoints", "confidence": 0.9}`,
				`{"score": 0.0, "explanation": "Balanced or neutral reporting", "confidence": 0.95}`,
				`{"score": 1.0, "explanation": "Strongly aligns with conservative viewpoints", "confidence": 0.9}`,
			},
		},
		{
			ID: "anthropic",
			Template: promptJsonFieldsFragment + "\nPlease analyze the political bias of the following article, " +
				"considering its language, framing, and source. " +
				"Your analysis should be on a scale from -1.0 (strongly left) to +1.0 (strongly right).",
			Examples: []string{
				`{"score": -1.0, "explanation": "Strongly left-leaning language", "confidence": 0.9}`,
				`{"score": 0.0, "explanation": "Neutral reporting", "confidence": 0.95}`,
				`{"score": 1.0, "explanation": "Strongly right-leaning language", "confidence": 0.9}`,
			},
		},
		{
			ID: "cohere_left",
			Template: promptJsonFieldsFragment + "\nRespond ONLY with a valid JSON object containing 'score', " +
				"'explanation', and 'confidence'. Do not include any other text or formatting.",
		},
		{
			ID: "cohere_center",
			Template: promptJsonFieldsFragment + "\nRespond ONLY with a valid JSON object containing 'score', " +
				"'explanation', and 'confidence'. Do not include any other text or formatting.",
		},
	}
}
