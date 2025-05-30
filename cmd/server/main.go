package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

var legacyHTML bool

func main() {
	flag.Parse()

	// Override the legacy HTML flag to always be false
	legacyHTML = false

	log.Println("<<<<< APPLICATION STARTED - BUILD/LOG TEST >>>>>") // DEBUG LOG ADDED

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file:", err)
	}
	// Initialize services
	dbConn, llmClient, rssCollector, scoreManager, progressManager, simpleCache := initServices()
	defer func() { _ = dbConn.Close() }() // Initialize Gin
	router := gin.Default()
	// Set up template functions
	router.SetFuncMap(template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		}, "split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
	})

	// Load Editorial HTML templates - check multiple possible paths
	templatesPattern := "web/templates/*.html"
	if matches, _ := filepath.Glob(templatesPattern); len(matches) == 0 {
		// Try alternative paths if running from different directories
		if matches, _ := filepath.Glob("./web/templates/*.html"); len(matches) > 0 {
			templatesPattern = "./web/templates/*.html"
		} else if matches, _ := filepath.Glob("../web/templates/*.html"); len(matches) > 0 {
			templatesPattern = "../web/templates/*.html"
		} else {
			log.Fatalf("Could not find HTML templates. Searched: %s", templatesPattern)
		}
	}
	log.Printf("Loading templates from: %s", templatesPattern)
	router.LoadHTMLGlob(templatesPattern)

	// Serve static assets (CSS, JS, images, fonts)
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
	// Register UI routes - using Editorial template rendering
	log.Println("Using Editorial template rendering with server-side data")

	// Serve templated HTML for articles list
	router.GET("/articles", templateIndexHandler(dbConn))

	// Serve templated HTML for article detail
	router.GET("/article/:id", templateArticleHandler(dbConn))

	// Root welcome endpoint - redirect to articles
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/articles")
	})

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

	// Admin dashboard route
	router.GET("/admin", templateAdminHandler(dbConn))

	// Add Swagger route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on :%s", port)

	if err := router.Run(":" + port); err != nil {
		log.Printf("ERROR: Failed to start server: %v", err)
		os.Exit(1)
	}
}

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
