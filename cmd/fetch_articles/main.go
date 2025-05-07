package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/joho/godotenv"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading .env:", err)
	}
	conn, err := sqlx.Open("sqlite", "news.db")
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close DB connection: %v", err)
		}
	}()

	llmClient, err := llm.NewLLMClient(conn)
	if err != nil {
		log.Fatalf("Failed to initialize LLM Client: %v", err)
	}

	feedURLs := []string{
		// Left-leaning
		"https://www.huffpost.com/section/front-page/feed",
		"https://www.theguardian.com/world/rss",
		"http://www.msnbc.com/feeds/latest",

		// Right-leaning
		"http://feeds.foxnews.com/foxnews/latest",
		"https://www.breitbart.com/feed/",
		"https://www.washingtontimes.com/rss/headlines/news/",

		// Centrist / Mainstream
		"https://feeds.bbci.co.uk/news/rss.xml",
		"https://www.npr.org/rss/rss.php?id=1001",
		"http://feeds.reuters.com/reuters/topNews",

		// International
		"https://www.aljazeera.com/xml/rss/all.xml",
		"https://rss.dw.com/rdf/rss-en-all",

		// Alternative / Niche
		"https://reason.com/feed/",
		"https://theintercept.com/feed/?lang=en",
	}

	collector := rss.NewCollector(conn, feedURLs, llmClient)
	collector.FetchAndStore()

	log.Println("RSS fetch complete.")
}
