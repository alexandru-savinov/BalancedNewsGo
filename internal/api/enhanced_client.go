package api

import (
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

// Global enhanced LLM client for use with enhanced handlers
var enhancedLLMClient *llm.EnhancedLLMClient

// SetEnhancedLLMClient sets the global enhanced LLM client
func SetEnhancedLLMClient(client *llm.EnhancedLLMClient) {
	enhancedLLMClient = client
}

// GetEnhancedLLMClient returns the global enhanced LLM client
func GetEnhancedLLMClient() *llm.EnhancedLLMClient {
	return enhancedLLMClient
}
