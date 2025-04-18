package api

import (
	"log"
	"net/http"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/gin-gonic/gin"
)

// RespondSuccess sends a standardized success response
func RespondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// RespondError handles application errors with standardized responses
func RespondError(c *gin.Context, err *apperrors.AppError) {
	if err == nil {
		err = NewAppError(ErrInternal, "Unknown error occurred")
	}

	// Map error codes to HTTP status codes
	status := getHTTPStatus(err.Code)
	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    err.Code,
			"message": err.Message,
		},
	})
}

// LogError logs errors with context, using structured format
func LogError(operation string, err error) {
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			log.Printf("[ERROR] Operation=%s Code=%s Message=%s",
				operation, appErr.Code, appErr.Message)
		} else {
			log.Printf("[ERROR] Operation=%s Message=%v",
				operation, err)
		}
	}
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
