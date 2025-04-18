package main

import (
	"context"
	"log"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"

	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not loaded: %v", err)
	}

	// Check for generic LLM_API_KEY first, then provider-specific fallback
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		log.Println("Warning: LLM_PROVIDER not set, assuming 'openai' for API key fallback check.")
		provider = "openai"
	}
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		fallbackKeyName := strings.ToUpper(provider) + "_API_KEY"
		apiKey = os.Getenv(fallbackKeyName)
		if apiKey == "" {
			log.Fatalf("LLM_API_KEY (and fallback %s) not set. Cannot run LLM test.", fallbackKeyName)
		}
		log.Printf("Warning: Using fallback API key %s. Set LLM_API_KEY.", fallbackKeyName)
	}

	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		log.Fatal("LLM_BASE_URL not set")
	}

	// TODO: Get models to test from env var or config based on provider?
	// Example models - adjust based on the configured provider (e.g., OpenRouter might need 'openai/gpt-3.5-turbo')
	models := []string{"gpt-3.5-turbo", "gpt-4"}
	if provider == "openrouter" {
		models = []string{"openai/gpt-3.5-turbo", "openai/gpt-4"} // Example OpenRouter model names
	}
	log.Printf("Using LLM Provider: %s", provider)
	client := resty.New()

	// Create service with base URL
	svc := llm.NewHTTPLLMService(client, baseURL)

	ctx := context.Background()
	for _, model := range models {
		log.Printf("Testing model: %s", model)

		req := &llm.AnalyzeRequest{
			Content: "Say hello to the world!",
			Model:   model,
			Variant: llm.PromptVariantDefault,
		}

		_, err := svc.Analyze(ctx, req)
		if err != nil {
			log.Printf("Model %s test failed: %v", model, err)
		} else {
			log.Printf("Model %s test succeeded", model)
		}
	}
}
