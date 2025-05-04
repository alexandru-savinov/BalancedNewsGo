package llm

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
)

// LLMService defines the interface for LLM analysis providers
type LLMService interface {
	ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (score float64, confidence float64, err error)
}

// HTTPLLMService implements LLMService using HTTP calls
type HTTPLLMService struct {
	client    *resty.Client
	apiKey    string
	backupKey string
	baseURL   string
}

// NewHTTPLLMService creates a new HTTP-based LLM service
func NewHTTPLLMService(c *resty.Client, primaryKey string, backupKey string, baseURL string) *HTTPLLMService {
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	// Ensure baseURL ends with /chat/completions
	if !strings.HasSuffix(baseURL, "/chat/completions") {
		if strings.HasSuffix(baseURL, "/") {
			baseURL = baseURL + "chat/completions"
		} else {
			baseURL = baseURL + "/chat/completions"
		}
	}
	return &HTTPLLMService{
		client:    c,
		apiKey:    primaryKey,
		backupKey: backupKey,
		baseURL:   baseURL,
	}
}

// callLLMAPIWithKey makes a direct API call to the LLM service
func (s *HTTPLLMService) callLLMAPIWithKey(modelName string, prompt string, apiKey string) (*resty.Response, error) {
	return s.client.R().
		SetAuthToken(apiKey).
		SetHeader("Content-Type", "application/json").
		SetHeader("HTTP-Referer", "https://github.com/alexandru-savinov/BalancedNewsGo").
		SetHeader("X-Title", "NewsBalancer").
		SetBody(map[string]interface{}{
			"model": modelName,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
		}).
		Post(s.baseURL)
}

// ScoreContent implements LLMService by making HTTP requests to score content
func (s *HTTPLLMService) ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (score float64, confidence float64, err error) {
	// Try primary key first
	resp, err := s.callLLMAPIWithKey(pv.Model, pv.FormatPrompt(art.Content), s.apiKey)

	// Handle rate limiting and try backup key if available
	if (err != nil && strings.Contains(err.Error(), "rate limit")) || (resp != nil && resp.StatusCode() == 429) {
		if s.backupKey != "" {
			// Try backup key if rate limited and backup key exists
			resp, err = s.callLLMAPIWithKey(pv.Model, pv.FormatPrompt(art.Content), s.backupKey)
			if (err != nil && strings.Contains(err.Error(), "rate limit")) || (resp != nil && resp.StatusCode() == 429) {
				// Both keys are rate limited for this model, try a different model
				config, err := LoadCompositeScoreConfig()
				if err != nil {
					return 0, 0, fmt.Errorf("failed to load config: %w", err)
				}

				// Find a different model to try
				for _, model := range config.Models {
					if model.ModelName != pv.Model {
						log.Printf("[INFO] Rate limited on model %s, trying alternative model %s", pv.Model, model.ModelName)
						// Try the alternative model with primary key
						resp, err = s.callLLMAPIWithKey(model.ModelName, pv.FormatPrompt(art.Content), s.apiKey)
						if err == nil && resp.StatusCode() < 400 {
							pv.Model = model.ModelName // Update the model name in the prompt variant
							break
						}
						// If still rate limited, try with backup key
						if s.backupKey != "" {
							resp, err = s.callLLMAPIWithKey(model.ModelName, pv.FormatPrompt(art.Content), s.backupKey)
							if err == nil && resp.StatusCode() < 400 {
								pv.Model = model.ModelName // Update the model name in the prompt variant
								break
							}
						}
					}
				}

				// If we still have an error after trying all models
				if err != nil || (resp != nil && resp.StatusCode() >= 400) {
					return 0, 0, fmt.Errorf("rate limit exceeded on all models and keys: %w", ErrBothLLMKeysRateLimited)
				}
			}
		} else {
			// No backup key, try a different model with primary key
			config, err := LoadCompositeScoreConfig()
			if err != nil {
				return 0, 0, fmt.Errorf("failed to load config: %w", err)
			}

			// Find a different model to try
			for _, model := range config.Models {
				if model.ModelName != pv.Model {
					log.Printf("[INFO] Rate limited on model %s, trying alternative model %s", pv.Model, model.ModelName)
					resp, err = s.callLLMAPIWithKey(model.ModelName, pv.FormatPrompt(art.Content), s.apiKey)
					if err == nil && resp.StatusCode() < 400 {
						pv.Model = model.ModelName // Update the model name in the prompt variant
						break
					}
				}
			}

			// If we still have an error after trying all models
			if err != nil || (resp != nil && resp.StatusCode() >= 400) {
				return 0, 0, fmt.Errorf("rate limit exceeded on all models: %w", ErrBothLLMKeysRateLimited)
			}
		}
	}

	if err != nil {
		return 0, 0, err
	}

	// Check for non-success status codes
	if resp.StatusCode() >= 400 {
		return 0, 0, formatHTTPError(resp)
	}

	// Log the raw response for debugging
	rawResponse := resp.String()
	log.Printf("[DEBUG][LLM] Raw response for article %d, model %s: %s", art.ID, pv.Model, rawResponse)

	// Parse the response
	score, _, confidence, err = parseNestedLLMJSONResponse(rawResponse)
	log.Printf("[DEBUG][LLM] Parsed response for article %d, model %s: score=%.4f, confidence=%.4f, err=%v",
		art.ID, pv.Model, score, confidence, err)
	return score, confidence, err
}

// formatHTTPError formats a helpful error message from HTTP responses
func formatHTTPError(resp *resty.Response) error {
	return LLMAPIError{
		Message:      "HTTP error from LLM API",
		StatusCode:   resp.StatusCode(),
		ResponseBody: resp.String(),
	}
}

// LLMAPIError represents an error from the LLM API service
type LLMAPIError struct {
	Message      string
	StatusCode   int
	ResponseBody string
}

// Error implements the error interface for LLMAPIError
func (e LLMAPIError) Error() string {
	return fmt.Sprintf("LLM API Error (status %d): %s", e.StatusCode, e.Message)
}
