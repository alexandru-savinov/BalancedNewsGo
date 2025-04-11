package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Standard response schema
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Helper to send a standardized success response
func RespondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// Helper to send a standardized error response
func RespondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// Logging helpers
func LogError(context string, err error) {
	log.Printf("[ERROR] %s: %v", context, err)
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

// Error codes
const (
	ErrInvalidArticleID = "INVALID_ARTICLE_ID"
	ErrNotFound         = "NOT_FOUND"
	ErrInternal         = "INTERNAL_ERROR"
	ErrValidation       = "VALIDATION_ERROR"
	ErrCacheMiss        = "CACHE_MISS"
	ErrBadRequest       = "BAD_REQUEST"
)
