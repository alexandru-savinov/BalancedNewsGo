package api

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Standard response schema
type SuccessResponse struct {
	Success  bool          `json:"success"`
	Data     interface{}   `json:"data,omitempty"`
	Warnings []WarningInfo `json:"warnings,omitempty"`
}

type WarningInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code     int                    `json:"code"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Helper to send a standardized success response
func RespondSuccess(c *gin.Context, data interface{}) {
	jsonBytes, _ := json.Marshal(SuccessResponse{Success: true, Data: data})
	log.Printf("[RespondSuccess] JSON response: %s", string(jsonBytes))
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// Helper to send a success response with warnings
func RespondSuccessWithWarnings(c *gin.Context, data interface{}, warnings []WarningInfo) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success:  true,
		Data:     data,
		Warnings: warnings,
	})
}

// Helper to send a standardized error response
func RespondError(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    status,
			Message: message,
		},
	})
}

// Helper to send an error response with metadata
func RespondErrorWithMetadata(c *gin.Context, status int, message string, metadata map[string]interface{}) {
	c.JSON(status, ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:     status,
			Message:  message,
			Metadata: metadata,
		},
	})
}

// Helper for validation errors
func RespondValidationError(c *gin.Context, message string, validationErrors []ValidationError) {
	metadata := map[string]interface{}{
		"validation_errors": validationErrors,
	}

	RespondErrorWithMetadata(c, http.StatusBadRequest, message, metadata)
}

// ValidationError represents a single field validation error
type ValidationError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

// Helper for LLM provider errors
func RespondProviderError(c *gin.Context, status int, message string, providerName string, rawError interface{}) {
	metadata := map[string]interface{}{
		"provider_name": providerName,
		"raw":           rawError,
	}

	RespondErrorWithMetadata(c, status, message, metadata)
}

// Helper for rate limit errors
func RespondRateLimitError(c *gin.Context, message string, headers map[string]string) {
	metadata := map[string]interface{}{
		"headers": headers,
	}

	RespondErrorWithMetadata(c, http.StatusTooManyRequests, message, metadata)
}

// Helper for content moderation errors
func RespondModerationError(c *gin.Context, message string, reasons []string, flaggedInput string, providerName string) {
	metadata := map[string]interface{}{
		"reasons":       reasons,
		"flagged_input": flaggedInput,
		"provider_name": providerName,
	}

	RespondErrorWithMetadata(c, http.StatusForbidden, message, metadata)
}

// Logging helpers with request ID for correlation
func LogError(context string, err error) {
	requestID := uuid.New().String()
	log.Printf("[ERROR][%s] %s: %v", requestID, context, err)

	// In development or when detailed logging is enabled, include stack trace
	if gin.Mode() == gin.DebugMode {
		log.Printf("[ERROR][%s] Stack trace: %s", requestID, debug.Stack())
	}
}

func LogWarning(context string, message string) {
	log.Printf("[WARNING] %s: %s", context, message)
}

func LogPerformance(context string, start time.Time) {
	elapsed := time.Since(start)
	log.Printf("[PERF] %s took %s", context, elapsed)
}

// Simple in-memory cache for demonstration
type CacheItem struct {
	Value      interface{}
	Expiration int64
}

type SimpleCache struct {
	items map[string]CacheItem
}

func NewSimpleCache() *SimpleCache {
	return &SimpleCache{
		items: make(map[string]CacheItem),
	}
}

func (c *SimpleCache) Set(key string, value interface{}, duration time.Duration) {
	c.items[key] = CacheItem{
		Value:      value,
		Expiration: time.Now().Add(duration).UnixNano(),
	}
}

func (c *SimpleCache) Get(key string) (interface{}, bool) {
	item, found := c.items[key]
	if !found {
		return nil, false
	}
	if time.Now().UnixNano() > item.Expiration {
		delete(c.items, key)
		return nil, false
	}
	return item.Value, true
}

// HTTP Status code constants
const (
	StatusOK                  = http.StatusOK
	StatusBadRequest          = http.StatusBadRequest
	StatusUnauthorized        = http.StatusUnauthorized
	StatusForbidden           = http.StatusForbidden
	StatusNotFound            = http.StatusNotFound
	StatusConflict            = http.StatusConflict
	StatusTooManyRequests     = http.StatusTooManyRequests
	StatusInternalServerError = http.StatusInternalServerError
	StatusBadGateway          = http.StatusBadGateway
	StatusServiceUnavailable  = http.StatusServiceUnavailable
)

// Error code constants
const (
	// Warning codes
	WarnPartialData   = "PARTIAL_DATA"
	WarnFallbackUsed  = "FALLBACK_USED"
	WarnLowConfidence = "LOW_CONFIDENCE"
	WarnCacheMiss     = "CACHE_MISS"

	// Error categories
	ErrInvalidArticleID = "INVALID_ARTICLE_ID"
	ErrNotFound         = "NOT_FOUND"
	ErrInternal         = "INTERNAL_ERROR"
	ErrValidation       = "VALIDATION_ERROR"
	ErrBadRequest       = "BAD_REQUEST"
	ErrRateLimit        = "RATE_LIMIT"
	ErrProviderError    = "PROVIDER_ERROR"
	ErrModeration       = "CONTENT_MODERATION"
)
