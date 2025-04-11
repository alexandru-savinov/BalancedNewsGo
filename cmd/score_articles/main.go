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

	const batchSize = 20
	totalArticles := 0
	totalScores := 0

	articles, err := db.FetchArticles(conn, "", "", batchSize, 0)
	if err != nil {
		log.Fatalf("Failed to fetch articles: %v", err)
	}
	if len(articles) == 0 {
		log.Println("No articles found to score.")
		return
	}

	// Log all article IDs being scored
	articleIDs := make([]int64, 0, len(articles))
	found788 := false
	for _, article := range articles {
		articleIDs = append(articleIDs, article.ID)
		if article.ID == 788 {
			found788 = true
		}
	}
	log.Printf("Scoring articles with IDs: %v", articleIDs)
	if found788 {
		log.Println("Article 788 is included in this scoring run.")
	} else {
		log.Println("WARNING: Article 788 is NOT included in this scoring run.")
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

	fmt.Printf("Scoring complete.\n")
	fmt.Printf("Total articles scored: %d\n", totalArticles)
	fmt.Printf("Total scores generated: %d\n", totalScores)
}
