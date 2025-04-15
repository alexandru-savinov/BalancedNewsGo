package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Error codes
const (
	ErrValidation         = middleware.ErrValidation
	ErrAuthentication     = middleware.ErrAuthentication
	ErrAuthorization      = middleware.ErrAuthorization
	ErrNotFound           = middleware.ErrNotFound
	ErrRateLimit          = middleware.ErrRateLimit
	ErrProviderError      = middleware.ErrProviderError
	ErrInternal           = middleware.ErrInternal
	ErrBadRequest         = middleware.ErrBadRequest
	ErrServiceUnavailable = middleware.ErrServiceUnavailable
)

// HTTP status codes
const (
	StatusOK                  = middleware.StatusOK
	StatusBadRequest          = middleware.StatusBadRequest
	StatusUnauthorized        = middleware.StatusUnauthorized
	StatusForbidden           = middleware.StatusForbidden
	StatusNotFound            = middleware.StatusNotFound
	StatusTooManyRequests     = middleware.StatusTooManyRequests
	StatusInternalServerError = middleware.StatusInternalServerError
	StatusServiceUnavailable  = middleware.StatusServiceUnavailable
)

// Warning codes
const (
	WarnRateLimit   = middleware.WarnRateLimit
	WarnModelError  = middleware.WarnModelError
	WarnPartialData = middleware.WarnPartialData
)

// WarningInfo represents a non-fatal warning
type WarningInfo = middleware.WarningInfo

// RespondError sends a standardized error response
func RespondError(c *gin.Context, status int, message string) {
	middleware.RespondWithError(c, status, getErrorCodeForStatus(status), message)
}

// RespondErrorWithCode sends a standardized error response with a specific error code
func RespondErrorWithCode(c *gin.Context, status int, code string, message string) {
	middleware.RespondWithError(c, status, code, message)
}

// RespondErrorWithDetails sends a standardized error response with details and metadata
func RespondErrorWithDetails(c *gin.Context, status int, code string, message string, details string, metadata map[string]interface{}) {
	middleware.RespondWithErrorAndDetails(c, status, code, message, details, metadata)
}

// RespondSuccess sends a standardized success response
func RespondSuccess(c *gin.Context, data interface{}) {
	middleware.RespondWithSuccess(c, data)
}

// RespondSuccessWithWarnings sends a standardized success response with warnings
func RespondSuccessWithWarnings(c *gin.Context, data interface{}, warnings []WarningInfo) {
	middleware.RespondWithSuccessAndWarnings(c, data, warnings)
}

// LogError logs an error with context
func LogError(context string, err error) {
	if err != nil {
		log.Printf("[ERROR] %s: %v", context, err)
	}
}

// LogWarning logs a warning with context
func LogWarning(context string, message string) {
	log.Printf("[WARNING] %s: %s", context, message)
}

// LogInfo logs an informational message with context
func LogInfo(context string, message string) {
	log.Printf("[INFO] %s: %s", context, message)
}

// LogPerformance logs performance metrics
func LogPerformance(operation string, startTime time.Time) {
	duration := time.Since(startTime)
	log.Printf("[PERF] %s: %v", operation, duration)
}

// getErrorCodeForStatus returns an appropriate error code for an HTTP status
func getErrorCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ErrBadRequest
	case http.StatusUnauthorized:
		return ErrAuthentication
	case http.StatusForbidden:
		return ErrAuthorization
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimit
	case http.StatusServiceUnavailable:
		return ErrServiceUnavailable
	default:
		return ErrInternal
	}
}

// CreateValidationError creates a detailed validation error response
func CreateValidationError(c *gin.Context, fieldErrors map[string]string) {
	// Create a formatted details string
	var details string
	for field, errMsg := range fieldErrors {
		details += fmt.Sprintf("%s: %s\n", field, errMsg)
	}

	// Create metadata
	metadata := map[string]interface{}{
		"field_errors": fieldErrors,
	}

	// Respond with error
	RespondErrorWithDetails(c, StatusBadRequest, ErrValidation, "Validation failed", details, metadata)
}

// CreateProviderError creates a detailed provider error response
func CreateProviderError(c *gin.Context, providerName string, errorType string, message string, metadata map[string]interface{}) {
	// Create combined metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["provider_name"] = providerName
	metadata["error_type"] = errorType

	// Respond with error
	RespondErrorWithDetails(c, StatusServiceUnavailable, ErrProviderError, message, "", metadata)
}

// CreateRateLimitError creates a detailed rate limit error response
func CreateRateLimitError(c *gin.Context, resetTime time.Time, limit int, window int) {
	// Calculate reset seconds
	resetSeconds := int(resetTime.Sub(time.Now()).Seconds())
	if resetSeconds < 0 {
		resetSeconds = 0
	}

	// Set headers
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	c.Header("X-RateLimit-Remaining", "0")
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
	c.Header("Retry-After", fmt.Sprintf("%d", resetSeconds))

	// Create metadata
	metadata := map[string]interface{}{
		"limit":         limit,
		"window":        window,
		"reset_seconds": resetSeconds,
		"reset_time":    resetTime.Format(time.RFC3339),
	}

	// Respond with error
	RespondErrorWithDetails(c, StatusTooManyRequests, ErrRateLimit, "Rate limit exceeded", "", metadata)
}
