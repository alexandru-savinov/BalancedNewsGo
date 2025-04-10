package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/api"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/metrics"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize services
	dbConn, llmClient, rssCollector := initServices()

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
	router.GET("/articles", articlesHandler(dbConn))

	// htmx endpoint for article detail
	router.GET("/article/:id", articleDetailHandler(dbConn))

	// Metrics endpoints
	router.GET("/metrics/validation", func(c *gin.Context) {
		metrics, err := metrics.GetValidationMetrics(dbConn)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, metrics)
	})

	router.GET("/metrics/feedback", func(c *gin.Context) {
		summary, err := metrics.GetFeedbackSummary(dbConn)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, summary)
	})

	router.GET("/metrics/uncertainty", func(c *gin.Context) {
		rates, err := metrics.GetUncertaintyRates(dbConn)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, rates)
	})

	router.GET("/metrics/disagreements", func(c *gin.Context) {
		disagreements, err := metrics.GetDisagreements(dbConn)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, disagreements)
	})

	router.GET("/metrics/outliers", func(c *gin.Context) {
		outliers, err := metrics.GetOutlierScores(dbConn)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, outliers)
	})

	// Start server
	log.Println("Server running on :8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initServices() (*sqlx.DB, *llm.LLMClient, *rss.Collector) {
	// Load environment variables from .env file if present
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file (this is okay if env vars are set elsewhere)")
	}

	// Initialize database
	dbConn, err := db.InitDB("news.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize LLM client
	llmClient := llm.NewLLMClient(dbConn)

	// Initialize RSS collector
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
	rssCollector := rss.NewCollector(dbConn, feedURLs, llmClient)
	rssCollector.StartScheduler()

	return dbConn, llmClient, rssCollector
}

func articlesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func articleDetailHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
