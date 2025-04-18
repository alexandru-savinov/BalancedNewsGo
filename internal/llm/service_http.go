package llm

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/alexandru-savinov/BalancedNewsGo/internal/db"
    "github.com/go-resty/resty/v2"
)

// LLMService defines interface for LLM providers
type LLMService interface {
    ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (score float64, confidence float64, err error)
}

// HTTPLLMService implements LLMService using HTTP API
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

// ScoreContent implements LLMService interface
func (s *HTTPLLMService) ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (score float64, confidence float64, err error) {
    prompt := pv.GeneratePrompt(art.Content)
    
    // Make API call
    resp, err := s.client.R().
        SetContext(ctx).
        SetHeader("Authorization", "Bearer "+s.apiKey).
        SetBody(map[string]interface{}{
            "model": "gpt-3.5-turbo",
            "messages": []map[string]string{
                {"role": "user", "content": prompt},
            },
        }).
        Post("https://api.openai.com/v1/chat/completions")

    if err != nil {
        return 0, 0, fmt.Errorf("API request failed: %w", err)
    }

    // Parse response
    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }
    if err := json.Unmarshal(resp.Body(), &result); err != nil {
        return 0, 0, fmt.Errorf("failed to parse response: %w", err)
    }

    if len(result.Choices) == 0 {
        return 0, 0, fmt.Errorf("no choices in response")
    }

    // Parse nested JSON response
    var scoreResult struct {
        Score      float64 `json:"score"`
        Confidence float64 `json:"confidence"`
    }
    if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &scoreResult); err != nil {
        return 0, 0, fmt.Errorf("failed to parse score result: %w", err)
    }

    if scoreResult.Score < -1 || scoreResult.Score > 1 {
        return 0, 0, fmt.Errorf("score out of valid range (-1 to 1): %f", scoreResult.Score)
    }

    return scoreResult.Score, scoreResult.Confidence, nil
}