package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/metrics"
	"github.com/gin-gonic/gin"
)

// RespondSuccess sends a standardized success response
func RespondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    data,
	})
}

// RespondError handles application errors with standardized responses
func RespondError(c *gin.Context, err error) {
	statusCode := http.StatusInternalServerError

	// Create error detail with default values
	errorDetail := ErrorDetail{
		Code:    ErrInternal,
		Message: "Internal server error",
	}

	// Handle LLMAPIError
	var llmErr llm.LLMAPIError
	if errors.As(err, &llmErr) {
		// Track detailed error metrics
		provider := "openrouter" // Default provider, could be made dynamic if needed
		model := c.Request.URL.Query().Get("model")
		if model == "" {
			model = "unknown"
		}
		metrics.IncLLMAPIError(provider, model, string(llmErr.ErrorType), llmErr.StatusCode)

		// Map LLM error types to appropriate HTTP status codes and responses
		switch llmErr.ErrorType {
		case llm.ErrTypeRateLimit:
			statusCode = http.StatusTooManyRequests // 429
			errorDetail.Code = ErrRateLimit
			errorDetail.Message = "LLM rate limit exceeded"
			// Add retry-after header if available
			if llmErr.RetryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(llmErr.RetryAfter))
			}
		case llm.ErrTypeCredits:
			statusCode = http.StatusPaymentRequired // 402
			errorDetail.Code = ErrLLMService
			errorDetail.Message = "LLM service payment required"
		case llm.ErrTypeAuthentication:
			statusCode = http.StatusUnauthorized // 401
			errorDetail.Code = ErrLLMService
			errorDetail.Message = "LLM service authentication failed"
		case llm.ErrTypeStreaming:
			statusCode = http.StatusServiceUnavailable // 503
			errorDetail.Code = ErrLLMService
			errorDetail.Message = "LLM streaming service error"
		default:
			statusCode = http.StatusServiceUnavailable // 503
			errorDetail.Code = ErrLLMService
			errorDetail.Message = "LLM service error"
		}

		// Log the error
		log.Printf("[ERROR] LLM error (%s): %s", llmErr.ErrorType, llmErr.Message)

		// We need to use gin.H to add details which is not part of ErrorDetail
		c.JSON(statusCode, gin.H{
			"success": false,
			"error": gin.H{
				"code":    errorDetail.Code,
				"message": errorDetail.Message,
				"details": map[string]interface{}{
					"provider":        "openrouter",
					"model":           c.Request.URL.Query().Get("model"),
					"llm_status_code": llmErr.StatusCode,
					"llm_message":     llmErr.Message,
					"error_type":      string(llmErr.ErrorType),
					"retry_after":     llmErr.RetryAfter,
					"correlation_id":  c.Request.Header.Get("X-Request-ID"),
				},
				"recommended_action": getRecommendedAction(llmErr),
			},
		})
		return
	}

	// Handle regular AppError
	var appError *apperrors.AppError
	if errors.As(err, &appError) {
		errorDetail.Code = appError.Code
		errorDetail.Message = appError.Message
		statusCode = getHTTPStatus(appError.Code)
	} else if err != nil {
		errorDetail.Message = err.Error()
	}

	// Use the standard ErrorResponse structure from models.go
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error:   errorDetail,
	})
}

// LogError logs errors with context, using structured format
func LogError(c *gin.Context, err error, operation string) {
	if err == nil {
		return
	}

	// Check for LLM API errors
	var llmErr llm.LLMAPIError
	if errors.As(err, &llmErr) {
		log.Printf("[ERROR] Operation=%s Type=LLM_ERROR Status=%d ErrorType=%s Message=%s",
			operation, llmErr.StatusCode, llmErr.ErrorType, llmErr.Message)
		return
	}

	// Check for app errors
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		log.Printf("[ERROR] Operation=%s Type=APP_ERROR Code=%s Message=%s",
			operation, appErr.Code, appErr.Message)
		return
	}

	// Generic error
	log.Printf("[ERROR] Operation=%s Type=GENERIC Message=%v",
		operation, err)
}

// LogPerformance logs performance metrics in a structured format
func LogPerformance(operation string, start time.Time) {
	duration := time.Since(start)
	log.Printf("[PERF] Operation=%s Duration=%v", operation, duration)
}

// getHTTPStatus maps error codes to HTTP status codes
func getHTTPStatus(code string) int {
	switch code {
	case ErrValidation:
		return http.StatusBadRequest
	case ErrNotFound:
		return http.StatusNotFound
	case ErrRateLimit:
		return http.StatusTooManyRequests
	case ErrLLMService:
		return http.StatusServiceUnavailable
	case ErrConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// getRecommendedAction returns user-friendly action suggestions based on LLM error type
func getRecommendedAction(err llm.LLMAPIError) string {
	switch err.ErrorType {
	case llm.ErrTypeRateLimit:
		if err.RetryAfter > 0 {
			return fmt.Sprintf("Retry after %d seconds", err.RetryAfter)
		}
		return "Retry after a short delay"
	case llm.ErrTypeAuthentication:
		return "Contact administrator to update API credentials"
	case llm.ErrTypeCredits:
		return "Check API usage and billing information"
	case llm.ErrTypeStreaming:
		return "Retry with non-streaming request or check server logs"
	default:
		return "Try again later or contact support"
	}
}
