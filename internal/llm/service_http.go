package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/metrics"
	"github.com/go-resty/resty/v2"
)

// LLMService defines the interface for LLM analysis providers
type LLMService interface {
	ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (score float64, confidence float64, err error)
}

// OpenRouterErrorType represents specific error types from OpenRouter
type OpenRouterErrorType string

const (
	// OpenRouter specific error types
	ErrTypeRateLimit      OpenRouterErrorType = "rate_limit"
	ErrTypeAuthentication OpenRouterErrorType = "authentication"
	ErrTypeCredits        OpenRouterErrorType = "credits"
	ErrTypeStreaming      OpenRouterErrorType = "streaming"
	ErrTypeUnknown        OpenRouterErrorType = "unknown"
)

// LLMAPIError wraps OpenRouter errors with additional context
type LLMAPIError struct {
	Message      string
	StatusCode   int
	ResponseBody string
	ErrorType    OpenRouterErrorType
	RetryAfter   int // For rate limit errors
}

// Error implements the error interface for LLMAPIError
func (e LLMAPIError) Error() string {
	errType := string(e.ErrorType)
	if errType == "" {
		errType = "unknown"
	}
	return fmt.Sprintf("LLM API Error (%s): %s (status %d)", errType, e.Message, e.StatusCode)
}

// Create predefined app errors for consistent handling
var (
	ErrLLMAuthenticationFailed = apperrors.New("LLM service authentication failed", "llm_authentication")
	ErrLLMCreditsExhausted     = apperrors.New("LLM service credits exhausted", "llm_credits")
	ErrLLMStreamingFailed      = apperrors.New("LLM streaming response failed", "llm_streaming")
)

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
			baseURL += "chat/completions"
		} else {
			baseURL += "/chat/completions"
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
					return 0, 0, LLMAPIError{
						Message:    "LLM rate limit exceeded: Rate limit exceeded",
						StatusCode: 429,
						ErrorType:  ErrTypeRateLimit,
						RetryAfter: 30,
					}
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
				return 0, 0, LLMAPIError{
					Message:    "LLM rate limit exceeded: Rate limit exceeded",
					StatusCode: 429,
					ErrorType:  ErrTypeRateLimit,
					RetryAfter: 30,
				}
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

// formatHTTPError converts HTTP responses to structured LLMAPIError objects
func formatHTTPError(resp *resty.Response) error {
	// Initialize default values
	errorType := ErrTypeUnknown
	retryAfter := 0
	var message string

	// Try to parse the error response
	var openRouterError struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(resp.String()), &openRouterError); err == nil && openRouterError.Error.Message != "" {
		message = openRouterError.Error.Message
	} else {
		// Use status text if can't parse JSON
		message = resp.Status()
	}

	// Identify error type and additional metadata
	switch resp.StatusCode() {
	case 429:
		errorType = ErrTypeRateLimit
		if retryHeader := resp.Header().Get("Retry-After"); retryHeader != "" {
			if parsed, err := strconv.Atoi(retryHeader); err == nil {
				retryAfter = parsed
			} else {
				log.Printf("Warning: Failed to parse Retry-After header '%s': %v", retryHeader, err)
			}
		}
		metrics.IncLLMFailure("openrouter", "", "rate_limit")
	case 402:
		errorType = ErrTypeCredits
		metrics.IncLLMFailure("openrouter", "", "credits")
	case 401:
		errorType = ErrTypeAuthentication
		metrics.IncLLMFailure("openrouter", "", "authentication")
	case 503:
		lowerMsg := strings.ToLower(message)
		if strings.Contains(lowerMsg, "stream") || strings.Contains(lowerMsg, "sse") {
			errorType = ErrTypeStreaming
			metrics.IncLLMFailure("openrouter", "", "streaming")
		} else {
			metrics.IncLLMFailure("openrouter", "", "other")
		}
	default:
		lowerMsg := strings.ToLower(message)
		if strings.Contains(lowerMsg, "stream") || strings.Contains(lowerMsg, "sse") {
			errorType = ErrTypeStreaming
			metrics.IncLLMFailure("openrouter", "", "streaming")
		} else {
			metrics.IncLLMFailure("openrouter", "", "other")
		}
	}

	// Special case: empty response body and status 500
	if resp.StatusCode() == 500 && strings.TrimSpace(resp.String()) == "" {
		return LLMAPIError{
			Message:      "LLM service error: 500 Internal Server Error",
			StatusCode:   resp.StatusCode(),
			ResponseBody: sanitizeResponse(resp.String()),
			ErrorType:    errorType,
			RetryAfter:   retryAfter,
		}
	}

	// Create appropriate error message prefix based on error type
	var errorPrefix string
	switch errorType {
	case ErrTypeRateLimit:
		errorPrefix = "LLM rate limit exceeded: "
	case ErrTypeCredits:
		errorPrefix = "LLM credits exhausted: "
	case ErrTypeAuthentication:
		errorPrefix = "LLM authentication failed: "
	case ErrTypeStreaming:
		errorPrefix = "LLM streaming failed: "
	default:
		errorPrefix = "LLM service error: "
	}

	return LLMAPIError{
		Message:      errorPrefix + message,
		StatusCode:   resp.StatusCode(),
		ResponseBody: sanitizeResponse(resp.String()),
		ErrorType:    errorType,
		RetryAfter:   retryAfter,
	}
}

// Sanitize response to remove sensitive info
func sanitizeResponse(response string) string {
	// Simple sanitization - remove potential API keys
	sanitized := regexp.MustCompile(`(sk-|or-)[a-zA-Z0-9]{20,}`).ReplaceAllString(response, "[REDACTED]")
	return sanitized
}
