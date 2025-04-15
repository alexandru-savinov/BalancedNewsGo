package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Status   int                    `json:"-"`                  // HTTP status code (not included in response)
	Code     string                 `json:"code"`               // Error code
	Message  string                 `json:"message"`            // User-friendly error message
	Details  string                 `json:"details,omitempty"`  // Optional detailed error message
	TraceID  string                 `json:"trace_id"`           // Unique ID for tracing the error
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Additional error metadata
}

// Error codes
const (
	ErrValidation         = "validation_error"
	ErrAuthentication     = "authentication_error"
	ErrAuthorization      = "authorization_error"
	ErrNotFound           = "not_found"
	ErrRateLimit          = "rate_limit_error"
	ErrProviderError      = "provider_error"
	ErrInternal           = "internal_error"
	ErrBadRequest         = "bad_request"
	ErrServiceUnavailable = "service_unavailable"
)

// HTTP status codes
const (
	StatusOK                  = http.StatusOK
	StatusBadRequest          = http.StatusBadRequest
	StatusUnauthorized        = http.StatusUnauthorized
	StatusForbidden           = http.StatusForbidden
	StatusNotFound            = http.StatusNotFound
	StatusTooManyRequests     = http.StatusTooManyRequests
	StatusInternalServerError = http.StatusInternalServerError
	StatusServiceUnavailable  = http.StatusServiceUnavailable
)

// WarningInfo represents a non-fatal warning
type WarningInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Warning codes
const (
	WarnRateLimit   = "rate_limit_warning"
	WarnModelError  = "model_error_warning"
	WarnPartialData = "partial_data_warning"
)

// ErrorHandlingMiddleware handles errors and panics
func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a unique trace ID for this request
		traceID := uuid.New().String()
		c.Set("TraceID", traceID)

		// Add trace ID to response headers
		c.Header("X-Trace-ID", traceID)

		// Recover from panics
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				log.Printf("[PANIC] TraceID: %s | Error: %v\nStack Trace:\n%s",
					traceID, r, string(debug.Stack()))

				// Create error response
				errResp := ErrorResponse{
					Status:  StatusInternalServerError,
					Code:    ErrInternal,
					Message: "An unexpected error occurred",
					TraceID: traceID,
					Metadata: map[string]interface{}{
						"recovered": fmt.Sprintf("%v", r),
					},
				}

				// Respond with JSON
				c.AbortWithStatusJSON(errResp.Status, errResp)
			}
		}()

		// Process the request
		c.Next()

		// Check if there were any errors
		if len(c.Errors) > 0 {
			// Get the first error
			err := c.Errors[0]

			// Check if it's already an ErrorResponse
			if errResp, ok := err.Err.(*ErrorResponse); ok {
				// Use the existing error response
				errResp.TraceID = traceID
				c.AbortWithStatusJSON(errResp.Status, errResp)
				return
			}

			// Create a generic error response
			errResp := ErrorResponse{
				Status:  StatusInternalServerError,
				Code:    ErrInternal,
				Message: "An error occurred",
				Details: err.Error(),
				TraceID: traceID,
			}

			// Check for specific error types
			errMsg := strings.ToLower(err.Error())

			// Validation errors
			if strings.Contains(errMsg, "validation") || strings.Contains(errMsg, "invalid") {
				errResp.Status = StatusBadRequest
				errResp.Code = ErrValidation
				errResp.Message = "Validation failed"
			}

			// Not found errors
			if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no such") {
				errResp.Status = StatusNotFound
				errResp.Code = ErrNotFound
				errResp.Message = "Resource not found"
			}

			// Rate limit errors
			if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "too many requests") {
				errResp.Status = StatusTooManyRequests
				errResp.Code = ErrRateLimit
				errResp.Message = "Rate limit exceeded"
			}

			// Respond with JSON
			c.AbortWithStatusJSON(errResp.Status, errResp)
		}
	}
}

// RequestLoggingMiddleware logs request details
func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Get trace ID
		traceID, _ := c.Get("TraceID")
		if traceID == nil {
			traceID = uuid.New().String()
			c.Set("TraceID", traceID)
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		status := c.Writer.Status()

		// Log request details
		log.Printf("[REQUEST] TraceID: %s | %s %s | Status: %d | Latency: %s | IP: %s | User-Agent: %s",
			traceID,
			c.Request.Method,
			c.Request.URL.Path,
			status,
			latency,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
	}
}

// RateLimitMiddleware implements rate limiting
func RateLimitMiddleware(limit int, windowSeconds int) gin.HandlerFunc {
	// Simple in-memory rate limiter
	type rateLimitKey struct {
		IP     string
		Path   string
		Method string
	}

	type rateLimitInfo struct {
		Count     int
		ResetTime time.Time
	}

	// Store rate limit info
	rateLimits := make(map[rateLimitKey]*rateLimitInfo)

	return func(c *gin.Context) {
		// Create key for this request
		key := rateLimitKey{
			IP:     c.ClientIP(),
			Path:   c.Request.URL.Path,
			Method: c.Request.Method,
		}

		// Get current time
		now := time.Now()

		// Check if rate limit info exists
		info, exists := rateLimits[key]
		if !exists || now.After(info.ResetTime) {
			// Create new rate limit info
			info = &rateLimitInfo{
				Count:     0,
				ResetTime: now.Add(time.Duration(windowSeconds) * time.Second),
			}
			rateLimits[key] = info
		}

		// Increment count
		info.Count++

		// Check if rate limit exceeded
		if info.Count > limit {
			// Calculate reset time
			resetTime := info.ResetTime
			resetSeconds := int(resetTime.Sub(now).Seconds())
			if resetSeconds < 0 {
				resetSeconds = 0
			}

			// Set rate limit headers
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
			c.Header("Retry-After", fmt.Sprintf("%d", resetSeconds))

			// Create error response
			errResp := ErrorResponse{
				Status:  StatusTooManyRequests,
				Code:    ErrRateLimit,
				Message: "Rate limit exceeded",
				TraceID: c.GetString("TraceID"),
				Metadata: map[string]interface{}{
					"limit":         limit,
					"window":        windowSeconds,
					"reset_seconds": resetSeconds,
					"reset_time":    resetTime.Format(time.RFC3339),
				},
			}

			// Respond with JSON
			c.AbortWithStatusJSON(errResp.Status, errResp)
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-info.Count))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", info.ResetTime.Unix()))

		// Process request
		c.Next()
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RespondWithError sends a standardized error response
func RespondWithError(c *gin.Context, status int, code string, message string) {
	// Get trace ID
	traceID, _ := c.Get("TraceID")
	if traceID == nil {
		traceID = uuid.New().String()
	}

	// Create error response
	errResp := ErrorResponse{
		Status:  status,
		Code:    code,
		Message: message,
		TraceID: traceID.(string),
	}

	// Respond with JSON
	c.AbortWithStatusJSON(status, errResp)
}

// RespondWithErrorAndDetails sends a standardized error response with details
func RespondWithErrorAndDetails(c *gin.Context, status int, code string, message string, details string, metadata map[string]interface{}) {
	// Get trace ID
	traceID, _ := c.Get("TraceID")
	if traceID == nil {
		traceID = uuid.New().String()
	}

	// Create error response
	errResp := ErrorResponse{
		Status:   status,
		Code:     code,
		Message:  message,
		Details:  details,
		TraceID:  traceID.(string),
		Metadata: metadata,
	}

	// Respond with JSON
	c.AbortWithStatusJSON(status, errResp)
}

// RespondWithSuccess sends a standardized success response
func RespondWithSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// RespondWithSuccessAndWarnings sends a standardized success response with warnings
func RespondWithSuccessAndWarnings(c *gin.Context, data interface{}, warnings []WarningInfo) {
	response := map[string]interface{}{
		"data":     data,
		"warnings": warnings,
	}
	c.JSON(http.StatusOK, response)
}
