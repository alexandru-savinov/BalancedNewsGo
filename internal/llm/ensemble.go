package llm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// PromptVariant defines a prompt template with few-shot examples
type PromptVariant struct {
	ID       string
	Template string
	Examples []string
}

func (pv *PromptVariant) GeneratePrompt(content string) string {
	examplesText := strings.Join(pv.Examples, "\n")
	return fmt.Sprintf("%s\n%s\nArticle:\n%s", pv.Template, examplesText, content)
}

// callLLM queries a specific LLM with a prompt variant
func (c *LLMClient) callLLM(articleID int64, modelName string, promptVariant PromptVariant, content string) (float64, string, float64, string, error) {
	prompt := promptVariant.GeneratePrompt(content)

	// Compute prompt hash for logging
	h := sha256.Sum256([]byte(prompt))
	promptHash := fmt.Sprintf("%x", h[:8]) // first 8 bytes as hex string
	promptSnippet := prompt
	if len(promptSnippet) > 80 {
		promptSnippet = promptSnippet[:80] + "..."
	}

	var rawResp string
	var err error

	switch modelName {
	case "gpt-3.5", "gpt-3.5-turbo", "gpt-4", "gpt-4-turbo":
		rawResp, err = c.callOpenAIAPI(modelName, prompt)
	case "claude":
		rawResp, err = c.callClaudeAPI(prompt)
	case "finetuned":
		rawResp, err = c.callFineTunedModelAPI(prompt)
	default:
		log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | Unsupported model", articleID, modelName, promptHash)
		return 0, "", 0, "", fmt.Errorf("unsupported model: %s", modelName)
	}
	if err != nil {
		rawSnippet := rawResp
		if len(rawSnippet) > 200 {
			rawSnippet = rawSnippet[:200] + "..."
		}
		log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | API error: %v | Raw response: %s", articleID, modelName, promptHash, err, rawSnippet)
		if articleID == 133 {
			log.Printf("[DEBUG][Article 133] API error: %v", err)
			log.Printf("[DEBUG][Article 133] FULL raw response:\n%s", rawResp)
		}
		return 0, "", 0, rawResp, err
	}

	score, explanation, confidence, parseErr := parseLLMResponse(rawResp)
	if parseErr != nil {
		rawSnippet := rawResp
		if len(rawSnippet) > 200 {
			rawSnippet = rawSnippet[:200] + "..."
		}
		log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | Parse error: %v | Raw response: %s", articleID, modelName, promptHash, parseErr, rawSnippet)
		if articleID == 133 {
			log.Printf("[DEBUG][Article 133] Parse error: %v", parseErr)
			log.Printf("[DEBUG][Article 133] FULL raw response:\n%s", rawResp)
		}
		return 0, "", 0, rawResp, parseErr
	}

	log.Printf("[LLM] ArticleID %d | Model %s | PromptHash %s | Success | Score: %.3f | Confidence: %.3f", articleID, modelName, promptHash, score, confidence)

	return score, explanation, confidence, rawResp, nil
}

func (c *LLMClient) callOpenAIAPI(model, prompt string) (string, error) {
	// Use the existing OpenAILLMService via type assertion
	openaiSvc, ok := c.llmService.(*OpenAILLMService)
	if !ok {
		return "", fmt.Errorf("LLMClient.llmService is not OpenAILLMService")
	}

	resp, err := openaiSvc.callOpenAI(model, prompt)
	if err != nil {
		return "", err
	}
	return string(resp.Body()), nil
}

func (c *LLMClient) callClaudeAPI(prompt string) (string, error) {
	return "", fmt.Errorf("Claude API integration not implemented")
}

func (c *LLMClient) callFineTunedModelAPI(prompt string) (string, error) {
	return "", fmt.Errorf("Fine-tuned model API integration not implemented")
}

// parseLLMResponse extracts score, explanation, confidence from raw response
func parseLLMResponse(rawResp string) (float64, string, float64, error) {
	var resp struct {
		Score       float64 `json:"score"`
		Explanation string  `json:"explanation"`
		Confidence  float64 `json:"confidence"`
	}
	err := json.Unmarshal([]byte(rawResp), &resp)
	if err != nil {
		return 0, "", 0, err
	}
	return resp.Score, resp.Explanation, resp.Confidence, nil
}

// EnsembleAnalyze performs multi-model, multi-prompt ensemble analysis
func (c *LLMClient) EnsembleAnalyze(articleID int64, content string) (*db.LLMScore, error) {
	models := []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"}
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

	const minValid = 5
	const maxAttempts = 20
	const confidenceThreshold = 0.5

	for _, model := range models {
		validResponses := make([]SubResult, 0, minValid)
		attempts := 0
	outer:
		for attempts < maxAttempts && len(validResponses) < minValid {
			for _, pv := range promptVariants {
				attempts++
				score, explanation, confidence, rawResp, err := c.callLLM(articleID, model, pv, content)
				if err != nil {
					continue
				}
				sub := SubResult{
					Model: model, PromptVariant: pv.ID,
					Score: score, Explanation: explanation,
					Confidence: confidence, RawResponse: rawResp,
				}
				allSubResults = append(allSubResults, sub)
				if confidence >= confidenceThreshold {
					validResponses = append(validResponses, sub)
				}
				if len(validResponses) >= minValid || attempts >= maxAttempts {
					break outer
				}
			}
		}

		if len(validResponses) == 0 {
			log.Printf("[Ensemble] Model %s: no valid high-confidence responses. Failing ensemble.", model)
			return nil, fmt.Errorf("ensemble failed: no valid high-confidence responses from model %s", model)
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
			"mean":          mean,
			"weighted_mean": weightedMean,
			"variance":      variance,
			"count":         float64(len(validResponses)),
		}

		log.Printf("[Ensemble] Model %s: %d valid responses, weighted mean=%.3f, variance=%.3f", model, len(validResponses), weightedMean, variance)
	}

	if len(perModelAgg) == 0 {
		return nil, fmt.Errorf("no valid LLM responses from any model")
	}

	// Aggregate across models
	var totalWeightedSum, totalSumWeights float64
	for _, agg := range perModelAgg {
		weight := agg["count"] // or customize per model
		totalWeightedSum += agg["weighted_mean"] * weight
		totalSumWeights += weight
	}
	finalScore := totalWeightedSum / math.Max(totalSumWeights, 1e-6)

	// Compute overall variance (average of per-model variances weighted by count)
	var totalVariance float64
	for _, agg := range perModelAgg {
		totalVariance += agg["variance"] * agg["count"]
	}
	totalVariance /= math.Max(totalSumWeights, 1e-6)

	uncertaintyFlag := totalVariance > 0.1 || (totalSumWeights/float64(len(perModelAgg)*minValid) < 0.5)

	meta := map[string]interface{}{
		"all_sub_results":       allSubResults,
		"per_model_results":     perModelResults,
		"per_model_aggregation": perModelAgg,
		"final_aggregation": map[string]interface{}{
			"weighted_mean":    finalScore,
			"variance":         totalVariance,
			"uncertainty_flag": uncertaintyFlag,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
	metaBytes, _ := json.Marshal(meta)

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
				"to 1.0 (strongly right). Respond with a JSON object containing 'score', 'explanation', and 'confidence'.",
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
				promptJsonFieldsFragment,
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
				promptJsonFieldsFragment,
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
				promptJsonFieldsFragment,
			Examples: []string{
				`{"score": -1.0, "explanation": "Strongly opposes conservative viewpoints", "confidence": 0.9}`,
				`{"score": 0.0, "explanation": "Balanced or neutral reporting", "confidence": 0.95}`,
				`{"score": 1.0, "explanation": "Strongly aligns with conservative viewpoints", "confidence": 0.9}`,
			},
		},
	}
}
