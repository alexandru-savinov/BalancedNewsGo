package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
)

var (
	ErrBothLLMKeysRateLimited = errors.New("both LLM API keys are rate-limited")
	LLMRateLimitErrorMessage  = "Both LLM API keys are rate-limited. Please try again later."
)

const (
	LabelUnknown = "unknown"
	LabelLeft    = "left"
	LabelRight   = "right"
	LabelNeutral = "neutral"
)

type ModelConfig struct {
	Perspective string `json:"perspective"`
	ModelName   string `json:"modelName"`
	URL         string `json:"url"`
}

type CompositeScoreConfig struct {
	Formula          string             `json:"formula"`
	Weights          map[string]float64 `json:"weights"`
	MinScore         float64            `json:"min_score"`
	MaxScore         float64            `json:"max_score"`
	DefaultMissing   float64            `json:"default_missing"`
	HandleInvalid    string             `json:"handle_invalid"`
	ConfidenceMethod string             `json:"confidence_method"`
	MinConfidence    float64            `json:"min_confidence"`
	MaxConfidence    float64            `json:"max_confidence"`
	Models           []ModelConfig      `json:"models"`
}

var (
	compositeScoreConfig     *CompositeScoreConfig
	compositeScoreConfigOnce sync.Once
)

func LoadCompositeScoreConfig() (*CompositeScoreConfig, error) {
	var err error
	compositeScoreConfigOnce.Do(func() {
		f, e := os.Open("configs/composite_score_config.json")
		if e != nil {
			err = e
			return
		}
		defer f.Close()
		decoder := json.NewDecoder(f)
		var cfg CompositeScoreConfig
		if e := decoder.Decode(&cfg); e != nil {
			err = e
			return
		}
		compositeScoreConfig = &cfg
	})
	return compositeScoreConfig, err
}

// Returns (compositeScore, confidence, error)
func ComputeCompositeScoreWithConfidence(scores []db.LLMScore) (float64, float64, error) {
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return 0, 0, err
	}

	// Map for left/center/right
	scoreMap := map[string]*float64{
		"left":   nil,
		"center": nil,
		"right":  nil,
	}
	validCount := 0
	sum := 0.0
	weightedSum := 0.0
	weightTotal := 0.0
	validModels := make(map[string]bool)
	for _, s := range scores {
		model := strings.ToLower(s.Model)
		if model == LabelLeft || model == "left" {
			model = "left"
		} else if model == LabelRight || model == "right" {
			model = "right"
		} else if model == "center" {
			model = "center"
		} else {
			continue
		}
		val := s.Score
		if cfg.HandleInvalid == "ignore" && (isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore) {
			continue
		}
		if isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore {
			val = cfg.DefaultMissing
		}
		scoreMap[model] = &val
		validCount++
		validModels[model] = true
	}

	// Use weights if formula is weighted
	for k, v := range scoreMap {
		score := cfg.DefaultMissing
		if v != nil {
			score = *v
		}
		w := 1.0
		if cfg.Formula == "weighted" {
			if weight, ok := cfg.Weights[k]; ok {
				w = weight
			}
		}
		weightedSum += score * w
		weightTotal += w
		sum += score
	}

	var composite float64
	switch cfg.Formula {
	case "average":
		composite = sum / 3.0
	case "weighted":
		if weightTotal > 0 {
			composite = weightedSum / weightTotal
		} else {
			composite = 0.0
		}
	case "min":
		composite = minNonNil(scoreMap, cfg.DefaultMissing)
	case "max":
		composite = maxNonNil(scoreMap, cfg.DefaultMissing)
	default:
		composite = sum / 3.0
	}

	// Composite favors balance (closer to center = higher score)
	composite = 1.0 - abs(composite)

	// Confidence metric
	var confidence float64
	switch cfg.ConfidenceMethod {
	case "count_valid":
		confidence = float64(len(validModels)) / 3.0
	case "spread":
		confidence = 1.0 - scoreSpread(scoreMap)
	default:
		confidence = float64(len(validModels)) / 3.0
	}
	if confidence < cfg.MinConfidence {
		confidence = cfg.MinConfidence
	}
	if confidence > cfg.MaxConfidence {
		confidence = cfg.MaxConfidence
	}

	return composite, confidence, nil
}

func ComputeCompositeScore(scores []db.LLMScore) float64 {
	score, _, _ := ComputeCompositeScoreWithConfidence(scores)
	return score
}

// Helper functions
func isInvalid(f float64) bool {
	return (f != f) || (f > 1e10) || (f < -1e10) // NaN or extreme values
}

func minNonNil(m map[string]*float64, def float64) float64 {
	min := def
	first := true
	for _, v := range m {
		if v != nil {
			if first || *v < min {
				min = *v
				first = false
			}
		}
	}
	return min
}

func maxNonNil(m map[string]*float64, def float64) float64 {
	max := def
	first := true
	for _, v := range m {
		if first || (v != nil && *v > max) {
			if v != nil {
				max = *v
			}
			first = false
		}
	}
	return max
}

func scoreSpread(m map[string]*float64) float64 {
	vals := []float64{}
	for _, v := range m {
		if v != nil {
			vals = append(vals, *v)
		}
	}
	if len(vals) < 2 {
		return 0.0
	}
	sort.Float64s(vals)
	return vals[len(vals)-1] - vals[0]
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// isRateLimitError checks if an error indicates an API rate limit.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	// Add more specific checks based on actual API error messages
	// Consider checking for specific error codes or types if available
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "quota") ||
		strings.Contains(errStr, "too many requests")
}

type BiasConfig struct {
	Categories          []string            `json:"categories"`
	ConfidenceThreshold float64             `json:"confidence_threshold"`
	KeywordHeuristics   map[string][]string `json:"keyword_heuristics"`
}

type BiasResult struct {
	Category    string  `json:"category"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

var (
	PromptTemplate string
	BiasCfg        BiasConfig
)

func LoadPromptTemplate(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func LoadBiasConfig(path string) (BiasConfig, error) {
	var cfg BiasConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(data, &cfg)

	return cfg, err
}

func init() {
	var err error

	cwd, _ := os.Getwd()
	log.Printf("Current working directory: %s", cwd)

	promptPath := os.Getenv("PROMPT_TEMPLATE_PATH")
	if promptPath != "" {
		log.Printf("Using prompt template path from environment: %s", promptPath)
		PromptTemplate, err = LoadPromptTemplate(promptPath)
		if err != nil {
			log.Fatalf("Failed to load prompt template from environment path: %v", err)
		}
	} else {
		PromptTemplate, err = LoadPromptTemplate("configs/prompt_template.txt")
		if err != nil {
			log.Printf("Error loading prompt template from configs/prompt_template.txt: %v", err)
			log.Fatalf("Failed to load prompt template: %v", err)
		}
	}

	BiasCfg, err = LoadBiasConfig("configs/bias_config.json")
	if err != nil {
		log.Printf("Error loading bias config from configs/bias_config.json: %v", err)
		log.Fatalf("Failed to load bias config: %v", err)
	}
}

// Cache key: hash(content) + model.
type cacheKey struct {
	ContentHash string
	Model       string
}

type Cache struct {
	mu    sync.RWMutex
	store map[cacheKey]*db.LLMScore
}

func NewCache() *Cache {
	return &Cache{
		store: make(map[cacheKey]*db.LLMScore),
	}
}

func (c *Cache) Get(contentHash, model string) (*db.LLMScore, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	score, ok := c.store[cacheKey{contentHash, model}]

	return score, ok
}

func (c *Cache) Set(contentHash, model string, score *db.LLMScore) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[cacheKey{contentHash, model}] = score
}

// Simple hash function (replace with a secure hash in production).
func hashContent(content string) string {
	// For simplicity, use content itself (not recommended for production)
	return content
}

type LLMService interface {
	// Analyze method removed as it relied on a default model concept,
	// which is incompatible with the ensemble configuration approach.
	// Use ensemble methods or AnalyzeWithPrompt/AnalyzeWithModel directly.
}

/*
type MockLLMService struct{}

func (m *MockLLMService) Analyze(content string) (*db.LLMScore, error) {
	score := 0.5 // fixed or random score
	metadata := `{"mock": true}`

	return &db.LLMScore{
		Model:    "mock",
		Score:    score,
		Metadata: metadata,
	}, nil
}
*/

// HTTPLLMService communicates with a generic HTTP-based LLM API.
type HTTPLLMService struct {
	client   *resty.Client
	apiKey   string
	provider string // e.g., "openai", "openrouter"
	baseURL  string // Base URL for the API endpoint
	// defaultModel field removed (Refactor: Eliminate Env Var Reliance)
}

// NewHTTPLLMService creates a new service, reading configuration from environment variables.
func NewHTTPLLMService(client *resty.Client) *HTTPLLMService {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		log.Println("LLM_PROVIDER environment variable not set, defaulting to 'openai'")
		provider = "openai" // Default provider
	}

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		// Attempt fallback to provider-specific key for backward compatibility
		fallbackKeyName := strings.ToUpper(provider) + "_API_KEY"
		apiKey = os.Getenv(fallbackKeyName)
		if apiKey == "" {
			log.Fatalf("LLM_API_KEY environment variable not set, and fallback %s also not set", fallbackKeyName)
		}
		log.Printf("Warning: Using fallback API key environment variable %s. Please set LLM_API_KEY.", fallbackKeyName)
	}

	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		// Determine default base URL based on provider
		switch provider {
		case "openrouter":
			baseURL = "https://openrouter.ai/api/v1" // Default OpenRouter URL
		case "openai":
			baseURL = "https://api.openai.com/v1" // Default OpenAI URL
		default:
			log.Fatalf("LLM_BASE_URL must be set for provider '%s'", provider)
		}
		log.Printf("LLM_BASE_URL not set, defaulting to %s for provider %s", baseURL, provider)
	}
	// Ensure base URL doesn't end with a slash for consistency
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Removed logic for LLM_DEFAULT_MODEL (Refactor: Eliminate Env Var Reliance)

	return &HTTPLLMService{
		client:   client,
		apiKey:   apiKey,
		provider: provider,
		baseURL:  baseURL,
		// defaultModel assignment removed (Refactor: Eliminate Env Var Reliance)
	}
}

// BaseURL returns the configured base URL for the LLM API.
func (s *HTTPLLMService) BaseURL() string {
	return s.baseURL
}

// DefaultModelName function removed as defaultModel field was removed.

func (s *HTTPLLMService) AnalyzeWithPrompt(model, prompt, content string) (*db.LLMScore, error) {
	prompt = strings.Replace(prompt, "{{ARTICLE_CONTENT}}", content, 1)
	var resp *resty.Response
	var err error

	// Attempt with primary key
	log.Printf("[AnalyzeWithPrompt] Attempting API call with primary key for model %s", model)
	resp, err = s.callLLMAPIWithKey(model, prompt, s.apiKey)
	if err == nil && resp != nil && resp.IsSuccess() {
		log.Printf("[AnalyzeWithPrompt] Primary key successful for model %s", model)
		return s.processLLMResponse(resp, content, model)
	}
	log.Printf("[AnalyzeWithPrompt] Primary key failed for model %s: %v", model, err)

	// If primary key failed with rate limit, try secondary key
	if isRateLimitError(err) {
		secondApiKey := os.Getenv("LLM_API_KEY_SECONDARY")
		if secondApiKey != "" && secondApiKey != s.apiKey {
			log.Printf("[AnalyzeWithPrompt] Primary key rate limited for model %s. Attempting with secondary key.", model)
			resp, err = s.callLLMAPIWithKey(model, prompt, secondApiKey)
			if err == nil && resp != nil && resp.IsSuccess() {
				log.Printf("[AnalyzeWithPrompt] Secondary key successful for model %s", model)
				return s.processLLMResponse(resp, content, model)
			}
			log.Printf("[AnalyzeWithPrompt] Secondary key failed for model %s: %v", model, err)
			// If secondary key also rate limited, return specific error
			if isRateLimitError(err) {
				log.Printf("[AnalyzeWithPrompt] Both keys rate limited for model %s.", model)
				return nil, ErrBothLLMKeysRateLimited
			}
		} else {
			log.Printf("[AnalyzeWithPrompt] Primary key rate limited for model %s, but no valid secondary key found.", model)
			// Return the original rate limit error from the primary key
			return nil, err
		}
	}

	// Return the last error encountered (either non-rate-limit from primary, or non-rate-limit from secondary)
	log.Printf("[AnalyzeWithPrompt] Failed after all attempts for model %s. Last error: %v", model, err)
	return nil, err
}

func (s *HTTPLLMService) AnalyzeWithModel(model, content string) (*db.LLMScore, error) {
	return s.AnalyzeWithPrompt(model, PromptTemplate, content)
}

// Satisfy LLMService interface

// Analyze function removed as it relied on the removed defaultModel field
// and is superseded by ensemble methods or explicit model calls like AnalyzeWithModel.

func (s *HTTPLLMService) callLLMAPIWithKey(model string, prompt string, apiKey string) (*resty.Response, error) { // Renamed and added apiKey param
	req := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+apiKey) // Use apiKey param

	body := map[string]interface{}{
		"model":       model, // Use the model passed as argument
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  300,
		"temperature": 0.7,
	}
	req.SetBody(body)

	// --- Enhanced Logging ---
	// Mask API Key for logging
	maskedAuthHeader := "Bearer ..."
	if len(apiKey) > 8 { // Use apiKey param
		// Basic masking, adjust if key format differs significantly
		prefix := strings.Split(apiKey, "_")[0]                                // e.g., "sk" for openai, "sk-or" for openrouter // Use apiKey param
		maskedAuthHeader = "Bearer " + prefix + "..." + apiKey[len(apiKey)-4:] // Use apiKey param
	}
	// Log headers (masking auth)
	log.Printf("[%s Request] Headers: Content-Type=%s, Authorization=%s",
		s.provider,
		req.Header.Get("Content-Type"),
		maskedAuthHeader)
	// Log body
	bodyBytes, _ := json.Marshal(body)
	log.Printf("[%s Request] Body: %s", s.provider, string(bodyBytes))
	// Log URL and Model being sent
	// Construct the full endpoint URL using configured base URL
	// Assuming a standard path, make this configurable if needed
	endpointPath := "/chat/completions"
	endpointURL := s.baseURL + endpointPath

	log.Printf("[%s Request] POST URL: %s | Model in Payload: %s", s.provider, endpointURL, body["model"])
	// --- End Enhanced Logging ---

	resp, err := req.Post(endpointURL) // POST to the correct endpoint path

	// --- Enhanced Response Logging ---
	if err != nil {
		// Log network/request-level errors
		log.Printf("[%s] API request error: %v", s.provider, err)
		return nil, err
	}

	// Log status and raw body regardless of success/failure
	log.Printf("[%s] Raw Response Status: %s", s.provider, resp.Status())
	log.Printf("[%s] Raw Response Body: %s", s.provider, resp.String()) // Log full body

	if !resp.IsSuccess() {
		// Try to parse standard LLM error structure (common in OpenAI/OpenRouter)
		var openRouterError struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(resp.Body(), &openRouterError); jsonErr == nil && openRouterError.Error.Message != "" {
			// Return a more specific error from OpenRouter response
			specificError := fmt.Errorf("%s API error (%s): %s (Type: %s, Code: %s)",
				s.provider, resp.Status(), openRouterError.Error.Message, openRouterError.Error.Type, openRouterError.Error.Code)
			log.Printf("[%s] %v", s.provider, specificError) // Log the specific error
			return resp, specificError
		}
		// Fallback generic error if parsing fails or structure is different
		genericError := fmt.Errorf("%s API response not successful (%s)", s.provider, resp.Status())
		log.Printf("[%s] %v", s.provider, genericError) // Log the generic error
		return resp, genericError
	}
	// --- End Enhanced Response Logging ---

	return resp, nil // Return success
}

// processLLMResponse processes the raw HTTP response from the LLM API call.
func (s *HTTPLLMService) processLLMResponse(resp *resty.Response, content string, model string) (*db.LLMScore, error) {
	// Use the renamed parser specific to this file's context
	contentResp, err := parseLLMAPIResponse(resp.Body())
	if err != nil {
		log.Printf("[processLLMResponse] Failed to parse %s response: %v\nRaw response:\n%s", s.provider, err, string(resp.Body()))
		return nil, fmt.Errorf("failed to parse %s response body: %w", s.provider, err)
	}

	log.Printf("[Analyze] Successful raw completion from model %s:\n%s", model, contentResp)

	biasResult := parseBiasResult(contentResp)
	heuristicCat := heuristicCategory(content)

	uncertain := false
	if biasResult.Category == LabelUnknown || biasResult.Confidence < 0.3 {
		uncertain = true
	}

	metadataMap := map[string]interface{}{
		"raw_response":       contentResp,
		"parsed_bias":        biasResult,
		"heuristic_category": heuristicCat,
		"uncertain":          uncertain,
	}
	metadataBytes, err := json.Marshal(metadataMap)
	if err != nil {
		log.Printf("Failed to marshal metadata: %v", err)
	}

	return &db.LLMScore{
		Model:    model,
		Score:    biasResult.Confidence,
		Metadata: string(metadataBytes),
	}, nil
}

func (s *HTTPLLMService) RobustAnalyze(content string) (*db.LLMScore, error) {
	var (
		validScores           []*db.LLMScore
		attempts              int
		maxValidScores        = 5
		maxAttemptsPerVariant = 3
		confidenceThreshold   = 0.5
	)

	// Define prompt variants
	promptVariants := []string{
		PromptTemplate,
		`Respond ONLY in this strict JSON format: {"parsed_bias":{"Category":"...", "Confidence":...}}. Article: {{ARTICLE_CONTENT}}`,
		`Return JSON: {"parsed_bias":{"Category":"...", "Confidence":...}}. Text: {{ARTICLE_CONTENT}}`,
	}

	// Define models: primary and fallback
	models := []string{"gpt-4", "gpt-3.5-turbo"}

	for _, modelName := range models {

		for variantIdx, _ := range promptVariants { // Use blank identifier for unused promptTemplate
			for attempt := 1; attempt <= maxAttemptsPerVariant; attempt++ {
				attempts++

				score, err := s.AnalyzeWithModel(modelName, content) // Corrected receiver 's' and method call, using :=

				failType := ""
				confidence := 0.0

				if err != nil || score == nil {
					failType = "api_error"
					log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Failure: %s | Error: %v", attempts, modelName, variantIdx, failType, err)
					continue
				}

				var meta struct {
					ParsedBias struct {
						Category   string  `json:"Category"`
						Confidence float64 `json:"Confidence"`
					} `json:"parsed_bias"`
				}
				if err := json.Unmarshal([]byte(score.Metadata), &meta); err != nil {
					failType = "parse_error"
					log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Failure: %s | Error: %v", attempts, modelName, variantIdx, failType, err)
					continue
				}

				cat := strings.ToLower(strings.TrimSpace(meta.ParsedBias.Category))
				confidence = meta.ParsedBias.Confidence

				if cat == "" || cat == LabelUnknown {
					failType = "empty_category"
					log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Failure: %s | Category: '%s'", attempts, modelName, variantIdx, failType, cat)
					continue
				}

				if confidence < confidenceThreshold {
					failType = "low_confidence"
					log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Failure: %s | Confidence: %.2f", attempts, modelName, variantIdx, failType, confidence)
					continue
				}

				// Success
				log.Printf("[RobustAnalyze] Attempt %d | Model: %s | PromptVariant: %d | Success | Category: '%s' | Confidence: %.2f", attempts, modelName, variantIdx, cat, confidence)
				validScores = append(validScores, score)

				if len(validScores) >= maxValidScores {
					break
				}
			}
			if len(validScores) >= maxValidScores {
				break
			}
		}
		if len(validScores) >= maxValidScores {
			break
		}
	}

	if len(validScores) < maxValidScores {
		return nil, fmt.Errorf("RobustAnalyze: only %d valid responses after %d attempts", len(validScores), attempts)
	}

	// Extract scores
	scores := make([]float64, len(validScores))
	for i, s := range validScores {
		scores[i] = s.Score
	}

	sort.Float64s(scores)

	// Average middle scores (trimmed mean)
	start := 1
	end := len(scores) - 2
	if end <= start {
		start = 0
		end = len(scores) - 1
	}
	sum := 0.0
	count := 0
	for i := start; i <= end; i++ {
		sum += scores[i]
		count++
	}
	avg := sum / float64(count)

	// Use metadata from median score
	medianIdx := len(validScores) / 2
	medianScore := validScores[medianIdx]

	return &db.LLMScore{
		Model:    "", // model info unavailable here
		Score:    avg,
		Metadata: medianScore.Metadata,
	}, nil
}

// parseLLMResponse attempts to extract the main text content from common LLM response structures.
// parseLLMAPIResponse attempts to extract the main text content from common LLM API response structures (like OpenAI/OpenRouter).
func parseLLMAPIResponse(body []byte) (string, error) {
	// First try to extract markdown-wrapped JSON (```json ... ``` or ``` ... ```)
	content := string(body)
	if strings.Contains(content, "```json") || strings.Contains(content, "```") {
		// Extract JSON between ```json and ``` or just ``` and ```
		var start, end int
		if idx := strings.Index(content, "```json"); idx >= 0 {
			start = idx + len("```json")
		} else if idx := strings.Index(content, "```"); idx >= 0 {
			start = idx + len("```")
		}
		if start > 0 {
			rest := content[start:]
			if idx := strings.Index(rest, "```"); idx >= 0 {
				end = start + idx
				content = strings.TrimSpace(content[start:end])
			}
		}
	}

	// Try parsing as direct JSON first (for models like qwen, mistral, gemini-flash)
	var directResponse struct {
		Content string `json:"content"`
		Text    string `json:"text"`   // Some models use "text" instead of "content"`
		Result  string `json:"result"` // Some models use "result"`
		Output  string `json:"output"` // Some models use "output"`
	}
	if err := json.Unmarshal([]byte(content), &directResponse); err == nil {
		// Return the first non-empty field we find
		if directResponse.Content != "" {
			return directResponse.Content, nil
		}
		if directResponse.Text != "" {
			return directResponse.Text, nil
		}
		if directResponse.Result != "" {
			return directResponse.Result, nil
		}
		if directResponse.Output != "" {
			return directResponse.Output, nil
		}
	}

	// --- Check for embedded error structure FIRST (e.g., OpenRouter rate limit in 200 OK) ---
	var genericResponse map[string]interface{}
	if err := json.Unmarshal(body, &genericResponse); err == nil {
		// Check if top-level 'error' key exists and is a map
		if errorField, ok := genericResponse["error"].(map[string]interface{}); ok {
			// Check if 'message' key exists within the error map and is a string
			if message, msgOK := errorField["message"].(string); msgOK && message != "" {
				// Extract other fields if they exist, providing defaults if not
				errType, _ := errorField["type"].(string)
				errCode, _ := errorField["code"].(interface{}) // Code might be string or number (like 429)

				// Check for specific rate limit indicators
				isRateLimit := strings.Contains(strings.ToLower(message), "rate limit exceeded") || fmt.Sprintf("%v", errCode) == "429"

				if isRateLimit {
					log.Printf("Detected rate limit error: %s (type: %s, code: %v)", message, errType, errCode) // Log detection
					return "", ErrBothLLMKeysRateLimited                                                        // Return specific sentinel error
				}
				// Otherwise, return the generic formatted error
				return "", fmt.Errorf("API error: %s (type: %s, code: %v)", message, errType, errCode)
			}
		}
	}
	// --- End embedded error check ---

	// Fall back to OpenAI-specific parsing for successful responses (choices)
	var standardResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	// Try parsing the standard success response (choices)
	if err := json.Unmarshal(body, &standardResp); err == nil {
		if len(standardResp.Choices) == 0 {
			// This case might indicate an issue, but could also be a valid empty response.
			// If we reached here, it means the generic error check above didn't find an error message.
			return "", errors.New("no choices in OpenAI response")
		}
		// Successfully parsed choices, return the content
		return standardResp.Choices[0].Message.Content, nil
	}

	// If parsing as *both* error and standard choices failed, then it's an unknown format.

	// Log raw response for debugging
	log.Printf("Failed to parse response as JSON or OpenAI format.\nRaw response:\n%s", string(body))
	return "", errors.New("invalid API response format")
}

func parseBiasResult(contentResp string) BiasResult {
	var biasResult BiasResult

	// Extract JSON strictly between triple backticks
	re := regexp.MustCompile("(?s)```(.*?)```")
	matches := re.FindStringSubmatch(contentResp)
	if len(matches) < 2 {
		log.Printf("No JSON block found between triple backticks in LLM response")
		return BiasResult{}
	}
	jsonStr := matches[1]

	if err := json.Unmarshal([]byte(jsonStr), &biasResult); err != nil {
		log.Printf("Failed to parse JSON inside triple backticks: %v", err)
		biasResult = BiasResult{}
	}

	// Validate category
	validCategory := false
	for _, cat := range BiasCfg.Categories {
		if biasResult.Category == cat {
			validCategory = true
			break
		}
	}
	if !validCategory {
		biasResult.Category = LabelUnknown
	}

	// Validate confidence
	if biasResult.Confidence < 0 || biasResult.Confidence > 1 {
		biasResult.Confidence = 0
	}

	return biasResult
}

func heuristicCategory(content string) string {
	contentLower := strings.ToLower(content)

	for cat, keywords := range BiasCfg.KeywordHeuristics {
		for _, kw := range keywords {
			if strings.Contains(contentLower, strings.ToLower(kw)) {
				return cat
			}
		}
	}

	return LabelUnknown
}

type LLMClient struct {
	client     *resty.Client
	cache      *Cache
	db         *sqlx.DB
	llmService LLMService
	config     *CompositeScoreConfig // Added field to hold ensemble config
}

func NewLLMClient(dbConn *sqlx.DB) *LLMClient {
	client := resty.New()
	cache := NewCache()

	provider := os.Getenv("LLM_PROVIDER")

	var service LLMService

	// Load the composite score configuration
	config, err := LoadCompositeScoreConfig()
	if err != nil {
		log.Fatalf("Failed to load composite score config: %v", err)
	}
	if config == nil || len(config.Models) == 0 {
		log.Fatalf("Composite score config loaded but is nil or contains no models.")
	}
	log.Printf("Loaded composite score config with %d models.", len(config.Models))

	switch provider {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			log.Fatal("ERROR: OPENAI_API_KEY not set, cannot use OpenAI LLM provider")
		}

		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-3.5-turbo"
		}

		service = NewHTTPLLMService(client) // Corrected constructor name
	case "openrouter":
		apiKey := os.Getenv("LLM_API_KEY") // Use generic LLM_API_KEY for OpenRouter
		if apiKey == "" {
			log.Fatal("ERROR: LLM_API_KEY not set, cannot use OpenRouter LLM provider")
		}
		// Base URL is handled within NewHTTPLLMService based on provider
		service = NewHTTPLLMService(client)
	default:
		log.Fatalf("ERROR: LLM_PROVIDER '%s' unknown, cannot initialize LLM service", provider)
	}

	return &LLMClient{
		client:     client,
		cache:      cache,
		db:         dbConn,
		llmService: service,
		config:     config, // Assign loaded config
	}
}

// analyzeContent uses the configured LLM service to analyze content.
// TODO: Review if urlParam is still needed or if baseURL from HTTPLLMService is sufficient.
// analyzeContent uses the configured LLM service to analyze content.
// analyzeContent uses the configured LLM service to analyze content.
func (c *LLMClient) analyzeContent(articleID int64, content string, model string) (*db.LLMScore, error) {
	log.Printf("[analyzeContent] Entry: articleID=%d, model=%s", articleID, model) // Log 1: Entry (Removed urlParam)
	log.Printf("[analyzeContent] Entry: articleID=%d, model=%s", articleID, model) // Log 1: Entry (Removed urlParam)
	contentHash := hashContent(content)

	// Check cache
	if cached, ok := c.cache.Get(contentHash, model); ok {
		return cached, nil
	}

	var score *db.LLMScore
	var err error

	// Use only the general/default prompt variant to limit to 1 API call per model
	generalPrompt := PromptVariant{
		ID: "default",
		Template: "Please analyze the political bias of the following article on a scale from -1.0 (strongly left) " +
			"to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'. Do not include any other text or formatting.",
		Examples: []string{
			`{"score": -1.0, "explanation": "Strongly left-leaning language", "confidence": 0.9}`,
			`{"score": 0.0, "explanation": "Neutral reporting", "confidence": 0.95}`,
			`{"score": 1.0, "explanation": "Strongly right-leaning language", "confidence": 0.9}`,
		},
	}

	scoreVal, explanation, confidence, _, err := c.callLLM(articleID, model, generalPrompt, content)
	if err != nil {
		return nil, err
	}

	meta := fmt.Sprintf(`{"explanation": %q, "confidence": %.3f}`, explanation, confidence)

	score = &db.LLMScore{
		ArticleID: articleID,
		Model:     model,
		Score:     scoreVal,
		Metadata:  meta,
		CreatedAt: time.Now(),
	}

	score.ArticleID = articleID
	score.Model = model
	score.CreatedAt = time.Now()

	// Cache it
	c.cache.Set(contentHash, model, score)

	return score, nil
}

func (c *LLMClient) ProcessUnscoredArticles() error {
	query := `
	SELECT a.* FROM articles a
	WHERE NOT EXISTS (
		SELECT 1 FROM llm_scores s
		WHERE s.article_id = a.id
	)
	`
	var articles []db.Article
	if err := c.db.Select(&articles, query); err != nil {
		return err
	}

	for _, article := range articles {
		if err := c.AnalyzeAndStore(&article); err != nil {
			log.Printf("Failed to analyze article ID %d: %v", article.ID, err)
		}
	}

	return nil
}

func (c *LLMClient) AnalyzeAndStore(article *db.Article) error {
	// Removed incorrect log from AnalyzeAndStore
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return fmt.Errorf("failed to load composite score config: %w", err)
	}

	for _, m := range cfg.Models {
		log.Printf("[DEBUG][AnalyzeAndStore] Article %d | Perspective: %s | ModelName passed: %s | URL: %s", article.ID, m.Perspective, m.ModelName, m.URL)
		score, err := c.analyzeContent(article.ID, article.Content, m.ModelName) // Removed URL argument
		if err != nil {
			log.Printf("Error analyzing article %d with model %s: %v", article.ID, m.ModelName, err)

			continue
		}

		_, err = db.InsertLLMScore(c.db, score)
		if err != nil {
			log.Printf("Error inserting LLM score for article %d model %s: %v", article.ID, m.ModelName, err)
		}
	}

	// Removed incorrect log from AnalyzeAndStore
	return nil
}

func (c *LLMClient) ReanalyzeArticle(articleID int64) error {
	log.Printf("[ReanalyzeArticle %d] Starting reanalysis", articleID) // Log start
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}

	// Delete existing scores
	log.Printf("[ReanalyzeArticle %d] Deleting existing scores", articleID)
	_, err = tx.Exec("DELETE FROM llm_scores WHERE article_id = ?", articleID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("tx.Rollback() failed: %v", rbErr)
		}

		return err
	}

	var article db.Article

	log.Printf("[ReanalyzeArticle %d] Fetching article data", articleID)
	err = tx.Get(&article, "SELECT * FROM articles WHERE id = ?", articleID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("tx.Rollback() failed: %v", rbErr)
		}

		return err
	}

	log.Printf("[ReanalyzeArticle %d] Fetched article: Title='%.50s'", articleID, article.Title)
	log.Printf("[ReanalyzeArticle %d] Loading composite score config", articleID)
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return fmt.Errorf("failed to load composite score config: %w", err)
	}

	log.Printf("[ReanalyzeArticle %d] Starting analysis loop for %d models", articleID, len(cfg.Models))
	for _, m := range cfg.Models {
		log.Printf("[ReanalyzeArticle %d] Calling analyzeContent for model: %s", articleID, m.ModelName)
		score, err := c.analyzeContent(article.ID, article.Content, m.ModelName) // Removed URL argument
		if err != nil {
			log.Printf("[ReanalyzeArticle %d] Error from analyzeContent for %s: %v", articleID, m.ModelName, err)
			continue
		}
		log.Printf("[ReanalyzeArticle %d] analyzeContent successful for: %s. Score: %.2f", articleID, m.ModelName, score.Score)
		// Removed extraneous closing brace

		_, err = tx.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata)
			VALUES (:article_id, :model, :score, :metadata)`, score)
		if err != nil {
			log.Printf("[ReanalyzeArticle %d] Error inserting score for %s: %v", articleID, m.ModelName, err)
		} else {
			log.Printf("[ReanalyzeArticle %d] Successfully inserted score for: %s", articleID, m.ModelName)
		}
	}

	// Call ensemble aggregation and save result
	ensembleScore, err := c.EnsembleAnalyze(article.ID, article.Content)
	if err != nil {
		log.Printf("Error generating ensemble score for article %d: %v", article.ID, err)
	} else {
		ensembleScore.ArticleID = article.ID
		_, err = tx.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata, created_at)
			VALUES (:article_id, :model, :score, :metadata, :created_at)`, ensembleScore)
		if err != nil {
			log.Printf("Error inserting ensemble score for article %d: %v", article.ID, err)
		}

		// Parse ensemble metadata to extract variance and compute confidence
		var metaMap map[string]interface{}
		if err := json.Unmarshal([]byte(ensembleScore.Metadata), &metaMap); err != nil {
			log.Printf("Error parsing ensemble metadata for article %d: %v", article.ID, err)
		} else {
			confidence := 0.0
			if finalAgg, ok := metaMap["final_aggregation"].(map[string]interface{}); ok {
				if varianceVal, ok := finalAgg["variance"].(float64); ok {
					confidence = 1.0 - varianceVal
					if confidence < 0 {
						confidence = 0
					}
					if confidence > 1 {
						confidence = 1
					}
				}
			}
			err = db.UpdateArticleScore(c.db, article.ID, ensembleScore.Score, confidence)
		}
	}
	return nil
}

// Exported wrapper for analyzeContent to allow external packages to call it
func (c *LLMClient) AnalyzeContent(articleID int64, content string, model string, url string) (*db.LLMScore, error) {
	return c.analyzeContent(articleID, content, model) // Removed URL argument
}

// --- Progress Tracking Helpers for Async Scoring ---

// GetArticle fetches an article by ID.
func (c *LLMClient) GetArticle(articleID int64) (db.Article, error) {
	var article db.Article
	err := c.db.Get(&article, "SELECT * FROM articles WHERE id = ?", articleID)
	return article, err
}

// DeleteScores deletes all LLM scores for an article.
func (c *LLMClient) DeleteScores(articleID int64) error {
	_, err := c.db.Exec("DELETE FROM llm_scores WHERE article_id = ?", articleID)
	return err
}

// FetchScores retrieves all LLM scores for a given article ID.
func (c *LLMClient) FetchScores(articleID int64) ([]db.LLMScore, error) {
	return db.FetchLLMScores(c.db, articleID)
}

// ScoreWithModel runs scoring for a single model and stores the result.
// ScoreWithModel runs scoring for a single model and stores the result.
// ScoreWithModel analyzes an article with a specific model.
// ScoreWithModel analyzes an article with a specific model.
func (c *LLMClient) ScoreWithModel(article db.Article, modelName string) (*db.LLMScore, error) {
	// Removed URL parameter from call
	score, err := c.analyzeContent(article.ID, article.Content, modelName) // Removed URL argument
	if err != nil {
		return nil, err
	}
	_, err = c.db.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata)
		VALUES (:article_id, :model, :score, :metadata)`, score)
	return score, err
}

// StoreEnsembleScore computes and stores the ensemble score for an article.
func (c *LLMClient) StoreEnsembleScore(article db.Article) (float64, error) { // MODIFIED return type
	ensembleScore, err := c.EnsembleAnalyze(article.ID, article.Content)
	if err != nil {
		return 0.0, err // MODIFIED return
	}
	ensembleScore.ArticleID = article.ID
	_, err = c.db.NamedExec(`INSERT INTO llm_scores (article_id, model, score, metadata, created_at)
		VALUES (:article_id, :model, :score, :metadata, :created_at)`, ensembleScore)
	if err != nil {
		return 0.0, err // MODIFIED return
	}

	// Helper function to prevent panic on short strings
	// Optionally update the article's composite score/confidence
	var metaMap map[string]interface{}
	if err := json.Unmarshal([]byte(ensembleScore.Metadata), &metaMap); err == nil {
		confidence := 0.0
		if finalAgg, ok := metaMap["final_aggregation"].(map[string]interface{}); ok {
			if varianceVal, ok := finalAgg["variance"].(float64); ok {
				confidence = 1.0 - varianceVal
				if confidence < 0 {
					confidence = 0
				}
				if confidence > 1 {
					confidence = 1
				}
			}
		}
		updateErr := db.UpdateArticleScore(c.db, article.ID, ensembleScore.Score, confidence) // CAPTURE ERROR
		if updateErr != nil {
			// Return the calculated score even if the update failed, but propagate the error
			return ensembleScore.Score, updateErr // MODIFIED return
		}
	}
	return ensembleScore.Score, nil // MODIFIED return
}

// Helper function to prevent panic on short strings
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
