package main

import (
	"context"
	"fmt"
	"log"
	"time"

	apiclient "github.com/alexandru-savinov/BalancedNewsGo/internal/api/wrapper"
)

func main() {
	fmt.Println("Testing wrapped API client...")

	// Create client with custom configuration
	client := apiclient.NewAPIClient(
		"http://localhost:8080",
		apiclient.WithTimeout(15*time.Second),
		apiclient.WithCacheTTL(1*time.Minute),
		apiclient.WithRetryConfig(3, time.Second),
		apiclient.WithUserAgent("NewsBalancer-Test/1.0.0"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test 1: Get articles
	fmt.Println("\n=== Test 1: Fetching Articles ===")
	params := apiclient.ArticlesParams{
		Limit: 5,
	}
	articles, err := client.GetArticles(ctx, params)
	if err != nil {
		log.Printf("Error fetching articles: %v", err)
		if apiErr, ok := err.(apiclient.APIError); ok {
			log.Printf("API Error Details - Status: %d, Code: %s", apiErr.StatusCode, apiErr.Code)
		}
	} else {
		fmt.Printf("✓ Successfully fetched %d articles\n", len(articles))
		if len(articles) > 0 {
			fmt.Printf("  First article: %s (ID: %d)\n", articles[0].Title, articles[0].ArticleID)
		}
	}

	// Test 2: Test caching by calling again
	fmt.Println("\n=== Test 2: Testing Cache ===")
	start := time.Now()
	articles2, err := client.GetArticles(ctx, params)
	elapsed := time.Since(start)
	if err != nil {
		log.Printf("Error in cached request: %v", err)
	} else {
		fmt.Printf("✓ Cached request completed in %v\n", elapsed)
		fmt.Printf("  Articles count matches: %t\n", len(articles) == len(articles2))
	}

	// Test 3: Get specific article (if we have one)
	if len(articles) > 0 {
		fmt.Println("\n=== Test 3: Fetching Specific Article ===")
		articleID := articles[0].ArticleID
		article, err := client.GetArticle(ctx, articleID)
		if err != nil {
			log.Printf("Error fetching article %d: %v", articleID, err)
		} else {
			fmt.Printf("✓ Successfully fetched article: %s\n", article.Title)
			fmt.Printf("  Score: %.3f, Confidence: %.3f\n", article.CompositeScore, article.Confidence)
		}

		// Test 4: Get article summary
		fmt.Println("\n=== Test 4: Fetching Article Summary ===")
		summary, err := client.GetArticleSummary(ctx, articleID)
		if err != nil {
			log.Printf("Error fetching summary for article %d: %v", articleID, err)
		} else {
			fmt.Printf("✓ Successfully fetched summary (%d chars)\n", len(summary))
			if len(summary) > 100 {
				fmt.Printf("  Preview: %s...\n", summary[:100])
			} else {
				fmt.Printf("  Summary: %s\n", summary)
			}
		}

		// Test 5: Get article bias
		fmt.Println("\n=== Test 5: Fetching Article Bias ===")
		bias, err := client.GetArticleBias(ctx, articleID)
		if err != nil {
			log.Printf("Error fetching bias for article %d: %v", articleID, err)
		} else {
			fmt.Printf("✓ Successfully fetched bias analysis\n")
			fmt.Printf("  Score: %.3f, Details count: %d\n", bias.Score, len(bias.Details))
		}
	}

	// Test 6: Feed health
	fmt.Println("\n=== Test 6: Checking Feed Health ===")
	health, err := client.GetFeedHealth(ctx)
	if err != nil {
		log.Printf("Error checking feed health: %v", err)
	} else {
		fmt.Printf("✓ Successfully fetched feed health\n")
		healthyCount := 0
		totalCount := len(health)
		for feed, isHealthy := range health {
			if isHealthy {
				healthyCount++
			}
			fmt.Printf("  %s: %t\n", feed, isHealthy)
		}
		fmt.Printf("  Overall: %d/%d feeds healthy\n", healthyCount, totalCount)
	}

	// Test 7: Cache statistics
	fmt.Println("\n=== Test 7: Cache Statistics ===")
	stats := client.GetCacheStats()
	fmt.Printf("✓ Cache stats: %+v\n", stats)

	fmt.Println("\n=== Smoke Test Completed ===")
	fmt.Println("All API client wrapper tests finished!")
}
