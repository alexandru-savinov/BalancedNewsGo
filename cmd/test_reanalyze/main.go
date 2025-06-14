package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/test_reanalyze/main.go <article_id>")
		os.Exit(1)
	}

	articleID, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Fatalf("Invalid article ID: %v", err)
	}

	// Initialize DB
	dbConn, err := db.InitDB("news.db")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}()

	// Get the article
	article, err := db.FetchArticleByID(dbConn, articleID)
	if err != nil {
		log.Fatalf("Failed to fetch article: %v", err)
	}

	// Set environment variables for NewLLMClient
	if err := os.Setenv("LLM_API_KEY", "dummy-key"); err != nil {
		log.Printf("Warning: failed to set LLM_API_KEY: %v", err)
	}
	if err := os.Setenv("LLM_API_KEY_SECONDARY", "dummy-backup-key"); err != nil {
		log.Printf("Warning: failed to set LLM_API_KEY_SECONDARY: %v", err)
	}
	if err := os.Setenv("LLM_BASE_URL", "http://localhost:8090"); err != nil { // Use local mock service URL
		log.Printf("Warning: failed to set LLM_BASE_URL: %v", err)
	}
	// Create client using constructor
	llmClient, err := llm.NewLLMClient(dbConn)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}
	// Create ScoreManager and its dependencies for the reanalysis
	cache := llm.NewCache()
	calculator := &llm.DefaultScoreCalculator{}
	progressMgr := llm.NewProgressManager(5 * time.Minute) // 5 minute cleanup interval
	scoreManager := llm.NewScoreManager(dbConn, cache, calculator, progressMgr)

	// Use short timeout for testing
	llmClient.SetHTTPLLMTimeout(5 * 1000000000) // 5 seconds in nanoseconds

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Reanalyze article
	log.Printf("Starting reanalysis for article ID %d: %s", articleID, article.Title)
	err = llmClient.ReanalyzeArticle(ctx, articleID, scoreManager)
	if err != nil {
		log.Fatalf("Reanalysis failed: %v", err)
	}
	log.Printf("Reanalysis completed successfully for article ID %d", articleID)

	// Fetch the final scores to verify
	scores, err := db.FetchLLMScores(dbConn, articleID)
	if err != nil {
		log.Fatalf("Failed to fetch scores: %v", err)
	}

	log.Printf("Found %d scores for article ID %d:", len(scores), articleID)
	for i, s := range scores {
		log.Printf("  Score[%d]: Model=%s, Score=%.4f, Metadata=%s, Version=%d",
			i, s.Model, s.Score, s.Metadata, s.Version)
	}
}
