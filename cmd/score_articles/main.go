package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file (this is okay if env vars are set elsewhere)")
	}

	dbPath := "news.db"
	conn, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	llmClient := llm.NewLLMClient(conn)

	const batchSize = 10
	offset := 0
	totalArticles := 0
	totalScores := 0

	for {
		articles, err := db.FetchArticles(conn, "", "", batchSize, offset)
		if err != nil {
			log.Fatalf("Failed to fetch articles: %v", err)
		}
		if len(articles) == 0 {
			break
		}

		// Limit to first 5 articles only
		if len(articles) > 5 {
			articles = articles[:5]
		}

		for _, article := range articles {
			err := llmClient.AnalyzeAndStore(&article)
			if err != nil {
				log.Printf("Error scoring article ID %d: %v", article.ID, err)
				continue
			}
			totalScores += 3 // left, center, right models
		}

		totalArticles += len(articles)
		// After processing first 5, exit loop
		break
	}

	fmt.Printf("Scoring complete.\n")
	fmt.Printf("Total articles scored: %d\n", totalArticles)
	fmt.Printf("Total scores generated: %d\n", totalScores)
}
