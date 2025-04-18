package llm

import "errors"

var (
	ErrBothLLMKeysRateLimited = errors.New("rate limited on both keys")
	ErrLLMServiceUnavailable  = errors.New("LLM service unavailable")
)
