package llm

import (
	"context"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
)

// LLMService defines the interface for LLM analysis providers
type LLMService interface {
	ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (score float64, confidence float64, err error)
}

// HTTPLLMService implements LLMService using HTTP calls
type HTTPLLMService struct {
	client *resty.Client
	apiKey string
}

// NewHTTPLLMService creates a new HTTP-based LLM service
func NewHTTPLLMService(c *resty.Client, key string) *HTTPLLMService {
	return &HTTPLLMService{
		client: c,
		apiKey: key,
	}
}

// ScoreContent implements LLMService by making HTTP requests to score content
func (s *HTTPLLMService) ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (score float64, confidence float64, err error) {
	// Construct prompt using the variant
	prompt := pv.FormatPrompt(art.Content)

	// Make API request with context and proper authorization
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(s.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model": pv.Model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
		}).
		Post(pv.URL)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, 0, ErrLLMServiceUnavailable
		}
		return 0, 0, err
	}

	// Parse response and extract score/confidence
	score, explanation, confidence, err := parseNestedLLMJSONResponse(resp.String())
	if err != nil {
		return 0, 0, err
	}

	// Store metadata if needed
	_ = explanation // can be used for debugging or logging

	return score, confidence, nil
}
