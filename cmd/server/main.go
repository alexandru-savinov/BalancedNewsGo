package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/api"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
	"github.com/gin-gonic/gin"
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

	// Serve static files from ./web
	router.Static("/static", "./web")

	// Health check
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Register API routes
	api.RegisterRoutes(router, dbConn, rssCollector, llmClient)

	// htmx endpoint for articles list with filters
	router.GET("/articles", func(c *gin.Context) {
		source := c.Query("source")
		leaning := c.Query("leaning")

		limit := 20

		offset := 0
		if o := c.Query("offset"); o != "" {
			if _, err := fmt.Sscanf(o, "%d", &offset); err != nil {
				offset = 0
			}
		}

		articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
		if err != nil {
			c.String(500, "Error fetching articles")

			return
		}

		html := ""
		for _, a := range articles {
			html += `<div>
				<h3>
					<a href="/article/` + strconv.FormatInt(a.ID, 10) + `"
					   hx-get="/article/` + strconv.FormatInt(a.ID, 10) + `"
					   hx-target="#articles" hx-swap="innerHTML">` + a.Title + `</a>
				</h3>
				<p>` + a.Source + ` | ` + a.PubDate.Format("2006-01-02") + `</p>
			</div>`
		}

		c.Header("Content-Type", "text/html")
		c.String(200, html)
	})

	// htmx endpoint for article detail
	router.GET("/article/:id", func(c *gin.Context) {
		id := c.Param("id")

		articleID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			c.String(400, "Invalid article ID")

			return
		}

		article, err := db.FetchArticleByID(dbConn, articleID)
		if err != nil {
			c.String(404, "Article not found")

			return
		}

		scores, _ := db.FetchLLMScores(dbConn, articleID)
		html := "<h2>" + article.Title + "</h2><p>" + article.Source + " | " +
			article.PubDate.Format("2006-01-02") + "</p><p>" + article.Content + "</p>"

		for _, s := range scores {
			html += "<p>" + s.Model + ": " + fmt.Sprintf("%.2f", s.Score) + "</p>"
		}

		c.Header("Content-Type", "text/html")
		c.String(200, html)
	})

	// Start server
	log.Println("Server running on :8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
