package main

import (
	"log"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not loaded: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	models := []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"}
	client := resty.New()

	for _, model := range models {
		log.Printf("Testing model: %s", model)
		service := llm.NewOpenAILLMService(client, apiKey)
		_, err := service.AnalyzeWithPrompt(model, "Say hello", "")
		if err != nil {
			log.Printf("Model %s test failed: %v", model, err)
		} else {
			log.Printf("Model %s test succeeded", model)
		}
	}
}
