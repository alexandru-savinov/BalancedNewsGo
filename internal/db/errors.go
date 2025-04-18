package db

import (
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
)

// Pre-defined database errors
var (
	ErrArticleNotFound  = apperrors.New("Article not found", "not_found")
	ErrScoreNotFound    = apperrors.New("Score not found", "not_found")
	ErrDuplicateArticle = apperrors.New("Article already exists", "conflict")
)

// handleDBError wraps database-specific error handling
func handleDBError(err error, context string) *apperrors.AppError {
	if appErr, ok := err.(*apperrors.AppError); ok {
		return appErr
	}

	// Map common database errors to appropriate error codes
	errMsg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errMsg, "no rows") || strings.Contains(errMsg, "not found"):
		return apperrors.New(context, "not_found")
	case strings.Contains(errMsg, "unique constraint") || strings.Contains(errMsg, "duplicate"):
		return apperrors.New(context, "conflict")
	case strings.Contains(errMsg, "constraint") || strings.Contains(errMsg, "invalid"):
		return apperrors.New(context, "invalid_input")
	default:
		return apperrors.New(context, "database_error")
	}
}
