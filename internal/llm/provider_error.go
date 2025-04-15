package llm

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ProviderError represents a structured error from an LLM provider
type ProviderError struct {
	StatusCode int                    `json:"status_code"`
	Message    string                 `json:"message"`
	Provider   string                 `json:"provider"`
	Type       string                 `json:"type"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	RawBody    string                 `json:"-"` // Not serialized to JSON
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s API error (%d): %s (Type: %s)",
		e.Provider, e.StatusCode, e.Message, e.Type)
}

// IsRateLimitError checks if this is a rate limit error
func (e *ProviderError) IsRateLimitError() bool {
	return e.StatusCode == 429 ||
		e.Type == "rate_limit_error" ||
		strings.Contains(strings.ToLower(e.Message), "rate limit")
}

// GetRateLimitReset returns the rate limit reset time if available
func (e *ProviderError) GetRateLimitReset() (time.Time, bool) {
	if e.Metadata == nil || e.Metadata["headers"] == nil {
		return time.Time{}, false
	}

	headers, ok := e.Metadata["headers"].(map[string]interface{})
	if !ok {
		return time.Time{}, false
	}

	resetStr, ok := headers["X-RateLimit-Reset"].(string)
	if !ok {
		return time.Time{}, false
	}

	resetMs, err := strconv.ParseInt(resetStr, 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	return time.Unix(0, resetMs*int64(time.Millisecond)), true
}

// AsMap returns the error as a map for API responses
func (e *ProviderError) AsMap() map[string]interface{} {
	return map[string]interface{}{
		"provider_name": e.Provider,
		"error_type":    e.Type,
		"message":       e.Message,
		"status_code":   e.StatusCode,
		"metadata":      e.Metadata,
	}
}
