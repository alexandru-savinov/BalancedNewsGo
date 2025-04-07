package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/api"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
)

func main() {
	// Initialize database
	dbConn, err := db.InitDB("news.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	var llmClient *llm.LLMClient

	// Initialize LLM client
	llmClient = llm.NewLLMClient(dbConn)

	// Initialize RSS collector
	feedURLs := []string{
		"https://rss.cnn.com/rss/edition.rss",
		"https://feeds.bbci.co.uk/news/rss.xml",
		"https://www.npr.org/rss/rss.php?id=1001",
	}
	rssCollector := rss.NewCollector(dbConn, feedURLs, llmClient)
	rssCollector.StartScheduler()

	// Initialize LLM client
	llmClient = llm.NewLLMClient(dbConn)

	// Initialize Gin
	router := gin.Default()

	// Health check
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Register API routes
	api.RegisterRoutes(router, dbConn, rssCollector, llmClient)

	// Start server
	log.Println("Server running on :8080")
	router.Run(":8080")
}
