package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not loaded: %v", err)
	}

	// Get API key and base URL
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		log.Fatal("LLM_API_KEY not set")
	}

	backupKey := os.Getenv("LLM_API_KEY_SECONDARY")
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1/chat/completions"
	}

	log.Printf("Using baseURL: %s", baseURL)

	// Create resty client with debug enabled
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetDebug(true)

	// Create service instance with OpenRouter configuration
	svc := llm.NewHTTPLLMService(client, apiKey, backupKey, baseURL)

	// Create test article
	testArticle := &db.Article{
		ID:      1,
		Title:   "Test Article",
		Content: "This is a test article that should be fairly neutral in its political bias.",
	}

	// Create test prompt variant
	promptVariant := llm.PromptVariant{
		ID:       "test",
		Model:    "mistralai/mistral-small-3.1-24b-instruct",  // Switch to Mistral model
		Template: "Please analyze the political bias of the following article on a scale from -1.0 (strongly left) to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'. Article: {{content}}",
		Examples: []string{
			`{"score": 0.0, "explanation": "This article appears neutral in its political bias", "confidence": 0.9}`,
		},
	}

	// Test the service
	ctx := context.Background()
	score, confidence, err := svc.ScoreContent(ctx, promptVariant, testArticle)
	if err != nil {
		log.Fatalf("Test failed: %v", err)
	}

	log.Printf("Test succeeded - Score: %.2f, Confidence: %.2f", score, confidence)
}
