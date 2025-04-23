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

	swaggerFiles "github.com/swaggo/files"        // Correct swagger embed files import
	ginSwagger "github.com/swaggo/gin-swagger"    // Swagger UI
	_ "github.com/swaggo/swag/example/basic/docs" // Swaggo docs generation
)

// @title           NewsBalancer API
// @version         1.0
// @description     API for the NewsBalancer application which analyzes political bias in news articles
// @termsOfService  http://swagger.io/terms/

// @contact.name   NewsBalancer Support
// @contact.url    https://github.com/alexandru-savinov/BalancedNewsGo
// @contact.email  support@newsbalancer.example

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api
// @schemes   http

func main() {
	log.Println("<<<<< APPLICATION STARTED - BUILD/LOG TEST >>>>>") // DEBUG LOG ADDED
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env file:", err)
	}
	// Initialize services
	dbConn, llmClient, rssCollector, scoreManager := initServices()

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
	api.RegisterRoutes(router, dbConn, rssCollector, llmClient, scoreManager)

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
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Removed placeholder function processArticleWithLLM as it's replaced by llmClient.EnsembleAnalyze

func initServices() (*sqlx.DB, *llm.LLMClient, *rss.Collector, *llm.ScoreManager) {
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

	// Initialize ScoreManager
	cache := llm.NewCache()
	calculator := &llm.DefaultScoreCalculator{
		Config: &llm.CompositeScoreConfig{
			MinScore: -1.0,
			MaxScore: 1.0,
		},
	}
	progressMgr := llm.NewProgressManager(10 * time.Minute) // Clean up progress data every 10 minutes
	scoreManager := llm.NewScoreManager(dbConn, cache, calculator, progressMgr)

	return dbConn, llmClient, rssCollector, scoreManager
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
		composite, confidence, _ := llm.ComputeCompositeScoreWithConfidence(scores)

		// Calculate bias label and confidence colors
		var biasLabel string
		if composite < -0.5 {
			biasLabel = "Left"
		} else if composite < -0.1 {
			biasLabel = "Slightly Left"
		} else if composite > 0.5 {
			biasLabel = "Right"
		} else if composite > 0.1 {
			biasLabel = "Slightly Right"
		} else {
			biasLabel = "Center"
		}

		var confidenceColor string
		if confidence >= 0.8 {
			confidenceColor = "var(--confidence-high)"
		} else if confidence >= 0.5 {
			confidenceColor = "var(--confidence-medium)"
		} else {
			confidenceColor = "var(--confidence-low)"
		}

		// Calculate model indicators for the bias slider
		var modelIndicators []map[string]interface{}
		for _, s := range scores {
			var meta struct {
				Confidence float64 `json:"confidence"`
			}
			_ = json.Unmarshal([]byte(s.Metadata), &meta)

			modelIndicators = append(modelIndicators, map[string]interface{}{
				"Position": ((s.Score + 1) / 2) * 100,
				"Title":    fmt.Sprintf("%s: %.2f (Confidence: %.0f%%)", s.Model, s.Score, meta.Confidence*100),
			})
		}

		// Render HTML template
		c.HTML(200, "article.html", gin.H{
			"ID":                article.ID,
			"Title":             article.Title,
			"Content":           article.Content,
			"Source":            article.Source,
			"PubDate":           article.PubDate.Format("2006-01-02 15:04:05"),
			"CreatedAt":         article.CreatedAt.Format("2006-01-02 15:04:05"),
			"CompositeScore":    fmt.Sprintf("%.2f", composite),
			"BiasLabel":         biasLabel,
			"ConfidencePercent": fmt.Sprintf("%.0f", confidence*100),
			"ConfidenceColor":   confidenceColor,
			"SliderPosition":    ((composite + 1) / 2) * 100,
			"ModelIndicators":   modelIndicators,
			"Explanation":       "Analysis based on multiple machine learning models evaluating political bias.",
		})
	}
}
