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

	// Get API key based on provider
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		log.Fatal("LLM_PROVIDER not set")
	}

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		log.Fatal("LLM_API_KEY not set")
	}

	// Create resty client with timeout
	client := resty.New()
	client.SetTimeout(30 * time.Second)

	// Create service instance
	svc := llm.NewHTTPLLMService(client, apiKey)

	// Create test article
	testArticle := &db.Article{
		ID:      1,
		Title:   "Test Article",
		Content: "This is a test article for LLM analysis.",
	}

	// Create test prompt variant
	promptVariant := llm.PromptVariant{
		ID:       "test",
		Template: "Please analyze the political bias of the following article on a scale from -1.0 (strongly left) to 1.0 (strongly right). Respond ONLY with a valid JSON object containing 'score', 'explanation', and 'confidence'.",
		Examples: []string{
			`{"score": 0.0, "explanation": "This is a neutral test article", "confidence": 0.9}`,
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
