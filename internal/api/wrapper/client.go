package client

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	rawclient "github.com/alexandru-savinov/BalancedNewsGo/internal/api/client"
)

// APIClient is a wrapper around the generated client with caching and configuration
type APIClient struct {
	raw   *rawclient.APIClient
	cache *sync.Map
	cfg   *Config
}

// Config holds the API client configuration
type Config struct {
	BaseURL    string
	Timeout    time.Duration
	CacheTTL   time.Duration
	MaxRetries int
	RetryDelay time.Duration
	UserAgent  string
}

// NewAPIClient creates a new wrapped API client
func NewAPIClient(baseURL string, opts ...ConfigOption) *APIClient {
	// Default configuration
	cfg := &Config{
		BaseURL:    baseURL,
		Timeout:    30 * time.Second,
		CacheTTL:   30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
		UserAgent:  "NewsBalancer-APIClient/1.0.0",
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Create raw client configuration
	rawCfg := rawclient.NewConfiguration()

	if baseURL != "" {
		parsedURL, err := url.Parse(baseURL)
		if err == nil && parsedURL.Host != "" {
			rawCfg.Host = parsedURL.Host
			rawCfg.Scheme = parsedURL.Scheme
			// The generated client might also have a BasePath field if the openapi spec included a base path like /api
			// If so, it should be set here too. For now, assuming Host and Scheme are sufficient.
			// Example: if parsedURL.Path is /api and rawCfg has BasePath, set rawCfg.BasePath = parsedURL.Path
			// Check if the generated client's configuration has a BasePath field or similar.
			// For now, we assume the paths used in client.raw.ArticlesAPI.GetArticles(ctx, rawParams)
			// are relative to the Host (e.g., "/api/articles").
		} else {
			// Fallback or error handling if URL parsing fails or host is empty
			// This case should ideally not happen with httptest.Server.URL
			// For safety, one might set default localhost values or log an error.
			// If we stick to the previous logic for a specific known URL:
			if baseURL == "http://localhost:8080" || baseURL == "https://localhost:8080" {
				rawCfg.Host = "localhost:8080"
				if strings.HasPrefix(baseURL, "https") {
					rawCfg.Scheme = "https"
				} else {
					rawCfg.Scheme = "http"
				}
			} // else, Host and Scheme might remain default from NewConfiguration()
		}
	}

	rawCfg.HTTPClient.Timeout = cfg.Timeout
	rawCfg.UserAgent = cfg.UserAgent

	// Create raw client
	rawClient := rawclient.NewAPIClient(rawCfg)

	return &APIClient{
		raw:   rawClient,
		cache: &sync.Map{},
		cfg:   cfg,
	}
}

// ConfigOption is a function that modifies the Config
type ConfigOption func(*Config)

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithCacheTTL sets the cache TTL
func WithCacheTTL(ttl time.Duration) ConfigOption {
	return func(c *Config) {
		c.CacheTTL = ttl
	}
}

// WithRetryConfig sets retry configuration
func WithRetryConfig(maxRetries int, delay time.Duration) ConfigOption {
	return func(c *Config) {
		c.MaxRetries = maxRetries
		c.RetryDelay = delay
	}
}

// WithUserAgent sets the user agent string
func WithUserAgent(userAgent string) ConfigOption {
	return func(c *Config) {
		c.UserAgent = userAgent
	}
}

// APIError represents a standardized API error
type APIError struct {
	StatusCode int         `json:"status_code"`
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("API Error [%d/%s]: %s", e.StatusCode, e.Code, e.Message)
}

// cacheEntry represents a cached value with expiration
type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// isExpired checks if the cache entry has expired
func (e *cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// getCached retrieves a value from cache if it exists and hasn't expired
func (c *APIClient) getCached(key string) (interface{}, bool) {
	if entry, exists := c.cache.Load(key); exists {
		cacheEntry, ok := entry.(*cacheEntry)
		if !ok {
			// Invalid cache entry type, clean it up
			c.cache.Delete(key)
			return nil, false
		}
		if !cacheEntry.isExpired() {
			return cacheEntry.value, true
		}
		// Clean up expired entry
		c.cache.Delete(key)
	}
	return nil, false
}

// setCached stores a value in cache with TTL
func (c *APIClient) setCached(key string, value interface{}) {
	entry := &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.cfg.CacheTTL),
	}
	c.cache.Store(key, entry)
}

// buildCacheKey creates a cache key from multiple components
func buildCacheKey(components ...interface{}) string {
	key := ""
	for i, comp := range components {
		if i > 0 {
			key += ":"
		}
		key += fmt.Sprintf("%v", comp)
	}
	return key
}
