package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/api"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
)

// EnhancedHTTPLLMService is an improved version of HTTPLLMService with better error handling
type EnhancedHTTPLLMService struct {
	client   *resty.Client
	apiKey   string
	provider string // e.g., "openai", "openrouter"
	baseURL  string // Base URL for the API endpoint
}

// NewEnhancedHTTPLLMService creates a new service with improved error handling
func NewEnhancedHTTPLLMService(client *resty.Client) *EnhancedHTTPLLMService {
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

	return &EnhancedHTTPLLMService{
		client:   client,
		apiKey:   apiKey,
		provider: provider,
		baseURL:  baseURL,
	}
}

// BaseURL returns the configured base URL for the LLM API.
func (s *EnhancedHTTPLLMService) BaseURL() string {
	return s.baseURL
}

// AnalyzeWithPrompt sends a prompt to the LLM API and processes the response
func (s *EnhancedHTTPLLMService) AnalyzeWithPrompt(model, prompt, content string) (*db.LLMScore, error) {
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

	// Check if it's a provider error
	if providerErr, ok := err.(*ProviderError); ok && providerErr.IsRateLimitError() {
		log.Printf("[AnalyzeWithPrompt] Primary key rate limited for model %s", model)

		// Try secondary key if available
		secondApiKey := os.Getenv("LLM_API_KEY_SECONDARY")
		if secondApiKey != "" && secondApiKey != s.apiKey {
			log.Printf("[AnalyzeWithPrompt] Attempting with secondary key for model %s", model)
			resp, err = s.callLLMAPIWithKey(model, prompt, secondApiKey)
			if err == nil && resp != nil && resp.IsSuccess() {
				log.Printf("[AnalyzeWithPrompt] Secondary key successful for model %s", model)
				return s.processLLMResponse(resp, content, model)
			}

			// Check if secondary key also hit rate limit
			if secProviderErr, ok := err.(*ProviderError); ok && secProviderErr.IsRateLimitError() {
				log.Printf("[AnalyzeWithPrompt] Both keys rate limited for model %s", model)

				// Create a combined rate limit error
				combinedErr := &ProviderError{
					StatusCode: api.StatusTooManyRequests,
					Message:    "All available API keys are rate limited",
					Provider:   s.provider,
					Type:       "rate_limit_error",
					Metadata: map[string]interface{}{
						"primary_reset":   providerErr.Metadata,
						"secondary_reset": secProviderErr.Metadata,
					},
				}

				return nil, combinedErr
			}
		} else {
			// No secondary key, return the original rate limit error
			return nil, providerErr
		}
	}

	// Return the last error encountered
	log.Printf("[AnalyzeWithPrompt] Failed after all attempts for model %s. Last error: %v", model, err)
	return nil, err
}

// AnalyzeWithModel uses the default prompt template with the specified model
func (s *EnhancedHTTPLLMService) AnalyzeWithModel(model, content string) (*db.LLMScore, error) {
	return s.AnalyzeWithPrompt(model, PromptTemplate, content)
}

// callLLMAPIWithKey makes the actual API call with enhanced error handling
func (s *EnhancedHTTPLLMService) callLLMAPIWithKey(model string, prompt string, apiKey string) (*resty.Response, error) {
	req := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+apiKey)

	body := map[string]interface{}{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  300,
		"temperature": 0.7,
	}
	req.SetBody(body)

	// Mask API Key for logging
	maskedAuthHeader := "Bearer ..."
	if len(apiKey) > 8 {
		prefix := strings.Split(apiKey, "_")[0]
		maskedAuthHeader = "Bearer " + prefix + "..." + apiKey[len(apiKey)-4:]
	}

	// Log request details
	log.Printf("[%s Request] Headers: Content-Type=%s, Authorization=%s",
		s.provider,
		req.Header.Get("Content-Type"),
		maskedAuthHeader)

	// Log body (excluding sensitive data)
	bodyBytes, _ := json.Marshal(body)
	log.Printf("[%s Request] Body: %s", s.provider, string(bodyBytes))

	// Construct the full endpoint URL
	endpointPath := "/chat/completions"
	endpointURL := s.baseURL + endpointPath
	log.Printf("[%s Request] POST URL: %s | Model in Payload: %s", s.provider, endpointURL, body["model"])

	// Make the API call
	resp, err := req.Post(endpointURL)

	// Handle network errors
	if err != nil {
		log.Printf("[%s] API request error: %v", s.provider, err)
		return nil, err
	}

	// Log response details
	log.Printf("[%s] Raw Response Status: %s", s.provider, resp.Status())
	log.Printf("[%s] Raw Response Body: %s", s.provider, resp.String())

	// Handle non-success responses
	if !resp.IsSuccess() {
		// Try to parse standard LLM error structure
		var openRouterError struct {
			Error struct {
				Message  string                 `json:"message"`
				Type     string                 `json:"type"`
				Code     string                 `json:"code"`
				Metadata map[string]interface{} `json:"metadata"`
			} `json:"error"`
		}

		if jsonErr := json.Unmarshal(resp.Body(), &openRouterError); jsonErr == nil && openRouterError.Error.Message != "" {
			// Create a structured error with metadata
			errorType := "provider_error"
			if openRouterError.Error.Type != "" {
				errorType = openRouterError.Error.Type
			}

			// Create metadata for the error
			metadata := map[string]interface{}{
				"provider_name": s.provider,
				"error_type":    errorType,
			}

			// Add rate limit headers if present
			if openRouterError.Error.Metadata != nil && openRouterError.Error.Metadata["headers"] != nil {
				metadata["headers"] = openRouterError.Error.Metadata["headers"]
			}

			// Add provider-specific error code if present
			if openRouterError.Error.Code != "" {
				metadata["provider_code"] = openRouterError.Error.Code
			}

			// Create a structured error
			providerErr := &ProviderError{
				StatusCode: resp.StatusCode(),
				Message:    openRouterError.Error.Message,
				Provider:   s.provider,
				Type:       errorType,
				Metadata:   metadata,
				RawBody:    string(resp.Body()),
			}

			log.Printf("[%s] Provider error: %v", s.provider, providerErr)
			return resp, providerErr
		}

		// Fallback generic error if parsing fails
		genericError := &ProviderError{
			StatusCode: resp.StatusCode(),
			Message:    fmt.Sprintf("%s API response not successful", s.provider),
			Provider:   s.provider,
			Type:       "unknown_error",
			RawBody:    string(resp.Body()),
		}

		log.Printf("[%s] Generic provider error: %v", s.provider, genericError)
		return resp, genericError
	}

	return resp, nil
}

// processLLMResponse processes the raw HTTP response from the LLM API call
func (s *EnhancedHTTPLLMService) processLLMResponse(resp *resty.Response, content string, model string) (*db.LLMScore, error) {
	contentResp, err := parseLLMAPIResponse(resp.Body())
	if err != nil {
		log.Printf("[processLLMResponse] Failed to parse %s response: %v\nRaw response:\n%s",
			s.provider, err, string(resp.Body()))
		return nil, fmt.Errorf("failed to parse %s response body: %w", s.provider, err)
	}

	// Extract the score from the response
	score, explanation, confidence, err := extractScoreFromContent(contentResp)
	if err != nil {
		log.Printf("[processLLMResponse] Failed to extract score from %s response: %v\nContent:\n%s",
			s.provider, err, contentResp)
		return nil, fmt.Errorf("failed to extract score from %s response: %w", s.provider, err)
	}

	// Create metadata JSON
	metadata := map[string]interface{}{
		"explanation": explanation,
		"confidence":  confidence,
		"provider":    s.provider,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("[processLLMResponse] Failed to marshal metadata: %v", err)
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create and return the LLM score
	return &db.LLMScore{
		Model:    model,
		Score:    score,
		Metadata: string(metadataJSON),
	}, nil
}
