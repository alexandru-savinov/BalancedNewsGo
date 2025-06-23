package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Configuration holds the client configuration
type Configuration struct {
	BasePath      string            `json:"basePath,omitempty"`
	Host          string            `json:"host,omitempty"`
	Scheme        string            `json:"scheme,omitempty"`
	DefaultHeader map[string]string `json:"defaultHeader,omitempty"`
	UserAgent     string            `json:"userAgent,omitempty"`
	HTTPClient    *http.Client
}

// NewConfiguration creates a new Configuration with default values
func NewConfiguration() *Configuration {
	cfg := &Configuration{
		BasePath:      "/api",
		Host:          "localhost:8080",
		Scheme:        "http",
		DefaultHeader: make(map[string]string),
		UserAgent:     "NewsBalancer-Go-Client/1.0.0",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	return cfg
}

// APIClient manages communication with the NewsBalancer API
type APIClient struct {
	cfg    *Configuration
	common service

	// API Services
	ArticlesAPI *ArticlesApiService
	LLMApi      *LLMApiService
	FeedsApi    *FeedsApiService
}

type service struct {
	client *APIClient
}

// NewAPIClient creates a new API client
func NewAPIClient(cfg *Configuration) *APIClient {
	if cfg == nil {
		cfg = NewConfiguration()
	}

	c := &APIClient{}
	c.cfg = cfg
	c.common.client = c

	// Initialize API services
	c.ArticlesAPI = (*ArticlesApiService)(&c.common)
	c.LLMApi = (*LLMApiService)(&c.common)
	c.FeedsApi = (*FeedsApiService)(&c.common)

	return c
}

// GetConfig returns the client configuration
func (c *APIClient) GetConfig() *Configuration {
	return c.cfg
}

// makeRequest performs the HTTP request
func (c *APIClient) makeRequest(ctx context.Context, method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	// Build URL
	u, err := url.Parse(c.cfg.Scheme + "://" + c.cfg.Host + c.cfg.BasePath + path)
	if err != nil {
		return nil, err
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	// Add default headers
	for k, v := range c.cfg.DefaultHeader {
		req.Header.Set(k, v)
	}

	// Add custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Make request
	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Error represents an API error
type APIError struct {
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("API Error [%s]: %s", e.Code, e.Message)
}

// Common response wrapper
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// checkResponse validates the HTTP response and returns an error if needed
func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Warning: failed to close response body: %v", closeErr)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read response body", resp.StatusCode)
	}

	var apiResp StandardResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if apiResp.Error != nil {
		return APIError{
			Code:    apiResp.Error.Code,
			Message: apiResp.Error.Message,
			Details: apiResp.Error.Details,
		}
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}
