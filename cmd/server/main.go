package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	_ "github.com/alexandru-savinov/BalancedNewsGo/docs" // This will import the generated docs
	"github.com/alexandru-savinov/BalancedNewsGo/internal/api"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/metrics"
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
	// Check for health check flag
	if len(os.Args) > 1 && os.Args[1] == "--health-check" {
		// Simple health check - just exit with 0 if binary can run
		os.Exit(0)
	}

	// --- START: Explicit File Logging Setup ---
	logPath := os.Getenv("LOG_FILE_PATH")
	if logPath == "" {
		// In Docker/container environments, use /tmp for logs
		testMode := os.Getenv("TEST_MODE")
		dockerMode := os.Getenv("DOCKER")
		log.Printf("DEBUG: TEST_MODE=%s, DOCKER=%s", testMode, dockerMode)
		if testMode == "true" || dockerMode == "true" {
			logPath = "/tmp/server_app.log"
		} else {
			logPath = "server_app.log"
		}
	}
	log.Printf("DEBUG: Using log path: %s", logPath)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // #nosec G304 - logPath is from configuration, controlled input
	if err != nil {
		// In test mode or if log file creation fails, just use stdout
		if os.Getenv("TEST_MODE") == "true" {
			log.Printf("Warning: Failed to open log file %s, using stdout only: %v", logPath, err)
			logFile = os.Stdout
		} else {
			log.Fatalf("Failed to open log file: %v", err)
		}
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

	log.Printf("<<<<< APPLICATION STARTED - LOGGING TO %s >>>>>", logPath)
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
	}) // Load HTML templates (skip in test mode if templates don't exist)
	if os.Getenv("TEST_MODE") != "true" {
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
			"templates/fragments/sources.html",
		) // Load specific files
	} else {
		log.Println("TEST_MODE: Skipping template loading")
	}
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

	// Serve static files
	router.Static("/static", "./static")

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
	// Start server with graceful shutdown
	log.Printf("Server running on :%s", port)

	// Create HTTP server with security timeouts to prevent Slowloris attacks
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,  // Prevent Slowloris attacks
		ReadTimeout:       60 * time.Second,  // Maximum time to read request
		WriteTimeout:      60 * time.Second,  // Maximum time to write response
		IdleTimeout:       120 * time.Second, // Maximum time for idle connections
	}

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("ERROR: Failed to start server: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give the server 5 seconds to finish current requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func initServices() (*sqlx.DB, *llm.LLMClient, *rss.Collector, *llm.ScoreManager, *llm.ProgressManager, *api.SimpleCache) {
	// Load environment variables from .env file if present
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file (this is okay if env vars are set elsewhere)")
	}

	// Initialize database
	dbPath := os.Getenv("DB_CONNECTION")
	if dbPath == "" {
		dbPath = "news.db" // Default database path
	}
	dbConn, err := db.InitDB(dbPath)
	if err != nil {
		log.Printf("ERROR: Failed to initialize database with path '%s': %v", dbPath, err)
		// In test mode, provide more helpful error information
		if os.Getenv("TEST_MODE") == "true" {
			log.Printf("TEST_MODE: Database initialization failed. This might be due to file permissions or SQLite driver issues.")
		}
		os.Exit(1)
	}

	// Initialize LLM client
	llmClient, err := llm.NewLLMClient(dbConn)
	if err != nil {
		log.Printf("ERROR: Failed to initialize LLM Client: %v", err)
		os.Exit(1)
	}

	// Initialize RSS collector with database sources
	log.Println("Initializing RSS collector...")

	// Try to load sources from database first
	var feedURLs []string
	sources, err := db.FetchEnabledSources(dbConn)
	if err != nil {
		log.Printf("WARNING: Failed to load sources from database: %v", err)
		log.Println("Falling back to JSON config...")

		// Fallback to JSON config
		feedConfigData, err := loadFeedSourcesConfig()
		if err != nil {
			// In test mode, create minimal config if file doesn't exist
			if os.Getenv("TEST_MODE") == "true" {
				log.Printf("WARNING: Feed sources config not found in test mode, using empty config")
				feedConfigData = []byte(`{"sources": []}`)
			} else {
				log.Printf("ERROR: Failed to read feed sources config: %v", err)
				os.Exit(1)
			}
		}
		var feedConfig struct { // Use anonymous struct for local parsing
			Sources []struct {
				URL string `json:"url"`
			} `json:"sources"`
		}
		if err := json.Unmarshal(feedConfigData, &feedConfig); err != nil {
			log.Printf("ERROR: Failed to parse feed sources config: %v", err)
			os.Exit(1)
		}
		feedURLs = make([]string, 0, len(feedConfig.Sources))
		for _, src := range feedConfig.Sources {
			feedURLs = append(feedURLs, src.URL)
		}
		log.Printf("Loaded %d sources from JSON config", len(feedURLs))
	} else {
		// Successfully loaded from database
		feedURLs = make([]string, 0, len(sources))
		for _, source := range sources {
			if source.ChannelType == "rss" && source.FeedURL != "" {
				feedURLs = append(feedURLs, source.FeedURL)
			}
		}
		log.Printf("Loaded %d RSS sources from database", len(feedURLs))
	}

	// Now initialize the collector with DB, URLs, and LLM client
	collector := rss.NewCollector(dbConn, feedURLs, llmClient)

	// Load sources from database on startup to ensure fresh data
	if err := collector.LoadSourcesFromDB(); err != nil {
		log.Printf("WARNING: Failed to refresh sources from database on startup: %v", err)
	}

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
	// Use shorter cleanup interval in test environments for faster cleanup
	cleanupInterval := time.Minute
	if os.Getenv("TEST_MODE") == "true" || os.Getenv("NO_AUTO_ANALYZE") == "true" {
		cleanupInterval = time.Second * 5 // Much shorter for tests
	}
	progressManager := llm.NewProgressManager(cleanupInterval)

	// Ensure ProgressManager is stopped when server shuts down
	defer progressManager.Stop()

	scoreManager := llm.NewScoreManager(dbConn, llmAPICache, calculator, progressManager)

	// SimpleCache provides in-memory caching for API responses (articles, summaries, etc).
	simpleAPICache := api.NewSimpleCache()

	return dbConn, llmClient, collector, scoreManager, progressManager, simpleAPICache
}

// loadFeedSourcesConfig loads the feed sources configuration from multiple possible locations
func loadFeedSourcesConfig() ([]byte, error) {
	// Try multiple possible locations for the config file
	var configPath string
	var err error

	// First try: relative to current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %v", err)
	} else {
		configPath = filepath.Join(wd, "configs", "feed_sources.json")
		if _, err := os.Stat(configPath); err == nil {
			log.Printf("Found feed sources config at: %s", configPath)
			return os.ReadFile(configPath) // #nosec G304 - configPath is from application configuration, controlled input
		}
	}

	// Second try: absolute path (for Docker containers)
	configPath = "/configs/feed_sources.json"
	if _, err := os.Stat(configPath); err == nil {
		log.Printf("Found feed sources config at: %s", configPath)
		return os.ReadFile(configPath) // #nosec G304 - configPath is from application configuration, controlled input
	}

	// Third try: relative to executable
	configPath = "configs/feed_sources.json"
	if _, err := os.Stat(configPath); err == nil {
		log.Printf("Found feed sources config at: %s", configPath)
		return os.ReadFile(configPath) // #nosec G304 - configPath is from application configuration, controlled input
	}

	log.Printf("Could not find feed sources config file in any of the expected locations")
	return nil, fmt.Errorf("feed sources config file not found")
}
