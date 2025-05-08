package llm

import "errors"

var (
	ErrBothLLMKeysRateLimited  = errors.New("rate limited on both keys")
	ErrLLMServiceUnavailable   = errors.New("LLM service unavailable")
	ErrRateLimited             = ErrBothLLMKeysRateLimited // Alias for compatibility with old code
	ErrAllPerspectivesInvalid  = errors.New("all perspectives invalid")
	ErrAllScoresZeroConfidence = errors.New("all scores have zero confidence")
)

// ArticleStatus constants relevant to LLM processing failures.
// These might be moved to a more central models package in the future.
const (
	ArticleStatusFailedAllInvalid     = "failed_all_invalid"
	ArticleStatusFailedZeroConfidence = "failed_zero_confidence"
)
