package client

import (
	"context"
	"encoding/json"
	"io"
	"log"
)

// FeedsApiService handles feed-related API calls
type FeedsApiService service

// TriggerRefresh triggers a refresh of all RSS feeds
func (f *FeedsApiService) TriggerRefresh(ctx context.Context) (string, error) {
	path := "/refresh"

	resp, err := f.client.makeRequest(ctx, "POST", path, nil, nil)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response StandardResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	if response.Data == nil {
		return "refresh started", nil
	}

	if result, ok := response.Data.(map[string]interface{}); ok {
		if status, exists := result["status"]; exists {
			if statusStr, ok := status.(string); ok {
				return statusStr, nil
			}
		}
	}

	return "refresh started", nil
}

// GetFeedHealth gets the health status of all RSS feeds
func (f *FeedsApiService) GetFeedHealth(ctx context.Context) (FeedHealth, error) {
	path := "/feeds/healthz"

	resp, err := f.client.makeRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

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
		return FeedHealth{}, nil
	}

	healthData, err := json.Marshal(response.Data)
	if err != nil {
		return nil, err
	}

	var health FeedHealth
	if err := json.Unmarshal(healthData, &health); err != nil {
		return nil, err
	}

	return health, nil
}
