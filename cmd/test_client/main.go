package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/api/client"
)

func main() {
	// Create client configuration
	cfg := client.NewConfiguration()
	cfg.Host = "localhost:8080"
	cfg.Scheme = "http"

	// Create API client
	apiClient := client.NewAPIClient(cfg)

	// Test getting articles
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("Testing API client connection...")

	// Test 1: Get articles
	fmt.Println("1. Fetching articles...")
	params := client.ArticlesParams{
		Limit: 5,
	}
	articles, err := apiClient.ArticlesAPI.GetArticles(ctx, params)
	if err != nil {
		log.Printf("Error fetching articles: %v", err)
	} else {
		fmt.Printf("Successfully fetched %d articles\n", len(articles))
		if len(articles) > 0 {
			fmt.Printf("First article: %s\n", articles[0].Title)
		}
	}

	// Test 2: Get feed health
	fmt.Println("\n2. Checking feed health...")
	health, err := apiClient.FeedsApi.GetFeedHealth(ctx)
	if err != nil {
		log.Printf("Error checking feed health: %v", err)
	} else {
		fmt.Printf("Feed health status: %+v\n", health)
	}

	fmt.Println("\nAPI client smoke test completed!")
}
