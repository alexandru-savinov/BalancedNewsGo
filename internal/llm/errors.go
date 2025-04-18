package llm

import (
	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
)

// Standard error messages
const (
	LLMRateLimitErrorMessage = "LLM API rate limit exceeded. Please try again later."
)

// Pre-defined LLM service errors
var (
	ErrLLMServiceUnavailable  = apperrors.New("LLM service unavailable", "service_unavailable")
	ErrBothLLMKeysRateLimited = apperrors.New("Both LLM API keys are rate limited", "rate_limit")
)

// handleLLMError wraps LLM-specific error handling
func handleLLMError(err error, context string) *apperrors.AppError {
	return apperrors.HandleError(err, context)
}
