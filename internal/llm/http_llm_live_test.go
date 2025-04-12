package llm

// This is a live integration test for a configured HTTP LLM API (e.g., OpenAI, OpenRouter).
// It is **skipped by default** unless the following environment variables are set:
//   - LLM_API_KEY=<your-api-key>
//   - LLM_PROVIDER=<provider-name> (e.g., "openai", "openrouter")
//   - LLM_BASE_URL=<api-base-url> (Optional, defaults based on provider)
//   - LLM_DEFAULT_MODEL=<model-name> (Optional, defaults based on provider)
//
// This ensures CI remains stable and does not make real API calls unless explicitly configured.
// Enable this test only for manual runs or special CI jobs with valid API keys and configuration.

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestLiveHTTPLLMIntegration(t *testing.T) {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		t.Skip("LLM_PROVIDER not set; skipping live LLM API test")
	}
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		// Try fallback for backward compatibility during transition
		fallbackKeyName := strings.ToUpper(provider) + "_API_KEY"
		apiKey = os.Getenv(fallbackKeyName)
		if apiKey == "" {
			t.Skipf("LLM_API_KEY (and fallback %s) not set; skipping live LLM API test for provider %s", fallbackKeyName, provider)
		}
	}
	// We don't need to check/set model here, NewHTTPLLMService handles defaults
	// Base URL and Default Model are handled by NewHTTPLLMService based on env vars or defaults

	client := resty.New()
	service := NewHTTPLLMService(client) // Use the generic service

	// Analyze uses the default model configured in the service
	testContent := fmt.Sprintf("Test content for live %s API integration check.", provider)
	resp, err := service.Analyze(testContent)
	if err != nil {
		t.Fatalf("Live %s API call failed: %v", provider, err)
	}

	if resp == nil {
		t.Fatalf("Live %s API call returned nil response", provider)
	}

	t.Logf("Live %s API call succeeded using model %s. Score: %f, Metadata: %s", provider, service.defaultModel, resp.Score, resp.Metadata)
}
