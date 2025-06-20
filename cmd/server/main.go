package main

import (
	"encoding/json" // Added for json marshalling
	"html/template"
	"io" // Added for io.MultiWriter
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx" // Added for sqlx
	"github.com/joho/godotenv"

	_ "github.com/alexandru-savinov/BalancedNewsGo/docs" // This will import the generated docs
	"github.com/alexandru-savinov/BalancedNewsGo/internal/api"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/metrics" // Added metrics import
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
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
	// --- START: Explicit File Logging Setup ---
	logFile, err := os.OpenFile("d:\\\\Dev\\\\NBG\\\\server_app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	// Defer closing the log file, though for a long-running server, it might only close on exit.
	// For critical logs before this point, they might go to stdout/stderr if not captured.
	// Consider a more robust logging library for production.

	// Set standard log output to multi-writer: file and original stdout
	// This allows seeing logs in console if running interactively AND in the file.
	multiWriter := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multiWriter)

	// Set Gin's default writer to the same multi-writer
	// This ensures Gin's logs (like request logs) also go to the file and stdout.
	gin.DefaultWriter = multiWriter
	gin.DefaultErrorWriter = multiWriter // Also capture Gin's errors

	log.Println("<<<<< APPLICATION STARTED - LOGGING TO server_app.log >>>>>")
	// --- END: Explicit File Logging Setup ---

	err = godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file:", err)
	}
	// Initialize services
	dbConn, llmClient, rssCollector, scoreManager, progressManager, simpleCache := initServices()
	defer func() { _ = dbConn.Close() }() // Initialize Gin
	router := gin.Default()

	// Configure template function map
	router.SetFuncMap(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"mul":   func(a, b float64) float64 { return a * b },
		"split": func(s, sep string) []string { return strings.Split(s, sep) },
		"date":  func(t time.Time, layout string) string { return t.Format(layout) },
	}) // Load HTML templates
	// router.LoadHTMLGlob("templates/*.html") // Load top-level html files
	// router.LoadHTMLGlob("templates/fragments/*.html") // Load fragment html files
	router.LoadHTMLFiles(
		"templates/articles.html",
		"templates/article.html",
		"templates/admin.html",
		"templates/article_htmx.html",
		"templates/articles_htmx.html",
		"templates/fragments/article-list.html",
		"templates/fragments/article-items.html",
		"templates/fragments/article-detail.html",
		"templates/fragments/error.html",
		"templates/fragments/summary.html",
	) // Load specific files
	// router.LoadHTMLGlob("templates/**/*.html") // Load all html files in templates and subdirectories
	// router.LoadHTMLFiles("templates/articles.html") // Attempt to load only articles.html for diagnostics

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// @Summary Health Check
	// @Description Returns the health status of the server.
	// @Tags Health
	// @Success 200 {object} map[string]string
	// @Router /healthz [get]
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Get port for server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Template routes for web pages
	router.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/articles")
	}) // Initialize TemplateHandlers with database connection
	templateHandlers := NewTemplateHandlers(dbConn)
	router.GET("/articles", templateHandlers.TemplateIndexHandler())
	router.GET("/article/:id", templateHandlers.TemplateArticleHandler())
	router.GET("/admin", templateHandlers.TemplateAdminHandler())
	// HTMX fragment routes for dynamic loading
	router.GET("/htmx/articles", templateHandlers.TemplateArticlesFragmentHandler())
	router.GET("/htmx/articles/load-more", templateHandlers.TemplateArticlesLoadMoreHandler())
	router.GET("/htmx/article/:id", templateHandlers.TemplateArticleFragmentHandler())

	// Register API routes on the router instance
	// The ProgressManager handles progress tracking for LLM scoring jobs.
	// The SimpleCache provides in-memory caching for API responses.
	api.RegisterRoutes(router, dbConn, rssCollector, llmClient, scoreManager, progressManager, simpleCache)

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

	// Add Swagger route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Start server
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
