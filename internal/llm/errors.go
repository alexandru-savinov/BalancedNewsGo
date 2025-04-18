package llm

import "errors"

var (
	ErrBothLLMKeysRateLimited = errors.New("rate limited on both keys")
	ErrLLMServiceUnavailable  = errors.New("LLM service unavailable")
	ErrRateLimited            = ErrBothLLMKeysRateLimited // Alias for compatibility with old code
)
