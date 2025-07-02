package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not loaded: %v", err)
	}

	// Get API key
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ ERROR: LLM_API_KEY not set")
		fmt.Println("Please set your OpenRouter API key in the .env file")
		fmt.Println("Get a new API key from: https://openrouter.ai/")
		os.Exit(1)
	}

	// Mask the API key for display
	maskedKey := maskKey(apiKey)
	fmt.Printf("🔑 Testing API key: %s\n", maskedKey)

	// Test the API key
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	client := resty.New()
	client.SetTimeout(30 * time.Second)

	// Make a simple test request
	resp, err := client.R().
		SetAuthToken(apiKey).
		SetHeader("Content-Type", "application/json").
		SetHeader("HTTP-Referer", "https://github.com/alexandru-savinov/BalancedNewsGo").
		SetHeader("X-Title", "NewsBalancer").
		SetBody(map[string]interface{}{
			"model": "openai/gpt-4.1-nano",
			"messages": []map[string]string{
				{"role": "user", "content": "Test message"},
			},
			"max_tokens": 1,
		}).
		Post(baseURL + "/chat/completions")

	if err != nil {
		fmt.Printf("❌ ERROR: Network error: %v\n", err)
		fmt.Println("Check your internet connection and try again")
		os.Exit(1)
	}

	// Check response status
	switch resp.StatusCode() {
	case 200, 201:
		fmt.Println("✅ SUCCESS: API key is valid and working!")
		fmt.Println("Your reanalysis functionality should work properly.")
	case 401:
		fmt.Println("❌ ERROR: Invalid API key (HTTP 401)")
		fmt.Println("Your API key is invalid or has been disabled.")
		fmt.Println("Please get a new API key from: https://openrouter.ai/")
		fmt.Println("Update the LLM_API_KEY in your .env file")
	case 402:
		fmt.Println("❌ ERROR: Payment required (HTTP 402)")
		fmt.Println("Your OpenRouter account has insufficient credits.")
		fmt.Println("Please add credits to your account at: https://openrouter.ai/")
	case 429:
		fmt.Println("⚠️  WARNING: Rate limited (HTTP 429)")
		fmt.Println("You're making too many requests. Try again in a few minutes.")
	default:
		fmt.Printf("❌ ERROR: Unexpected response (HTTP %d)\n", resp.StatusCode())
		fmt.Printf("Response: %s\n", resp.String())
		fmt.Println("Please check the OpenRouter service status")
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
		os.Exit(1)
	}
}

// maskKey masks an API key for logging purposes
func maskKey(key string) string {
	if key == "" {
		return "<empty>"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
