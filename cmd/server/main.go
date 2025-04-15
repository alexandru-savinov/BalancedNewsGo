package main

import (
	"encoding/json"
	"errors"
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
	"github.com/alexandru-savinov/BalancedNewsGo/internal/middleware"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("<<<<< APPLICATION STARTED - BUILD/LOG TEST >>>>>") // DEBUG LOG ADDED
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file:", err)
	}
	// Initialize services
	dbConn, llmClient, rssCollector := initServices()

	// Initialize Gin with custom middleware
	router := gin.New()

	// Apply custom middleware for enhanced error handling
	router.Use(gin.Recovery())
	router.Use(middleware.ErrorHandlingMiddleware())
	router.Use(middleware.RequestLoggingMiddleware())
	router.Use(middleware.CORSMiddleware())

	// Apply rate limiting to API endpoints
	apiRateLimiter := middleware.RateLimitMiddleware(100, 60) // 100 requests per minute

	// Serve static files from ./web
	router.Static("/static", "./web")

	// Health check
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Create API group with rate limiting
	apiGroup := router.Group("/api")
	apiGroup.Use(apiRateLimiter)

	// Register API routes with the rate-limited group
	api.RegisterRoutes(apiGroup, dbConn, rssCollector, llmClient)

	// htmx endpoint for articles list with filters
	router.GET("/articles", articlesHandler(dbConn))

	// htmx endpoint for article detail
	router.GET("/article/:id", articleDetailHandler(dbConn))

	// Metrics endpoints with enhanced error handling
	metricsGroup := router.Group("/metrics")

	metricsGroup.GET("/validation", func(c *gin.Context) {
		metrics, err := metrics.GetValidationMetrics(dbConn)
		if err != nil {
			api.LogError("GetValidationMetrics", err)
			api.RespondError(c, api.StatusInternalServerError, "Failed to retrieve validation metrics")
			return
		}
		api.RespondSuccess(c, metrics)
	})

	metricsGroup.GET("/feedback", func(c *gin.Context) {
		summary, err := metrics.GetFeedbackSummary(dbConn)
		if err != nil {
			api.LogError("GetFeedbackSummary", err)
			api.RespondError(c, api.StatusInternalServerError, "Failed to retrieve feedback summary")
			return
		}
		api.RespondSuccess(c, summary)
	})

	metricsGroup.GET("/uncertainty", func(c *gin.Context) {
		rates, err := metrics.GetUncertaintyRates(dbConn)
		if err != nil {
			api.LogError("GetUncertaintyRates", err)
			api.RespondError(c, api.StatusInternalServerError, "Failed to retrieve uncertainty rates")
			return
		}
		api.RespondSuccess(c, rates)
	})

	metricsGroup.GET("/disagreements", func(c *gin.Context) {
		disagreements, err := metrics.GetDisagreements(dbConn)
		if err != nil {
			api.LogError("GetDisagreements", err)
			api.RespondError(c, api.StatusInternalServerError, "Failed to retrieve disagreements data")
			return
		}
		api.RespondSuccess(c, disagreements)
	})

	metricsGroup.GET("/outliers", func(c *gin.Context) {
		outliers, err := metrics.GetOutlierScores(dbConn)
		if err != nil {
			api.LogError("GetOutlierScores", err)
			api.RespondError(c, api.StatusInternalServerError, "Failed to retrieve outlier scores")
			return
		}
		api.RespondSuccess(c, outliers)
	})

	// Root welcome endpoint
	router.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	// Start server
	log.Println("Server running on :8080")

	// Start background reprocessing loop
	// go startReprocessingLoop(dbConn, llmClient) // Temporarily disabled for debugging

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

		// Call the ensemble analysis method directly, using the configured models
		// The EnsembleAnalyze method should handle its own internal logic based on config
		_, err := llmClient.EnsembleAnalyze(article.ID, article.Content)
		// Determine success based on the error returned by EnsembleAnalyze
		success := err == nil
		if err != nil && !errors.Is(err, llm.ErrBothLLMKeysRateLimited) { // Log errors unless it's just rate limiting
			log.Printf("Error during ensemble analysis for article %d: %v", article.ID, err)
		} else if errors.Is(err, llm.ErrBothLLMKeysRateLimited) {
			log.Printf("Skipping update for article %d due to rate limiting: %v", article.ID, err)
			// Optionally continue to next article instead of updating status below
			// continue
		}

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

// Removed placeholder function processArticleWithLLM as it's replaced by llmClient.EnsembleAnalyze

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
	// Create both standard and enhanced LLM clients
	llmClient := llm.NewLLMClient(dbConn)

	// Also create the enhanced client for use with enhanced handlers
	enhancedLLMClient := llm.NewEnhancedLLMClient(dbConn)

	// Store the enhanced client in a global variable for use with enhanced handlers
	api.SetEnhancedLLMClient(enhancedLLMClient)

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
				api.LogWarning("ArticlesHandler", "Invalid offset parameter: "+o)
				offset = 0
			}
		}

		// Start performance tracking
		start := time.Now()
		articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
		if err != nil {
			api.LogError("FetchArticles", err)
			c.Header("Content-Type", "text/html")
			c.String(500, "<div class='error'>Error fetching articles. Please try again later.</div>")
			return
		}
		api.LogPerformance("FetchArticles", start)

		// Track if any scores failed to load
		var warnings []api.WarningInfo

		html := ""
		for _, a := range articles {
			// Fetch scores for this article
			scores, err := db.FetchLLMScores(dbConn, a.ID)
			var compositeScore float64
			var avgConfidence float64

			if err != nil {
				api.LogWarning("FetchLLMScores", fmt.Sprintf("Error fetching scores for article %d: %v", a.ID, err))
				warnings = append(warnings, api.WarningInfo{
					Code:    api.WarnPartialData,
					Message: fmt.Sprintf("Could not load scores for some articles"),
				})
			} else if len(scores) > 0 {
				var weightedSum, sumWeights float64
				for _, s := range scores {
					var meta struct {
						Confidence float64 `json:"confidence"`
					}
					if jsonErr := json.Unmarshal([]byte(s.Metadata), &meta); jsonErr != nil {
						api.LogWarning("ParseMetadata", fmt.Sprintf("Error parsing metadata for score %d: %v", s.ID, jsonErr))
						continue
					}
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

		// If there were warnings, log them but still return the HTML
		if len(warnings) > 0 {
			log.Printf("[WARNING] Articles handler had %d warnings", len(warnings))
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
			api.LogWarning("ArticleDetailHandler", "Invalid article ID: "+id)
			c.Header("Content-Type", "text/html")
			c.String(400, "<div class='error'>Invalid article ID</div>")
			return
		}

		article, err := db.FetchArticleByID(dbConn, articleID)
		if err != nil {
			api.LogError("FetchArticleByID", fmt.Errorf("article ID %d: %w", articleID, err))
			c.Header("Content-Type", "text/html")
			c.String(404, "<div class='error'>Article not found</div>")
			return
		}

		// Track if scores failed to load
		var warnings []api.WarningInfo

		scores, err := db.FetchLLMScores(dbConn, articleID)
		if err != nil {
			api.LogWarning("FetchLLMScores", fmt.Sprintf("Error fetching scores for article %d: %v", articleID, err))
			warnings = append(warnings, api.WarningInfo{
				Code:    api.WarnPartialData,
				Message: "Could not load scores for this article",
			})
		}

		html := "<h2>" + article.Title + "</h2><p>" + article.Source + " | " +
			article.PubDate.Format("2006-01-02") + "</p><p>" + article.Content + "</p>"

		if len(scores) > 0 {
			html += "<h3>Scores</h3>"
			for _, s := range scores {
				html += "<p>" + s.Model + ": " + fmt.Sprintf("%.2f", s.Score) + "</p>"
			}
		} else {
			html += "<p>No scores available for this article.</p>"
		}

		// If there were warnings, log them but still return the HTML
		if len(warnings) > 0 {
			log.Printf("[WARNING] Article detail handler had %d warnings for article %d", len(warnings), articleID)
		}

		c.Header("Content-Type", "text/html")
		c.String(200, html)
	}
}
