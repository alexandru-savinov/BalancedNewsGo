package api

import (
	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
)

// Pre-defined error codes
const (
	ErrValidation = "validation_error"
	ErrNotFound   = "not_found"
	ErrInternal   = "internal_error"
	ErrRateLimit  = "rate_limit"
	ErrLLMService = "llm_service_error"
	ErrConflict   = "conflict_error"
)

const errInvalidArticleID = "Invalid article ID"

// Pre-defined API errors
var (
	ErrInvalidArticleID = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Invalid article ID",
	}

	ErrArticleNotFound = &apperrors.AppError{
		Code:    ErrNotFound,
		Message: "Article not found",
	}

	ErrInvalidPayload = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Invalid request payload",
	}

	ErrMissingFields = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Missing required fields",
	}

	ErrRateLimited = &apperrors.AppError{
		Code:    ErrRateLimit,
		Message: "Rate limit exceeded",
	}

	ErrLLMUnavailable = &apperrors.AppError{
		Code:    ErrLLMService,
		Message: "LLM service unavailable",
	}

	ErrDuplicateURL = &apperrors.AppError{
		Code:    ErrConflict,
		Message: "Article with this URL already exists",
	}

	ErrInvalidScore = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Score must be between -1.0 and 1.0",
	}
)

// Feedback-specific errors
var (
	ErrInvalidCategory = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Invalid feedback category. Must be one of: agree, disagree, unclear, other",
	}

	ErrInvalidFeedback = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Invalid feedback content",
	}

	ErrMissingFeedbackFields = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Missing required feedback fields",
	}

	ErrDuplicateFeedback = &apperrors.AppError{
		Code:    ErrConflict,
		Message: "Duplicate feedback submission for this article",
	}

	ErrFeedbackTooLong = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Feedback text exceeds maximum length",
	}

	ErrInvalidFeedbackSource = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Invalid feedback source identifier",
	}
)

// Bias-specific errors
var (
	ErrInvalidBiasRange = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Invalid bias score range",
	}

	ErrInvalidSortOrder = &apperrors.AppError{
		Code:    ErrValidation,
		Message: "Invalid sort order. Must be 'asc' or 'desc'",
	}

	ErrBiasDataUnavailable = &apperrors.AppError{
		Code:    ErrNotFound,
		Message: "Bias scoring data not available",
	}
)

// NewAppError creates a new application error with the given code and message
func NewAppError(code, message string) *apperrors.AppError {
	return &apperrors.AppError{
		Code:    code,
		Message: message,
	}
}

// WrapError wraps a generic error with context into an AppError
func WrapError(err error, code string, context string) *apperrors.AppError {
	if err == nil {
		return nil
	}
	message := context
	if message == "" {
		message = err.Error()
	}
	return NewAppError(code, message)
}
