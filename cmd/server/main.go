package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	_ "github.com/alexandru-savinov/BalancedNewsGo/docs" // This will import the generated docs
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           NewsBalancer API
// @version         1.0
// @description     API for the NewsBalancer application which analyzes political bias in news articles using LLM models
// @termsOfService  http://swagger.io/terms/

// @contact.name   NewsBalancer Support
// @contact.url    https://github.com/alexandru-savinov/BalancedNewsGo
// @contact.email  support@newsbalancer.example

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api
// @schemes   http https

// @tag.name Articles
// @tag.description Operations related to news articles
// @tag.name Feedback
// @tag.description Operations related to user feedback
// @tag.name LLM
// @tag.description Operations related to LLM processing and scoring
// @tag.name Feeds
// @tag.description Operations related to RSS feeds
// @tag.name Admin
// @tag.description Administrative operations
// @tag.name Health
// @tag.description Health check operations
// @tag.name Scoring
// @tag.description Operations related to article scoring and manual scoring
// @tag.name Analysis
// @tag.description Operations related to article analysis and summaries

func main() {
	log.Println("<<<<< APPLICATION STARTED - BUILD/LOG TEST >>>>>") // DEBUG LOG ADDED
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file:", err)
	}
	// Initialize services
	dbConn, llmClient, rssCollector, scoreManager, progressManager, simpleCache := initServices()
	defer func() { _ = dbConn.Close() }()

	// Initialize Gin
	router := gin.Default()

	// Load HTML templates
	router.LoadHTMLGlob("web/*.html")

	// Serve static files from ./web
	router.Static("/static", "./web")

	// @Summary Health Check
	// @Description Returns the health status of the server.
	// @Tags Health
	// @Success 200 {object} map[string]string
	// @Router /healthz [get]
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Register API routes on the router instance
	// The ProgressManager handles progress tracking for LLM scoring jobs.
	// The SimpleCache provides in-memory caching for API responses.
	api.RegisterRoutes(router, dbConn, rssCollector, llmClient, scoreManager, progressManager, simpleCache)

	// @Summary Get Articles
	// @Description Fetches a list of articles with optional filters.
	// @Tags Articles
	// @Param source query string false "Filter by source"
	// @Param leaning query string false "Filter by political leaning"
	// @Param offset query int false "Pagination offset"
	// @Param limit query int false "Pagination limit"
	// @Success 200 {array} db.Article
	// @Router /articles [get]
	router.GET("/articles", articlesHandler(dbConn))

	// @Summary Get Article Details
	// @Description Fetches details of a specific article by ID.
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Success 200 {object} db.Article
	// @Router /article/{id} [get]
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

	// Add Swagger route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start server
	log.Println("Server running on :8080")

	// Start background reprocessing loop
	// go startReprocessingLoop(dbConn, llmClient) // Temporarily disabled for debugging

	if err := router.Run(":8080"); err != nil {
		log.Printf("ERROR: Failed to start server: %v", err)
		os.Exit(1)
	}
}

// Removed placeholder function processArticleWithLLM as it's replaced by llmClient.EnsembleAnalyze

func initServices() (*sqlx.DB, *llm.LLMClient, *rss.Collector, *llm.ScoreManager, *llm.ProgressManager, *api.SimpleCache) {
	// Load environment variables from .env file if present
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file (this is okay if env vars are set elsewhere)")
	}

	// Initialize database
	dbConn, err := db.InitDB("news.db")
	if err != nil {
		log.Printf("ERROR: Failed to initialize database: %v", err)
		os.Exit(1)
	}

	// Initialize LLM client
	llmClient, err := llm.NewLLMClient(dbConn)
	if err != nil {
		log.Printf("ERROR: Failed to initialize LLM Client: %v", err)
		os.Exit(1)
	}

	// Initialize RSS collector from external config
	// Load feed URLs from config file first
	feedSourcesPath := "configs/feed_sources.json"
	feedConfigData, err := os.ReadFile(feedSourcesPath)
	if err != nil {
		log.Printf("ERROR: Failed to read feed sources config '%s': %v", feedSourcesPath, err)
		os.Exit(1)
	}
	var feedConfig struct { // Use anonymous struct for local parsing
		Sources []struct {
			URL string `json:"url"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(feedConfigData, &feedConfig); err != nil {
		log.Printf("ERROR: Failed to parse feed sources config '%s': %v", feedSourcesPath, err)
		os.Exit(1)
	}
	feedURLs := make([]string, 0, len(feedConfig.Sources))
	for _, src := range feedConfig.Sources {
		feedURLs = append(feedURLs, src.URL)
	}

	// Now initialize the collector with DB, URLs, and LLM client
	collector := rss.NewCollector(dbConn, feedURLs, llmClient)
	// Assuming NewCollector does not return an error based on previous usage

	// Set HTTP client timeout if configured
	if timeoutStr := os.Getenv("LLM_HTTP_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			log.Printf("Setting LLM HTTP timeout to %v", timeout)
			llmClient.SetHTTPLLMTimeout(timeout)
		} else {
			log.Printf("Warning: Invalid LLM_HTTP_TIMEOUT value: %s. Using default.", timeoutStr)
		}
	}

	// Start cron job for fetching RSS feeds

	// Initialize ScoreManager
	llmAPICache := llm.NewCache() // This is the cache for the LLM service, distinct from the API cache.
	calculator := &llm.DefaultScoreCalculator{}
	// ProgressManager handles progress tracking and cleanup for LLM scoring jobs.
	// Using a 1-minute cleanup interval as a reasonable default.
	progressManager := llm.NewProgressManager(time.Minute)
	scoreManager := llm.NewScoreManager(dbConn, llmAPICache, calculator, progressManager)

	// SimpleCache provides in-memory caching for API responses (articles, summaries, etc).
	simpleAPICache := api.NewSimpleCache()

	return dbConn, llmClient, collector, scoreManager, progressManager, simpleAPICache
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

// TODO: Restore articleDetailHandler function definition
func articleDetailHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	// Placeholder implementation
	return func(c *gin.Context) {
		c.String(http.StatusNotImplemented, "Handler not implemented yet")
	}
}
