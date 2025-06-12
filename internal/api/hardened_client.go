package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"internal/metrics"

	"github.com/xeipuuv/gojsonschema"
)

// HardenedClient wraps the generated API client with additional features
type HardenedClient struct {
	baseClient *APIClient
	cache      sync.Map
}

// NewHardenedClient creates a new hardened client
func NewHardenedClient(cfg *Configuration) *HardenedClient {
	return &HardenedClient{
		baseClient: NewAPIClient(cfg),
	}
}

// CacheItem represents a cached item with expiration time
type CacheItem struct {
	Data       interface{}
	Expiration time.Time
}

// isExpired checks if the cached item has expired
func (ci *CacheItem) isExpired() bool {
	return time.Now().After(ci.Expiration)
}

// getFromCache retrieves an item from cache if valid
func (hc *HardenedClient) getFromCache(key string) (interface{}, bool) {
	item, ok := hc.cache.Load(key)
	if !ok {
		metrics.RecordCacheMiss()
		return nil, false
	}

	cacheItem, ok := item.(CacheItem)
	if !ok || cacheItem.isExpired() {
		metrics.RecordCacheMiss()
		return nil, false
	}

	metrics.RecordCacheHit()
	return cacheItem.Data, true
}

// setCache stores an item in cache with TTL
func (hc *HardenedClient) setCache(key string, data interface{}, ttl time.Duration) {
	hc.cache.Store(key, CacheItem{
		Data:       data,
		Expiration: time.Now().Add(ttl),
	})
}

// ValidateResponse validates response against JSON schema
func ValidateResponse(data []byte, schema string) error {
	schemaLoader := gojsonschema.NewStringLoader(schema)
	documentLoader := gojsonschema.NewBytesLoader(data)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errs string
		for _, desc := range result.Errors() {
			errs += fmt.Sprintf("- %s\n", desc)
		}
		return fmt.Errorf("response validation failed:\n%s", errs)
	}

	return nil
}

// WithRetries executes a request with retry logic
func WithRetries(ctx context.Context, fn func() (*http.Response, error), maxRetries int) (*http.Response, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := fn()
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond) // Exponential backoff
	}
	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// ExecuteRequest executes an API request with hardening features
func (hc *HardenedClient) ExecuteRequest(req *http.Request, schema string, cacheKey string, cacheTTL time.Duration) ([]byte, error) {
	start := time.Now()

	// Check cache for GET requests
	if req.Method == http.MethodGet && cacheKey != "" {
		if cached, ok := hc.getFromCache(cacheKey); ok {
			return cached.([]byte), nil
		}
	}

	// Execute request with retries
	resp, err := WithRetries(req.Context(), func() (*http.Response, error) {
		return hc.baseClient.cfg.HTTPClient.Do(req)
	}, 3)

	if err != nil {
		metrics.RecordError("request_failed")
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		metrics.RecordError("read_body_failed")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Validate response schema if provided
	if schema != "" {
		if err := ValidateResponse(body, schema); err != nil {
			metrics.RecordError("validation_failed")
			return nil, err
		}
	}

	// Cache successful GET responses
	if req.Method == http.MethodGet && cacheKey != "" && resp.StatusCode == http.StatusOK {
		hc.setCache(cacheKey, body, cacheTTL)
	}

	// Record metrics
	metrics.RecordLatency(time.Since(start))
	metrics.RecordStatus(resp.StatusCode)

	return body, nil
}
