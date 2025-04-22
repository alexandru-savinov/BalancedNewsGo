//go:build test

package llm

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
)

func init() {
	testEnsureProjectRoot()
}

// TestNewLLMClientOpenRouterProvider checks that NewLLMClient initializes HTTPLLMService for OpenRouter
func TestNewLLMClientOpenRouterProvider(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "openrouter")
	os.Setenv("LLM_API_KEY", "test-key")
	// Use a nil DB for this test, as we only care about LLMService init
	client := NewLLMClient((*sqlx.DB)(nil))

	httpService, ok := client.llmService.(*HTTPLLMService)
	if !ok {
		t.Fatalf("Expected llmService to be *HTTPLLMService, got %T", client.llmService)
	}
	if httpService.apiKey != "test-key" {
		t.Errorf("Expected apiKey to be 'test-key', got '%s'", httpService.apiKey)
	}
}
