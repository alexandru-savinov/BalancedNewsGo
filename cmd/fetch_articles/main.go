package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
)

func main() {
	conn, err := sqlx.Open("sqlite", "news.db")
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer conn.Close()

	llmClient := llm.NewLLMClient(conn)

	feedURLs := []string{
		"https://feeds.bbci.co.uk/news/rss.xml",
		"https://www.npr.org/rss/rss.php?id=1001",
		// Add more feeds as needed
	}

	collector := rss.NewCollector(conn, feedURLs, llmClient)
	collector.FetchAndStore()

	log.Println("RSS fetch complete.")
}
