package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// LLMApiService handles LLM-related API calls
type LLMApiService service

// ReanalyzeArticle triggers reanalysis of an article
func (l *LLMApiService) ReanalyzeArticle(ctx context.Context, id int64, req *ManualScoreRequest) (string, error) {
	path := fmt.Sprintf("/llm/reanalyze/%d", id)

	var body interface{}
	if req != nil {
		body = req
	}

	resp, err := l.client.makeRequest(ctx, "POST", path, body, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Warning: failed to close response body: %v", closeErr)
		}
	}()

	if err := checkResponse(resp); err != nil {
		return "", err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response StandardResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", err
	}

	if response.Data == nil {
		return "", fmt.Errorf("no reanalysis response data")
	}

	if result, ok := response.Data.(string); ok {
		return result, nil
	}

	return fmt.Sprintf("%v", response.Data), nil
}

// GetScoreProgress gets the progress of a scoring operation via SSE
// Note: This is a simplified implementation. In a real implementation,
// you would want to handle Server-Sent Events properly.
func (l *LLMApiService) GetScoreProgress(ctx context.Context, id int64) (*ProgressState, error) {
	path := fmt.Sprintf("/llm/score-progress/%d", id)

	resp, err := l.client.makeRequest(ctx, "GET", path, nil, map[string]string{
		"Accept": "application/json", // Override SSE for now
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Warning: failed to close response body: %v", closeErr)
		}
	}()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response StandardResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Data == nil {
		return nil, fmt.Errorf("no progress data returned")
	}

	progressData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, err
	}

	var progress ProgressState
	if err := json.Unmarshal(progressData, &progress); err != nil {
		return nil, err
	}

	return &progress, nil
}
