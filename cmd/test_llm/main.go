package main

import (
	"log"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"

	"strings" // Added import

	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not loaded: %v", err)
	}

	// Check for generic LLM_API_KEY first, then provider-specific fallback
	provider := os.Getenv("LLM_PROVIDER") // Need provider for fallback key name
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

	// TODO: Get models to test from env var or config based on provider?
	// Example models - adjust based on the configured provider (e.g., OpenRouter might need 'openai/gpt-3.5-turbo')
	models := []string{"gpt-3.5-turbo", "gpt-4"}
	if provider == "openrouter" {
		models = []string{"openai/gpt-3.5-turbo", "openai/gpt-4"} // Example OpenRouter model names
	}
	log.Printf("Using LLM Provider: %s", provider)
	client := resty.New()
	// Create the service once, configured via environment variables
	service := llm.NewHTTPLLMService(client)
	log.Printf("Service configured with BaseURL: %s", service.BaseURL()) // Removed DefaultModelName() call

	for _, model := range models {
		log.Printf("Testing model: %s", model)
		// Use the single service instance to test the specific model
		// AnalyzeWithPrompt requires content, using placeholder
		_, err := service.AnalyzeWithPrompt(model, "Say hello to the world!", "Placeholder content")
		if err != nil {
			log.Printf("Model %s test failed: %v", model, err)
		} else {
			log.Printf("Model %s test succeeded", model)
		}
	}
}
