package llm

// This is a live integration test for OpenAI API.
// It is **skipped by default** unless the following environment variables are set:
//   - LLM_PROVIDER=openai
//   - OPENAI_API_KEY=<your-api-key>
// Optionally:
//   - OPENAI_MODEL=<model-name> (defaults to gpt-3.5-turbo)
//
// This ensures CI remains stable and does not make real API calls unless explicitly configured.
// Enable this test only for manual runs or special CI jobs with valid API keys.

import (
	"os"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestLiveOpenAIIntegration(t *testing.T) {
	if os.Getenv("LLM_PROVIDER") != "openai" {
		t.Skip("LLM_PROVIDER is not set to 'openai'; skipping live OpenAI API test")
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set; skipping live OpenAI API test")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	client := resty.New()
	service := NewOpenAILLMService(client, model, apiKey)

	resp, err := service.Analyze("Test content for live OpenAI API integration check.")
	if err != nil {
		t.Fatalf("Live OpenAI API call failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Live OpenAI API call returned nil response")
	}

	t.Logf("Live OpenAI API call succeeded. Score: %f, Metadata: %s", resp.Score, resp.Metadata)
}
