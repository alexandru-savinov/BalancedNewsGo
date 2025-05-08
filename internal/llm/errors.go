package llm

import "errors"

var (
	ErrBothLLMKeysRateLimited  = errors.New("rate limited on both keys")
	ErrLLMServiceUnavailable   = errors.New("LLM service unavailable")
	ErrRateLimited             = ErrBothLLMKeysRateLimited // Alias for compatibility with old code
	ErrAllScoresZeroConfidence = errors.New("all LLMs returned empty or zero-confidence responses")
)

// ErrAllPerspectivesInvalid indicates that despite attempting analysis across
// configured perspectives, no valid score could be obtained from any of them.
var ErrAllPerspectivesInvalid = errors.New("failed to get valid scores from any LLM perspective")
