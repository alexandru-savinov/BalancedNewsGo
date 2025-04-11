package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

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
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file:", err)
	}
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

	// Root welcome endpoint
	router.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	// Start server
	log.Println("Server running on :8080")

	// Start background reprocessing loop
	go startReprocessingLoop(dbConn, llmClient)

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func startReprocessingLoop(dbConn *sqlx.DB, llmClient *llm.LLMClient) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		reprocessFailedArticles(dbConn, llmClient)
		<-ticker.C
	}
}

func reprocessFailedArticles(dbConn *sqlx.DB, llmClient *llm.LLMClient) {
	log.Println("Reprocessing failed articles...")

	var articles []db.Article
	err := dbConn.Select(&articles, "SELECT * FROM articles WHERE status = 'failed' OR fail_count > 0")
	if err != nil {
		log.Printf("Error fetching failed articles: %v", err)
		return
	}

	for _, article := range articles {
		log.Printf("Reprocessing article ID %d (fail count: %d)", article.ID, article.FailCount)

		// Adaptive prompt/model switching based on fail count
		var prompt string
		var model string
		if article.FailCount >= 3 {
			model = "gpt-4"
			prompt = "Simplified prompt for difficult article"
		} else {
			model = "gpt-3.5"
			prompt = "Standard prompt"
		}

		success := processArticleWithLLM(llmClient, article, prompt, model)

		now := time.Now()
		if success {
			_, err := dbConn.Exec(`UPDATE articles SET status='processed', fail_count=0, last_attempt=?, escalated=0 WHERE id=?`, now, article.ID)
			if err != nil {
				log.Printf("Error updating article %d: %v", article.ID, err)
			} else {
				log.Printf("Article %d reprocessed successfully", article.ID)
			}
		} else {
			newFailCount := article.FailCount + 1
			escalated := 0
			status := "failed"
			if newFailCount >= 5 {
				escalated = 1
				status = "escalated"
			}
			_, err := dbConn.Exec(`UPDATE articles SET status=?, fail_count=?, last_attempt=?, escalated=? WHERE id=?`, status, newFailCount, now, escalated, article.ID)
			if err != nil {
				log.Printf("Error updating failed article %d: %v", article.ID, err)
			} else {
				log.Printf("Article %d failed again (fail count: %d)", article.ID, newFailCount)
			}
		}
	}
}

func processArticleWithLLM(llmClient *llm.LLMClient, article db.Article, prompt, model string) bool {
	// Placeholder for actual LLM call
	// Use llmClient with adaptive prompt/model
	// Return true if successful, false if failed

	// Example:
	// result, err := llmClient.ProcessArticle(article.Content, prompt, model)
	// if err != nil {
	//     return false
	// }
	// return result.Success

	// For now, simulate success/failure randomly
	return time.Now().UnixNano()%2 == 0
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

	// Initialize RSS collector from external config
	type FeedSource struct {
		Category string `json:"category"`
		URL      string `json:"url"`
	}
	type FeedSourcesConfig struct {
		Sources []FeedSource `json:"sources"`
	}

	var feedConfig FeedSourcesConfig
	feedConfigFile := "configs/feed_sources.json"
	feedConfigData, err := os.ReadFile(feedConfigFile)
	if err != nil {
		log.Fatalf("Failed to read feed sources config: %v", err)
	}
	if err := json.Unmarshal(feedConfigData, &feedConfig); err != nil {
		log.Fatalf("Failed to parse feed sources config: %v", err)
	}
	feedURLs := make([]string, 0, len(feedConfig.Sources))
	for _, src := range feedConfig.Sources {
		feedURLs = append(feedURLs, src.URL)
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
			// Fetch scores for this article
			scores, err := db.FetchLLMScores(dbConn, a.ID)
			var compositeScore float64
			var avgConfidence float64
			if err == nil && len(scores) > 0 {
				var weightedSum, sumWeights float64
				for _, s := range scores {
					var meta struct {
						Confidence float64 `json:"confidence"`
					}
					_ = json.Unmarshal([]byte(s.Metadata), &meta)
					weightedSum += s.Score * meta.Confidence
					sumWeights += meta.Confidence
				}
				if sumWeights > 0 {
					compositeScore = weightedSum / sumWeights
					avgConfidence = sumWeights / float64(len(scores))
				}
			}

			html += `<div>
				<h3>
					<a href="/article/` + strconv.FormatInt(a.ID, 10) + `"
					   hx-get="/article/` + strconv.FormatInt(a.ID, 10) + `"
					   hx-target="#articles" hx-swap="innerHTML">` + a.Title + `</a>
				</h3>
				<p>` + a.Source + ` | ` + a.PubDate.Format("2006-01-02") + `</p>
				<p>Score: ` + fmt.Sprintf("%.2f", compositeScore) + ` | Confidence: ` + fmt.Sprintf("%.0f%%", avgConfidence*100) + `</p>
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
