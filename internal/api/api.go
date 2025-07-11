package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Define constants for commonly used strings
const (
	ModelEnsemble   = "ensemble"
	ModelSummarizer = "summarizer"
	SortAsc         = "asc"
	SortDesc        = "desc"
)

var (
	articlesCache     = NewSimpleCache()
	articlesCacheLock sync.RWMutex
)

// Progress tracking vars
// Removed local progress tracking functions - now using ScoreManager's ProgressManager

// RegisterRoutes registers all API routes on the provided router
func RegisterRoutes(
	router *gin.Engine,
	dbConn *sqlx.DB,
	rssCollector rss.CollectorInterface,
	llmClient *llm.LLMClient,
	scoreManager *llm.ScoreManager,
	progressManager *llm.ProgressManager,
	cache *SimpleCache,
) {
	// Articles endpoints
	// @Summary Get all articles
	// @Description Get a list of all articles with optional filtering
	// @Tags Articles
	// @Accept json
	// @Produce json
	// @Param source query string false "Filter by news source"
	// @Param offset query integer false "Pagination offset"
	// @Param limit query integer false "Number of items per page"
	// @Success 200 {array} api.Article
	// @Failure 500 {object} ErrorResponse
	// @Router /api/articles [get]
	router.GET("/api/articles", SafeHandler(getArticlesHandler(dbConn)))

	// @Summary Get article by ID
	// @Description Get detailed information about a specific article
	// @Tags Articles
	// @Accept json
	// @Produce json
	// @Param id path integer true "Article ID"
	// @Success 200 {object} api.Article
	// @Failure 404 {object} ErrorResponse
	// @Router /api/articles/{id} [get]
	router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(dbConn)))

	// @Summary Create article
	// @Description Create a new article
	// @Tags Articles
	// @Accept json
	// @Produce json
	// @Param article body models.CreateArticleRequest true "Article object"
	// @Success 200 {object} StandardResponse{data=api.Article}
	// @Failure 400 {object} ErrorResponse
	// @Router /api/articles [post]
	router.POST("/api/articles", SafeHandler(createArticleHandler(dbConn)))

	// Feed management
	// @Summary Refresh feeds
	// @Description Trigger a refresh of all RSS feeds
	// @Tags Feeds
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/refresh [post]
	// @ID triggerRssRefresh
	router.POST("/api/refresh", SafeHandler(refreshHandler(rssCollector)))

	// LLM Analysis
	// @Summary      Re-analyze article via LLM
	// @Param        id    path     int                   true  "Article ID"
	// @Param        score body    ManualScoreRequest    false "Optional manual score override"
	// @Success      202   {object} StandardResponse{data=string}  "Reanalysis queued"
	// @Failure      400   {object} StandardResponse
	// @Failure      401   {object} StandardResponse
	// @Failure      402   {object} StandardResponse
	// @Failure      429   {object} StandardResponse
	// @Failure      503   {object} StandardResponse
	// @Router       /api/llm/reanalyze/{id} [post]
	// @ID reanalyzeArticle
	router.POST("/api/llm/reanalyze/:id", SafeHandler(reanalyzeHandler(llmClient, dbConn, scoreManager)))

	// Scoring
	// @Summary Add manual score
	// @Description Add a manual bias score for an article
	// @Tags Scoring
	// @Accept json
	// @Produce json
	// @Param id path integer true "Article ID"
	// @Param score body api.ManualScoreRequest true "Score information"
	// @Success 200 {object} StandardResponse
	// @Failure 400 {object} ErrorResponse
	// @Router /api/manual-score/{id} [post]
	// @ID addManualScore
	router.POST("/api/manual-score/:id", SafeHandler(manualScoreHandler(dbConn)))

	// Article analysis
	// @Summary      Get article summary
	// @Description  Returns the generated text summary for an article
	// @Tags         articles
	// @Param        id   path     int  true  "Article ID"
	// @Success      200  {object} StandardResponse{data=string}
	// @Failure      404  {object} StandardResponse
	// @Router       /api/articles/{id}/summary [get]
	// @ID getArticleSummary
	handler := NewSummaryHandler(&db.DBInstance{DB: dbConn})
	router.GET("/api/articles/:id/summary", SafeHandler(handler.Handle))

	// @Summary Get bias analysis
	// @Description Get the bias analysis for an article
	// @Tags Analysis
	// @Accept json
	// @Produce json
	// @Param id path integer true "Article ID"
	// @Success 200 {object} api.ScoreResponse
	// @Failure 404 {object} ErrorResponse
	// @Router /api/articles/{id}/bias [get]
	router.GET("/api/articles/:id/bias", SafeHandler(biasHandler(dbConn)))

	// @Summary Get ensemble details
	// @Description Get detailed ensemble analysis results for an article
	// @Tags Analysis
	// @Param id path integer true "Article ID"
	// @Success 200 {object} api.StandardResponse
	// @Failure 404 {object} ErrorResponse
	// @Router /api/articles/{id}/ensemble [get]
	// @ID getArticleEnsemble
	router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(dbConn)))

	// For backward compatibility with the frontend
	router.GET("/api/articles/:id/ensemble-details", SafeHandler(ensembleDetailsHandler(dbConn)))

	// Feedback
	// @Summary Submit feedback
	// @Description Submit user feedback for an article analysis
	// @Tags Feedback
	// @Accept json
	// @Produce json
	// @Param feedback body models.FeedbackRequest true "Feedback information"
	// @Success 200 {object} StandardResponse
	// @Failure 400 {object} ErrorResponse
	// @Router /api/feedback [post]
	// @ID submitFeedback
	router.POST("/api/feedback", SafeHandler(feedbackHandler(dbConn, llmClient)))

	// Health checks
	// @Summary Get RSS feed health status
	// @Description Returns the health status of all configured RSS feeds
	// @Tags Feeds
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]bool "Feed health status mapping feed names to boolean status"
	// @Failure 500 {object} ErrorResponse "Server error"
	// @Router /api/feeds/healthz [get]
	// @ID getFeedsHealth
	router.GET("/api/feeds/healthz", SafeHandler(feedHealthHandler(rssCollector)))

	// @Summary Check LLM API key health
	// @Description Validates the LLM API key and returns health status
	// @Tags LLM
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse{data=map[string]interface{}} "API key is valid"
	// @Failure 401 {object} ErrorResponse "Invalid API key"
	// @Failure 402 {object} ErrorResponse "Insufficient credits"
	// @Failure 429 {object} ErrorResponse "Rate limited"
	// @Failure 503 {object} ErrorResponse "Service unavailable"
	// @Router /api/llm/health [get]
	// @ID getLLMHealth
	router.GET("/api/llm/health", SafeHandler(llmHealthHandler(llmClient)))

	// Progress tracking
	// @Summary Score progress
	// @Description Get real-time progress updates for article scoring
	// @Tags LLM
	// @Accept json
	// @Produce text/event-stream
	// @Param id path integer true "Article ID"	// @Success 200 {object} models.ProgressResponse
	// @Router /api/llm/score-progress/{id} [get]
	// @ID getScoreProgress
	router.GET("/api/llm/score-progress/:id", SafeHandler(scoreProgressSSEHandler(scoreManager)))

	// Source management endpoints
	// @Summary Get all sources
	// @Description Get a list of all sources with optional filtering and pagination
	// @Tags Sources
	// @Accept json
	// @Produce json
	// @Param enabled query boolean false "Filter by enabled status"
	// @Param channel_type query string false "Filter by channel type"
	// @Param category query string false "Filter by category (left, center, right)"
	// @Param include_stats query boolean false "Include source statistics"
	// @Param limit query integer false "Number of items per page (default: 50, max: 100)"
	// @Param offset query integer false "Pagination offset (default: 0)"
	// @Success 200 {object} StandardResponse{data=models.SourceListResponse}
	// @Failure 400 {object} ErrorResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/sources [get]
	router.GET("/api/sources", SafeHandler(getSourcesHandler(dbConn)))

	// @Summary Create a new source
	// @Description Create a new news source
	// @Tags Sources
	// @Accept json
	// @Produce json
	// @Param source body models.CreateSourceRequest true "Source object"
	// @Success 201 {object} StandardResponse{data=models.Source}
	// @Failure 400 {object} ErrorResponse
	// @Failure 409 {object} ErrorResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/sources [post]
	router.POST("/api/sources", SafeHandler(createSourceHandler(dbConn)))

	// @Summary Get source by ID
	// @Description Get a specific source by its ID
	// @Tags Sources
	// @Accept json
	// @Produce json
	// @Param id path integer true "Source ID"
	// @Param include_stats query boolean false "Include source statistics"
	// @Success 200 {object} StandardResponse{data=models.SourceWithStats}
	// @Failure 400 {object} ErrorResponse
	// @Failure 404 {object} ErrorResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/sources/{id} [get]
	router.GET("/api/sources/:id", SafeHandler(getSourceByIDHandler(dbConn)))

	// @Summary Update source
	// @Description Update an existing source
	// @Tags Sources
	// @Accept json
	// @Produce json
	// @Param id path integer true "Source ID"
	// @Param source body models.UpdateSourceRequest true "Source update object"
	// @Success 200 {object} StandardResponse{data=models.Source}
	// @Failure 400 {object} ErrorResponse
	// @Failure 404 {object} ErrorResponse
	// @Failure 409 {object} ErrorResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/sources/{id} [put]
	router.PUT("/api/sources/:id", SafeHandler(updateSourceHandler(dbConn)))

	// @Summary Delete source (soft delete)
	// @Description Disable a source (soft delete)
	// @Tags Sources
	// @Accept json
	// @Produce json
	// @Param id path integer true "Source ID"
	// @Success 200 {object} StandardResponse{data=string}
	// @Failure 400 {object} ErrorResponse
	// @Failure 404 {object} ErrorResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/sources/{id} [delete]
	router.DELETE("/api/sources/:id", SafeHandler(deleteSourceHandler(dbConn)))

	// @Summary Get source statistics
	// @Description Get detailed statistics for a specific source
	// @Tags Sources
	// @Accept json
	// @Produce json
	// @Param id path integer true "Source ID"
	// @Success 200 {object} StandardResponse{data=models.SourceStats}
	// @Failure 400 {object} ErrorResponse
	// @Failure 404 {object} ErrorResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/sources/{id}/stats [get]
	router.GET("/api/sources/:id/stats", SafeHandler(getSourceStatsHandler(dbConn)))

	// Admin endpoints
	// @Summary Refresh all RSS feeds
	// @Description Triggers a manual refresh of all configured RSS feeds
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/admin/refresh-feeds [post]
	router.POST("/api/admin/refresh-feeds", SafeHandler(adminRefreshFeedsHandler(rssCollector)))

	// @Summary Reset feed errors
	// @Description Resets error states for RSS feeds
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Router /api/admin/reset-feed-errors [post]
	router.POST("/api/admin/reset-feed-errors", SafeHandler(adminResetFeedErrorsHandler(rssCollector)))

	// @Summary Get sources status
	// @Description Returns health status of all RSS feed sources
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Router /api/admin/sources [get]
	router.GET("/api/admin/sources", SafeHandler(adminGetSourcesStatusHandler(rssCollector)))

	// @Summary Reanalyze recent articles
	// @Description Triggers reanalysis of recent articles using LLM
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Failure 503 {object} ErrorResponse
	// @Router /api/admin/reanalyze-recent [post]
	router.POST("/api/admin/reanalyze-recent", SafeHandler(adminReanalyzeRecentHandler(llmClient, scoreManager, dbConn)))

	// @Summary Clear analysis errors
	// @Description Clears error states for articles with failed analysis
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Router /api/admin/clear-analysis-errors [post]
	router.POST("/api/admin/clear-analysis-errors", SafeHandler(adminClearAnalysisErrorsHandler(dbConn)))

	// @Summary Validate bias scores
	// @Description Validates consistency and validity of bias scores
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Router /api/admin/validate-scores [post]
	router.POST("/api/admin/validate-scores", SafeHandler(adminValidateBiasScoresHandler(llmClient, scoreManager, dbConn)))

	// @Summary Optimize database
	// @Description Runs database optimization (VACUUM and ANALYZE)
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/admin/optimize-db [post]
	router.POST("/api/admin/optimize-db", SafeHandler(adminOptimizeDatabaseHandler(dbConn)))

	// @Summary Export data
	// @Description Exports articles and scores as CSV
	// @Tags Admin
	// @Produce text/csv
	// @Success 200 {string} string "CSV file download"
	// @Router /api/admin/export [get]
	router.GET("/api/admin/export", SafeHandler(adminExportDataHandler(dbConn)))

	// @Summary Cleanup old articles
	// @Description Deletes articles older than 30 days
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /api/admin/cleanup-old [delete]
	router.DELETE("/api/admin/cleanup-old", SafeHandler(adminCleanupOldArticlesHandler(dbConn)))

	// @Summary Get system metrics
	// @Description Returns system statistics and metrics
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} SystemStatsResponse
	// @Router /api/admin/metrics [get]
	router.GET("/api/admin/metrics", SafeHandler(adminGetMetricsHandler(dbConn)))

	// @Summary Get system logs
	// @Description Returns recent system log entries
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Router /api/admin/logs [get]
	router.GET("/api/admin/logs", SafeHandler(adminGetLogsHandler()))

	// @Summary Run health check
	// @Description Performs comprehensive system health check
	// @Tags Admin
	// @Accept json
	// @Produce json
	// @Success 200 {object} SystemHealthResponse
	// @Router /api/admin/health-check [post]
	router.POST("/api/admin/health-check", SafeHandler(adminRunHealthCheckHandler(dbConn, llmClient, rssCollector)))

	// HTMX Admin Source Management Routes
	router.GET("/htmx/sources", SafeHandler(adminSourcesListHandler(dbConn)))
	router.GET("/htmx/sources/new", SafeHandler(adminSourceFormHandler(dbConn)))
	router.GET("/htmx/sources/:id/edit", SafeHandler(adminSourceFormHandler(dbConn)))
	router.GET("/htmx/sources/:id/stats", SafeHandler(adminSourceStatsHandler(dbConn)))
}

// SafeHandler wraps a handler function with panic recovery to prevent server crashes
func SafeHandler(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] %v\\n%s", r, debug.Stack())
				RespondError(c, NewAppError(ErrInternal, fmt.Sprintf("Internal Server Error: %v", r)))
			}
		}()
		handler(c)
	}
}

// Helper: Convert db.Article to API ArticleResponse
func toArticleResponse(a *db.Article) ArticleResponse {
	// Handle nil pointers for scores
	composite := 0.0
	if a.CompositeScore != nil {
		composite = *a.CompositeScore
	}

	confidence := 0.0
	if a.Confidence != nil {
		confidence = *a.Confidence
	}

	scoreSource := ""
	if a.ScoreSource != nil {
		scoreSource = *a.ScoreSource
	}

	return ArticleResponse{
		ArticleID:   a.ID,
		Source:      a.Source,
		URL:         a.URL,
		Title:       a.Title,
		Content:     a.Content,
		PublishedAt: a.PubDate.Format(time.RFC3339),
		Composite:   composite,
		Confidence:  confidence,
		ScoreSource: scoreSource,
	}
}

// Handler for POST /api/articles
// @Summary Create article
// @Description Creates a new article with the provided information
// @Tags Articles
// @Accept json
// @Produce json
// @Param request body CreateArticleRequest true "Article information"
// @Success 200 {object} StandardResponse{data=CreateArticleResponse} "Article created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 409 {object} ErrorResponse "Article URL already exists"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /api/articles [post]
// @ID createArticle
func createArticleHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Source  string `json:"source"`
			PubDate string `json:"pub_date"`
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			if strings.Contains(err.Error(), "unknown field") {
				RespondError(c, NewAppError(ErrValidation, "Request contains unknown or extra fields"))
				return
			}
			RespondError(c, ErrInvalidPayload)
			return
		}

		// Validate required fields
		var missingFields []string
		if req.Source == "" {
			missingFields = append(missingFields, "source")
		}
		if req.URL == "" {
			missingFields = append(missingFields, "url")
		}
		if req.Title == "" {
			missingFields = append(missingFields, "title")
		}
		if req.Content == "" {
			missingFields = append(missingFields, "content")
		}
		if req.PubDate == "" {
			missingFields = append(missingFields, "pub_date")
		}

		if len(missingFields) > 0 {
			RespondError(c, NewAppError(ErrValidation,
				fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", "))))
			return
		}

		// Validate URL format
		if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
			RespondError(c, NewAppError(ErrValidation, "Invalid URL format (must start with http:// or https://)"))
			return
		}

		exists, err := db.ArticleExistsByURL(dbConn, req.URL)
		if err != nil {
			RespondError(c, WrapError(err, ErrInternal, "Failed to check for existing article"))
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "duplicate_url",
					"message": "Article with this URL already exists",
				},
			})
			return
		}

		// Parse pub_date
		pubDate, err := time.Parse(time.RFC3339, req.PubDate)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid pub_date format (expected RFC3339)"))
			return
		}
		zero := 0.0
		llmSource := "llm"
		article := &db.Article{
			Source:         req.Source,
			PubDate:        pubDate,
			URL:            req.URL,
			Title:          req.Title,
			Content:        req.Content,
			CreatedAt:      time.Now(),
			CompositeScore: &zero,
			Confidence:     &zero,
			ScoreSource:    &llmSource,
		}

		// Insert article (retry logic is handled at the database layer)
		id, err := db.InsertArticle(dbConn, article)
		if err != nil {
			if errors.Is(err, db.ErrDuplicateURL) {
				c.JSON(http.StatusConflict, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "duplicate_url",
						"message": "Article with this URL already exists",
					},
				})
				return
			}
			RespondError(c, WrapError(err, ErrInternal, "Failed to create article"))
			return
		}

		// Fetch the full article object after creation
		createdArticle, err := db.FetchArticleByID(dbConn, id)
		if err != nil {
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch created article"))
			return
		}

		resp := toArticleResponse(createdArticle)
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data":    resp,
		})
	}
}

// Utility function for handling article array results
// ... remove handleArticleBatch function ...

// Handler for GET /api/articles
// @Summary Get articles
// @Description Fetches a list of articles with optional filtering by source, leaning, and pagination
// @Tags Articles
// @Accept json
// @Produce json
// @Param source query string false "Filter by news source"
// @Param leaning query string false "Filter by political leaning (left/center/right)"
// @Param offset query integer false "Pagination offset" default(0) minimum(0)
// @Param limit query integer false "Number of items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} StandardResponse{data=[]ArticleResponse} "List of articles"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /api/articles [get]
// @ID getArticlesList
func getArticlesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		safeLogf("[DEBUG] getArticlesHandler: Entered handler. Request: %s", c.Request.URL.String())

		source := c.Query("source")
		leaning := c.Query("leaning")
		limitStr := c.DefaultQuery("limit", "20")
		offsetStr := c.DefaultQuery("offset", "0")

		safeLogf("[DEBUG] getArticlesHandler: Parsed query params - source: %s, leaning: %s, limit: %s, offset: %s", source, leaning, limitStr, offsetStr)

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			safeLogf("[ERROR] getArticlesHandler: invalid limit parameter: %v. Value: %s", err, limitStr)
			RespondError(c, NewAppError(ErrValidation, "Invalid 'limit' parameter"))
			return
		}
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			safeLogf("[ERROR] getArticlesHandler: invalid offset parameter: %v. Value: %s", err, offsetStr)
			RespondError(c, NewAppError(ErrValidation, "Invalid 'offset' parameter"))
			return
		}

		safeLogf("[INFO] getArticlesHandler: Fetching articles (source=%s, leaning=%s, limit=%d, offset=%d)", source, leaning, limit, offset)
		// Corrected parameters for db.FetchArticles
		safeLogf("[DEBUG] getArticlesHandler: Calling db.FetchArticles with source: '%s', leaning: '%s', limit: %d, offset: %d", source, leaning, limit, offset)
		articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
		// totalCount is not returned by FetchArticles, so its usage is removed for now.
		log.Printf("[DEBUG] getArticlesHandler: After db.FetchArticles. Error: %v. Articles count: %d", err, len(articles))

		if err != nil {
			safeLogf("[ERROR] getArticlesHandler: Raw error from db.FetchArticles: %#v. Params - Source: '%s', Leaning: '%s', Limit: %d, Offset: %d", err, source, leaning, limit, offset)
			RespondError(c, WrapError(err, ErrInternal, fmt.Sprintf("Failed to fetch articles: %v", err)))
			return
		}

		// Estimate totalCount for now. This should be replaced with a proper count query later.
		totalCount := offset + len(articles)
		if len(articles) == limit {
			// If we fetched a full page, there might be more records.
			// This is a rough estimation and might not be accurate.
			// A separate COUNT(*) query would be more reliable.
			totalCount += 1 // Placeholder to indicate more might exist
		}

		if len(articles) == 0 {
			c.Header("X-Total-Count", strconv.Itoa(totalCount))
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    []ArticleResponse{},
			})
			return
		}

		// Enhance articles with composite scores and confidence (simplified error handling for now)
		for i := range articles {
			scores, fetchErr := db.FetchLLMScores(dbConn, articles[i].ID)
			if fetchErr != nil {
				log.Printf("WARNING: getArticlesHandler - Error fetching LLM scores for article ID %d: %v", articles[i].ID, fetchErr)
			} else if len(scores) > 0 {
				var weightedSum, sumWeights float64
				validScoresCount := 0
				for _, s := range scores {
					var meta struct {
						Confidence float64 `json:"confidence"`
					}
					if s.Metadata != "" {
						if metaErr := json.Unmarshal([]byte(s.Metadata), &meta); metaErr != nil {
							log.Printf("WARNING: getArticlesHandler - Error unmarshalling metadata for score ID %d (article ID %d): %v", s.ID, articles[i].ID, metaErr)
							continue // Skip this score if metadata is malformed
						}
					} else {
						log.Printf("WARNING: getArticlesHandler - Empty metadata for score ID %d (article ID %d)", s.ID, articles[i].ID)
						continue // Skip this score if metadata is empty
					}
					weightedSum += s.Score * meta.Confidence
					sumWeights += meta.Confidence
					validScoresCount++
				}
				if sumWeights > 0 && validScoresCount > 0 {
					compositeScore := weightedSum / sumWeights
					avgConfidence := sumWeights / float64(validScoresCount)
					articles[i].CompositeScore = &compositeScore
					articles[i].Confidence = &avgConfidence
				}
			}
		}

		var out []ArticleResponse
		for i := range articles {
			out = append(out, toArticleResponse(&articles[i]))
		}

		c.Header("X-Total-Count", strconv.Itoa(totalCount))
		log.Printf("[DEBUG] getArticlesHandler: Preparing to send response. Number of articles: %d", len(out))
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    out,
		})
		log.Printf("[DEBUG] getArticlesHandler: Response sent successfully.")
	}
}

// Helper: Validate article ID from path param
func getValidArticleID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	if idStr == "null" || idStr == "undefined" || idStr == "" {
		RespondError(c, NewAppError(ErrValidation, "Invalid article ID (null or empty)"))
		return 0, false
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 1 {
		RespondError(c, NewAppError(ErrValidation, "Invalid article ID (must be a positive integer)"))
		return 0, false
	}
	return id, true
}

// getArticleByIDHandler handles GET /articles/:id
// @Summary Get article by ID
// @Description Fetches a specific article by its ID with scores and metadata
// @Tags Articles
// @Accept json
// @Produce json
// @Param id path int true "Article ID" minimum(1)
// @Success 200 {object} StandardResponse "Success with article details"
// @Failure 400 {object} ErrorResponse "Invalid article ID"
// @Failure 404 {object} ErrorResponse "Article not found"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /api/articles/{id} [get]
// @ID getArticleById
func getArticleByIDHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		id, ok := getValidArticleID(c)
		if !ok {
			return
		}

		// Check for cache busting parameter
		_, skipCache := c.GetQuery("_t")

		// Caching
		cacheKey := "article:" + strconv.FormatInt(id, 10)
		if !skipCache {
			articlesCacheLock.RLock()
			if cached, found := articlesCache.Get(cacheKey); found {
				articlesCacheLock.RUnlock()
				RespondSuccess(c, cached)
				LogPerformance("getArticleByIDHandler (cache hit)", start)
				return
			}
			articlesCacheLock.RUnlock()
		} else {
			log.Printf("[getArticleByIDHandler] Cache busting requested for article %d", id)
		}

		article, err := db.FetchArticleByID(dbConn, id)
		if err != nil {
			if errors.Is(err, db.ErrArticleNotFound) {
				RespondError(c, ErrArticleNotFound)
				return
			}
			LogError(c, err, "getArticleByIDHandler: fetch article")
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch article"))
			return
		}

		resp := toArticleResponse(article)

		// Cache the result for 30 seconds
		articlesCacheLock.Lock()
		articlesCache.Set(cacheKey, resp, 30*time.Second)
		articlesCacheLock.Unlock()

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    resp,
		})
		LogPerformance("getArticleByIDHandler", start)
	}
}

// @Summary Trigger RSS feed refresh
// @Description Initiates a manual RSS feed refresh job to fetch new articles from configured RSS sources
// @Tags Feeds
// @Accept json
// @Produce json
// @Success 200 {object} StandardResponse{data=map[string]string} "Refresh started successfully"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /api/refresh [post]
// @ID triggerRssRefresh
func refreshHandler(rssCollector rss.CollectorInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		go rssCollector.ManualRefresh()
		RespondSuccess(c, map[string]string{"status": "refresh started"})
		LogPerformance("refreshHandler", start)
	}
}

// translateReanalysisError converts technical errors into user-friendly messages
func translateReanalysisError(err error) (userMessage string, step string) {
	errStr := strings.ToLower(err.Error())

	// Check for specific error patterns
	switch {
	case strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "timeout"):
		if strings.Contains(errStr, "awaiting headers") {
			return "API connection failed. Please check your API key configuration.", "API Connection Failed"
		}
		return "Request timed out. The LLM service may be experiencing high load.", "Request Timeout"

	case strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "invalid api key"):
		return "Invalid API key. Please check your OpenRouter API key configuration.", "Authentication Failed"

	case strings.Contains(errStr, "402") || strings.Contains(errStr, "payment required") || strings.Contains(errStr, "credits"):
		return "Insufficient credits. Please add credits to your OpenRouter account.", "Payment Required"

	case strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit"):
		return "Rate limit exceeded. Please wait a few minutes before trying again.", "Rate Limited"

	case strings.Contains(errStr, "503") || strings.Contains(errStr, "service unavailable"):
		return "LLM service is temporarily unavailable. Please try again later.", "Service Unavailable"

	case strings.Contains(errStr, "network") || strings.Contains(errStr, "connection"):
		return "Network connection error. Please check your internet connection.", "Network Error"

	case strings.Contains(errStr, "llm_api_key not set"):
		return "API key not configured. Please set your OpenRouter API key.", "Configuration Error"

	default:
		return "Analysis failed due to an unexpected error. Please try again.", "Analysis Failed"
	}
}

// Refactored reanalyzeHandler to use ScoreManager for scoring, storage, and progress
// @Summary Reanalyze article
// @Description Trigger a new LLM analysis for a specific article and update its scores.
// @Tags LLM
// @Accept json
// @Produce json
// @Param id path integer true "Article ID"
// @Success 202 {object} StandardResponse "Reanalysis started"
// @Failure 400 {object} ErrorResponse "Invalid article ID"
// @Failure 401 {object} ErrorResponse "LLM authentication failed"
// @Failure 402 {object} ErrorResponse "LLM payment required or credits exhausted"
// @Failure 404 {object} ErrorResponse "Article not found"
// @Failure 429 {object} ErrorResponse "LLM rate limit exceeded"
// @Failure 500 {object} ErrorResponse "Server error"
// @Failure 503 {object} ErrorResponse "LLM service unavailable or streaming error"
// @Router /api/llm/reanalyze/{id} [post]
// @ID reanalyzeArticle
func reanalyzeHandler(llmClient *llm.LLMClient, dbConn *sqlx.DB, scoreManager *llm.ScoreManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := getValidArticleID(c)
		if !ok {
			return
		}
		articleID := id
		log.Printf("[POST /api/llm/reanalyze] ArticleID=%d", articleID)

		// Verify article exists
		_, err := db.FetchArticleByID(dbConn, articleID)
		if err != nil {
			if errors.Is(err, db.ErrArticleNotFound) {
				RespondError(c, ErrArticleNotFound)
				return
			}
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch article"))
			return
		}

		// Accept empty or missing JSON body gracefully
		var raw map[string]interface{}
		if c.Request.ContentLength == 0 {
			raw = map[string]interface{}{} // treat as empty
		} else {
			if err := c.ShouldBindJSON(&raw); err != nil {
				RespondError(c, ErrInvalidPayload)
				return
			}
		}

		// Direct score update path - check if "score" field exists
		if scoreRaw, hasScore := raw["score"]; hasScore {
			var scoreFloat float64
			switch s := scoreRaw.(type) {
			case float64:
				scoreFloat = s
			case float32:
				scoreFloat = float64(s)
			case int:
				scoreFloat = float64(s)
			case int64:
				scoreFloat = float64(s)
			case string:
				var parseErr error
				scoreFloat, parseErr = strconv.ParseFloat(s, 64)
				if parseErr != nil {
					LogError(c, parseErr, "reanalyzeHandler: invalid score format")
					RespondError(c, NewAppError(ErrValidation, "Invalid score value"))
					return
				}
			default:
				RespondError(c, NewAppError(ErrValidation, "Invalid score value"))
				LogError(c, nil, "reanalyzeHandler: invalid score value")
				return
			}

			if scoreFloat < -1.0 || scoreFloat > 1.0 {
				RespondError(c, NewAppError(ErrValidation, "Score must be between -1.0 and 1.0"))
				LogError(c, nil, "reanalyzeHandler: invalid score value")
				return
			}

			confidence := 1.0 // Use maximum confidence for direct score updates
			err = db.UpdateArticleScoreLLM(dbConn, articleID, scoreFloat, confidence)
			if err != nil {
				RespondError(c, NewAppError(ErrInternal, "Failed to update article score"))
				LogError(c, err, "reanalyzeHandler: failed to update article score")
				return
			}

			// Invalidate cache using ScoreManager
			if scoreManager != nil {
				scoreManager.InvalidateScoreCache(articleID)
			}

			RespondSuccess(c, map[string]interface{}{
				"status":     "score updated",
				"article_id": articleID,
				"score":      scoreFloat,
			})
			return
		}

		// Load composite score config to get the models
		cfg, cfgErr := llm.LoadCompositeScoreConfig()
		if cfgErr != nil || len(cfg.Models) == 0 {
			RespondError(c, WrapError(cfgErr, ErrLLMService, "Failed to load LLM configuration"))
			return
		}

		// Check if models are configured
		if len(cfg.Models) == 0 {
			log.Printf("[reanalyzeHandler %d] No models configured, cannot proceed.", articleID)
			RespondError(c, NewAppError(ErrLLMService, "No LLM models configured"))
			return
		}

		log.Printf("[reanalyzeHandler %d] Proceeding with reanalysis - ReanalyzeArticle will handle model fallbacks", articleID)

		// Start the reanalysis process
		if scoreManager != nil {
			// Set initial progress BEFORE responding to the client
			initialProgress := &models.ProgressState{
				Status:  "Queued",
				Step:    "Pending",
				Message: "Reanalysis queued for all configured models",
			}
			log.Printf("[reanalyzeHandler %d] Setting initial progress: %+v", articleID, initialProgress)
			scoreManager.SetProgress(articleID, initialProgress)

			// Check for an environment variable to skip auto-analysis during tests
			if os.Getenv("NO_AUTO_ANALYZE") != "true" {
				go func() {
					// Pass scoreManager to ReanalyzeArticle
					err := llmClient.ReanalyzeArticle(context.Background(), articleID, scoreManager)
					if err != nil {
						log.Printf("[reanalyzeHandler %d] Error during reanalysis: %v", articleID, err)
						// Ensure scoreManager is not nil before using
						if scoreManager != nil {
							userMessage, step := translateReanalysisError(err)
							scoreManager.SetProgress(articleID, &models.ProgressState{
								Status:  "Error",
								Step:    step,
								Message: userMessage,
								Error:   err.Error(), // Keep technical error for debugging
							})
						}
						return
					}
					// Ensure scoreManager is not nil before using
					if scoreManager != nil {
						// Fetch the final score to include in the progress update
						finalProgressState := scoreManager.GetProgress(articleID)
						var finalScore *float64
						article, fetchErr := db.FetchArticleByID(dbConn, articleID)
						if fetchErr == nil && article.CompositeScore != nil {
							finalScore = article.CompositeScore
						} else if fetchErr != nil {
							log.Printf("[reanalyzeHandler %d] Could not fetch article to get final score for progress: %v", articleID, fetchErr)
						} else {
							log.Printf("[reanalyzeHandler %d] Article fetched but composite score is nil for progress update.", articleID)
						}

						// If the ReanalyzeArticle function set a near-complete state, update it to full "Complete"
						// Otherwise, create a new one.
						log.Printf("[reanalyzeHandler %d] Current progress state: %+v", articleID, finalProgressState)
						if finalProgressState != nil && finalProgressState.Status == "InProgress" && finalProgressState.Percent == 99 {
							log.Printf("[reanalyzeHandler %d] Updating existing progress state to Complete", articleID)
							finalProgressState.Status = "Complete"
							finalProgressState.Step = "Done"
							finalProgressState.Message = "Analysis complete"
							finalProgressState.Percent = 100
							finalProgressState.FinalScore = finalScore
							scoreManager.SetProgress(articleID, finalProgressState)
						} else {
							log.Printf("[reanalyzeHandler %d] Setting new Complete progress state", articleID)
							scoreManager.SetProgress(articleID, &models.ProgressState{
								Status:     "Complete",
								Step:       "Done",
								Message:    "Analysis complete",
								Percent:    100,
								FinalScore: finalScore, // Include final score if available
							})
						}
					}
				}()
			} else {
				log.Printf("[reanalyzeHandler %d] NO_AUTO_ANALYZE is set, skipping background reanalysis.", articleID)
				// Optionally, set progress to complete or a specific "skipped" state
				scoreManager.SetProgress(articleID, &models.ProgressState{
					Status:      "Skipped", // Ensure this status is handled by SSE or test
					Step:        "Skipped",
					Message:     "Automatic reanalysis skipped by test configuration.",
					Percent:     100,
					LastUpdated: time.Now().Unix(),
				})
			}
		} else {
			log.Printf("[reanalyzeHandler %d] ScoreManager is nil, cannot set progress.", articleID)
		}

		RespondSuccess(c, map[string]interface{}{
			"status":     "reanalyze queued",
			"article_id": articleID,
		})
	}
}

// @Summary   Stream LLM scoring progress
// @Produce   text/event-stream
// @Param     id  path  int  true  "Article ID"
// @Success   200  {object} models.ProgressState  "SSE stream of progress updates"
// @Failure   400  {object} StandardResponse
// @Router    /api/llm/score-progress/{id} [get]
// @ID getScoreProgress
func scoreProgressSSEHandler(scoreManager *llm.ScoreManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := getValidArticleID(c)
		if !ok {
			// It's important to set headers before writing the body for SSE
			c.Writer.Header().Set("Content-Type", "text/event-stream")
			c.Writer.Header().Set("Cache-Control", "no-cache")
			c.Writer.Header().Set("Connection", "keep-alive")
			if _, err := c.Writer.Write([]byte("event: error\ndata: {\"error\":\"Invalid article ID\"}\n\n")); err != nil {
				log.Printf("[SSE HANDLER] Error writing SSE error for invalid article ID: %v", err)
			}
			c.Writer.Flush() // Ensure data is sent
			return
		}
		articleID := id
		log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Connection established.", articleID)

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		// Explicitly flush headers to ensure client connection is established before first tick
		c.Writer.Flush()

		lastProgressJSON := ""
		ticker := time.NewTicker(250 * time.Millisecond) // Reduced ticker for faster updates during debugging
		defer func() {
			ticker.Stop()
			log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Connection closed.", articleID)
		}()

		// Send an initial "connected" or "pending" event immediately
		// This helps confirm the connection is working before the first tick
		initialState := &models.ProgressState{
			Status:  "Connected",
			Step:    "Initializing",
			Message: "SSE connection established, awaiting progress.",
			Percent: 0,
		}
		if scoreManager != nil {
			// Check if there's already an initial state (e.g. "Queued" from reanalyzeHandler)
			// If so, send that instead of a generic "Connected"
			existingState := scoreManager.GetProgress(articleID)
			if existingState != nil {
				initialState = existingState
				log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Found existing initial state: %+v", articleID, initialState)
			} else {
				log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: No existing state, sending generic 'Connected' state.", articleID)
			}
		} else {
			log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: ScoreManager is nil, cannot fetch initial state.", articleID)
		}

		initialData, err := json.Marshal(initialState)
		if err == nil {
			log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Sending initial event: event: progress, data: %s", articleID, string(initialData))
			if _, writeErr := fmt.Fprintf(c.Writer, "event: progress\ndata: %s\n\n", initialData); writeErr != nil {
				log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Error writing initial SSE event: %v", articleID, writeErr)
				// Don't return here, try to continue with the ticker
			}
			c.Writer.Flush() // Ensure the initial event is sent immediately
		} else {
			log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Error marshalling initial state: %v", articleID, err)
		}

		for {
			select {
			case <-c.Request.Context().Done():
				log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Client disconnected.", articleID)
				return
			case <-ticker.C:
				var progress *models.ProgressState
				if scoreManager != nil {
					progress = scoreManager.GetProgress(articleID)
				} else {
					log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: ScoreManager is nil in ticker loop.", articleID)
					// Optionally send an error event or just continue
					continue
				}

				if progress == nil {
					// log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: No progress update found.", articleID)
					continue // No progress update yet
				}

				// log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Fetched progress: %+v", articleID, progress)

				data, err := json.Marshal(progress)
				if err != nil {
					log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Error marshalling progress: %v", articleID, err)
					continue
				}

				currentProgressJSON := string(data)
				if currentProgressJSON != lastProgressJSON {
					log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Sending progress update: %s", articleID, currentProgressJSON)
					if _, err := fmt.Fprintf(c.Writer, "event: progress\ndata: %s\n\n", data); err != nil {
						log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Error writing to client: %v", articleID, err)
						return // Stop if we can't write to client
					}
					c.Writer.Flush() // Ensure data is sent immediately
					lastProgressJSON = currentProgressJSON

					// Check for terminal states
					if progress.Status == "Complete" || progress.Status == "Error" || progress.Status == "Success" {
						log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Terminal progress status '%s' received. Closing SSE stream.", articleID, progress.Status)

						// For "Complete" status, delay closure to allow frontend to process and display completion
						if progress.Status == "Complete" {
							log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Delaying SSE closure for 3 seconds to allow frontend completion processing.", articleID)
							time.Sleep(3 * time.Second)
						}

						// Send one last time to be sure, then close.
						// It's possible the client might miss this if we return immediately.
						// However, the test client should see the event and then the connection close.
						return
					}
				} else {
					// Progress unchanged - no action needed
					log.Printf("[SSE HANDLER /api/llm/score-progress] ArticleID=%d: Progress unchanged: %s", articleID, currentProgressJSON)
				}
			}
		}
	}
}

// @Summary Get RSS feed health status
// @Description Returns the health status of all configured RSS feeds
// @Tags Feeds
// @Accept json
// @Produce json
// @Success 200 {object} map[string]bool "Feed health status mapping feed names to boolean status"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /api/feeds/healthz [get]
// @ID getFeedsHealth
func feedHealthHandler(rssCollector rss.CollectorInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := rssCollector.CheckFeedHealth()
		c.JSON(200, status)
	}
}

// @Summary Check LLM API key health
// @Description Validates the LLM API key and returns health status
// @Tags LLM
// @Accept json
// @Produce json
// @Success 200 {object} StandardResponse{data=map[string]interface{}} "API key is valid"
// @Failure 401 {object} ErrorResponse "Invalid API key"
// @Failure 402 {object} ErrorResponse "Insufficient credits"
// @Failure 429 {object} ErrorResponse "Rate limited"
// @Failure 503 {object} ErrorResponse "Service unavailable"
// @Router /api/llm/health [get]
// @ID getLLMHealth
func llmHealthHandler(llmClient *llm.LLMClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if llmClient == nil {
			RespondError(c, NewAppError(ErrInternal, "LLM client not initialized"))
			return
		}

		// Validate the API key
		err := llmClient.ValidateAPIKey()
		if err != nil {
			log.Printf("[LLM Health Check] API key validation failed: %v", err)

			// Determine appropriate HTTP status code based on error
			var statusCode int
			var errorType string

			errStr := strings.ToLower(err.Error())
			switch {
			case strings.Contains(errStr, "invalid api key"):
				statusCode = 401
				errorType = "invalid_api_key"
			case strings.Contains(errStr, "insufficient credits"):
				statusCode = 402
				errorType = "insufficient_credits"
			case strings.Contains(errStr, "rate limited"):
				statusCode = 429
				errorType = "rate_limited"
			case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "unavailable"):
				statusCode = 503
				errorType = "service_unavailable"
			default:
				statusCode = 503
				errorType = "unknown_error"
			}

			c.JSON(statusCode, gin.H{
				"success": false,
				"error": gin.H{
					"type":    errorType,
					"message": err.Error(),
				},
			})
			return
		}

		// API key is valid
		RespondSuccess(c, map[string]interface{}{
			"status":     "healthy",
			"api_key":    "valid",
			"message":    "LLM API key is valid and working",
			"checked_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// @Summary Get article summary
// @Description Retrieves the text summary for an article
// @Tags Summary
// @Param id path int true "Article ID" minimum(1)
// @Success 200 {object} StandardResponse
// @Failure 404 {object} ErrorResponse "Summary not available"
// @Router /api/articles/{id}/summary [get]
// @ID getArticleSummary
type SummaryHandler struct {
	db db.DBOperations
}

func NewSummaryHandler(db db.DBOperations) *SummaryHandler {
	return &SummaryHandler{db: db}
}

func (h *SummaryHandler) Handle(c *gin.Context) {
	start := time.Now()
	id, ok := getValidArticleID(c)
	if !ok {
		return
	}

	// Caching
	cacheKey := "summary:" + strconv.FormatInt(id, 10)
	articlesCacheLock.RLock()
	if cached, found := articlesCache.Get(cacheKey); found {
		articlesCacheLock.RUnlock()
		RespondSuccess(c, cached)
		LogPerformance("summaryHandler (cache hit)", start)
		return
	}
	articlesCacheLock.RUnlock()

	// Verify article exists
	_, err := h.db.FetchArticleByID(c, id)
	if err != nil {
		if errors.Is(err, db.ErrArticleNotFound) {
			RespondError(c, ErrArticleNotFound)
			return
		}
		RespondError(c, WrapError(err, ErrInternal, "Failed to fetch article"))
		return
	}

	scores, err := h.db.FetchLLMScores(c, id)
	if err != nil {
		RespondError(c, WrapError(err, ErrInternal, "Failed to fetch article summary"))
		return
	}

	for _, score := range scores {
		if score.Model == ModelSummarizer {
			// Extract summary text from JSON metadata
			var summaryText string
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(score.Metadata), &meta); err == nil {
				if s, ok := meta["summary"].(string); ok {
					summaryText = s
				}
			}
			result := map[string]interface{}{
				"summary":    summaryText,
				"created_at": score.CreatedAt,
			}
			articlesCacheLock.Lock()
			articlesCache.Set(cacheKey, result, 30*time.Second)
			articlesCacheLock.Unlock()

			RespondSuccess(c, result)
			LogPerformance("summaryHandler", start)
			return
		}
	}

	RespondError(c, NewAppError(ErrNotFound, "Article summary not available"))
	LogPerformance("summaryHandler", start)
}

// biasHandler returns article bias scores and composite score.
// @Summary Get article bias analysis
// @Description Retrieves the political bias score and individual model results for an article
// @Tags Analysis
// @Accept json
// @Produce json
// @Param id path int true "Article ID" minimum(1)
// @Param min_score query number false "Minimum score filter" default(-1) minimum(-1) maximum(1)
// @Param max_score query number false "Maximum score filter" default(1) minimum(-1) maximum(1)
// @Param sort query string false "Sort order (asc or desc)" Enums(asc, desc) default(desc)
// @Success 200 {object} StandardResponse{data=ScoreResponse} "Success"
// @Failure 400 {object} ErrorResponse "Invalid parameters"
// @Failure 404 {object} ErrorResponse "Article not found"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /api/articles/{id}/bias [get]
// @ID getArticleBias
// If no valid LLM scores are available, the API responds with:
//   - "composite_score": null
//   - "status": "scoring_unavailable"
//
// instead of defaulting to zero values.
// This indicates that scoring data is currently unavailable.
func biasHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		id, ok := getValidArticleID(c)
		if !ok {
			return
		}

		minScore, err := strconv.ParseFloat(c.DefaultQuery("min_score", "-1"), 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid min_score"))
			LogError(c, err, "biasHandler: invalid min_score")
			return
		}
		maxScore, err := strconv.ParseFloat(c.DefaultQuery("max_score", "1"), 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid max_score"))
			LogError(c, err, "biasHandler: invalid max_score")
			return
		}
		sortOrder := c.DefaultQuery("sort", SortDesc)
		if sortOrder != SortAsc && sortOrder != SortDesc {
			RespondError(c, NewAppError(ErrValidation, "Invalid sort order"))
			LogError(c, nil, "biasHandler: invalid sort order")
			return
		}

		// Caching
		cacheKey := "bias:" + strconv.FormatInt(id, 10) + ":" +
			c.DefaultQuery("min_score", "-1") + ":" +
			c.DefaultQuery("max_score", "1") + ":" + sortOrder
		articlesCacheLock.RLock()
		if cached, found := articlesCache.Get(cacheKey); found {
			articlesCacheLock.RUnlock()
			RespondSuccess(c, cached)
			LogPerformance("biasHandler (cache hit)", start)
			return
		}
		articlesCacheLock.RUnlock()

		scores, err := db.FetchLLMScores(dbConn, id)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch bias data"))
			LogError(c, err, "biasHandler: fetch scores")
			return
		}

		var latestEnsembleScore *db.LLMScore
		individualResults := make([]map[string]interface{}, 0)

		// Find the latest ensemble score and gather individual scores
		for i := range scores {
			score := scores[i] // Create a copy to avoid loop variable issues if needed later

			if score.Model == ModelEnsemble {
				if latestEnsembleScore == nil || score.CreatedAt.After(latestEnsembleScore.CreatedAt) {
					latestEnsembleScore = &score // Store pointer to the score
				}
			} else {
				// Parse metadata for individual scores
				var meta struct {
					Confidence  *float64 `json:"Confidence"`  // Use pointer for optional field
					Explanation *string  `json:"Explanation"` // Use pointer for optional field
				}
				// Default values
				confidence := 0.0
				explanation := ""

				if score.Metadata != "" {
					if err := json.Unmarshal([]byte(score.Metadata), &meta); err == nil {
						if meta.Confidence != nil {
							confidence = *meta.Confidence
						}
						if meta.Explanation != nil {
							explanation = *meta.Explanation
						}
					} else {
						log.Printf("WARN: biasHandler: Failed to unmarshal metadata for score ID %d, model %s: %v", score.ID, score.Model, err)
					}
				} else {
					log.Printf("WARN: biasHandler: Empty metadata for score ID %d, model %s", score.ID, score.Model)
				}

				// Add to results, applying score filtering
				if score.Score >= minScore && score.Score <= maxScore {
					individualResults = append(individualResults, map[string]interface{}{
						"model":       score.Model,
						"score":       score.Score,
						"confidence":  confidence,
						"explanation": explanation,
						"created_at":  score.CreatedAt, // Include timestamp if needed by frontend/sorting
					})
				}
			}
		}

		// Sort individual results
		sort.SliceStable(individualResults, func(i, j int) bool {
			scoreI, okI := individualResults[i]["score"].(float64)
			scoreJ, okJ := individualResults[j]["score"].(float64)

			if !okI && !okJ { // Both are invalid
				log.Printf("WARN: biasHandler sorting: both scores invalid for comparison at indices %d and %d. Treating as equal.", i, j)
				return false
			}
			if !okI { // item i is invalid, j is valid. Invalid items go to the end.
				log.Printf("WARN: biasHandler sorting: invalid score for result at index %d. Sorting to end.", i)
				return false
			}
			if !okJ { // item i is valid, j is invalid. Invalid items go to the end.
				log.Printf("WARN: biasHandler sorting: invalid score for result at index %d. Sorting to end.", j)
				return true
			}

			// Both are valid
			if sortOrder == SortAsc {
				return scoreI < scoreJ
			}
			return scoreI > scoreJ // desc
		})

		var compositeScoreValue interface{} = nil // Default to null
		status := ""
		if latestEnsembleScore != nil {
			compositeScoreValue = latestEnsembleScore.Score
		} else {
			// If no ensemble score exists, explicitly set status
			status = "scoring_unavailable"
		}

		resp := map[string]interface{}{
			"composite_score": compositeScoreValue,
			"results":         individualResults,
		}
		// Add status only if it's set (i.e., no ensemble score found)
		if status != "" {
			resp["status"] = status
		}

		// Cache the result for 30 seconds
		articlesCacheLock.Lock()
		articlesCache.Set(cacheKey, resp, 30*time.Second)
		articlesCacheLock.Unlock()

		// DEBUG: Log the response being sent, especially for article 1646
		if id == 1646 {
			log.Printf("[biasHandler DEBUG 1646] Sending response: %+v", resp)
		}

		RespondSuccess(c, resp)
		LogPerformance("biasHandler", start)
	}
}

// @Summary Get ensemble scoring details
// @Description Retrieves individual model results and aggregation for an article's ensemble score
// @Tags Analysis
// @Param id path int true "Article ID" minimum(1)
// @Success 200 {object} StandardResponse
// @Failure 404 {object} ErrorResponse "Ensemble data not found"
// @Router /api/articles/{id}/ensemble [get]
// @ID getArticleEnsemble
func ensembleDetailsHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		id, ok := getValidArticleID(c)
		if !ok {
			return
		}

		// Skip cache if _t query param exists (cache busting)
		if _, skipCache := c.GetQuery("_t"); skipCache {
			log.Printf("[ensembleDetailsHandler] Cache busting requested for article %d", id)
			scores, err := db.FetchLLMScores(dbConn, int64(id))
			if err != nil {
				RespondError(c, NewAppError(ErrInternal, "Failed to fetch ensemble data"))
				LogError(c, err, "ensembleDetailsHandler: fetch scores")
				return
			}
			details := processEnsembleScores(scores)
			if len(details) == 0 {
				RespondError(c, NewAppError(ErrNotFound, "Ensemble data not found"))
				return
			}
			RespondSuccess(c, gin.H{"scores": details})
			LogPerformance("ensembleDetailsHandler (cache bust)", start)
			return
		}

		// Regular caching logic
		cacheKey := "ensemble:" + strconv.FormatInt(id, 10)
		articlesCacheLock.RLock()
		if cachedRaw, found := articlesCache.Get(cacheKey); found {
			articlesCacheLock.RUnlock()
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"scores":  cachedRaw,
			})
			LogPerformance("ensembleDetailsHandler (cache hit)", start)
			return
		}
		articlesCacheLock.RUnlock()

		scores, err := db.FetchLLMScores(dbConn, int64(id))
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch ensemble data"))
			LogError(c, err, "ensembleDetailsHandler: fetch scores")
			return
		}

		details := processEnsembleScores(scores)
		if len(details) == 0 {
			RespondError(c, NewAppError(ErrNotFound, "Ensemble data not found"))
			LogPerformance("ensembleDetailsHandler", start)
			return
		}

		// Add debug logging to help troubleshoot
		log.Printf("[ensembleDetailsHandler] Found %d ensemble records for article %d", len(details), id)
		for i, detail := range details {
			subResults, ok := detail["sub_results"].([]map[string]interface{})
			if !ok {
				subResults = nil
			}
			numResults := 0
			if subResults != nil {
				numResults = len(subResults)
			}
			log.Printf("[ensembleDetailsHandler] Ensemble #%d: score=%.2f, has %d sub_results",
				i+1, detail["score"], numResults)
		}

		articlesCacheLock.Lock()
		articlesCache.Set(cacheKey, details, 30*time.Second)
		articlesCacheLock.Unlock()

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"scores":  details,
		})
		LogPerformance("ensembleDetailsHandler", start)
	}
}

// Helper function to process ensemble scores
func processEnsembleScores(scores []db.LLMScore) []map[string]interface{} {
	details := make([]map[string]interface{}, 0)
	for _, score := range scores {
		if score.Model != ModelEnsemble {
			continue
		}

		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(score.Metadata), &meta); err != nil {
			log.Printf("[ensembleDetailsHandler] Error unmarshalling metadata for score ID %d: %v", score.ID, err)
			details = append(details, map[string]interface{}{
				"score":       score.Score,
				"sub_results": []interface{}{},
				"aggregation": map[string]interface{}{},
				"created_at":  score.CreatedAt,
				"error":       "Metadata parsing failed",
			})
			continue
		}

		// Process sub-results to make sure they're properly formatted
		subResults, ok := meta["sub_results"].([]interface{})
		if !ok {
			subResults = []interface{}{}
		}
		processedResults := make([]map[string]interface{}, 0, len(subResults))

		// Process each sub-result individually to ensure proper format
		for _, sr := range subResults {
			srMap, ok := sr.(map[string]interface{})
			if !ok {
				continue // Skip invalid entries
			}

			// Ensure all required fields exist with proper types
			model, _ := srMap["model"].(string)
			scoreVal, ok1 := srMap["score"].(float64)
			confidence, ok2 := srMap["confidence"].(float64)
			explanation, _ := srMap["explanation"].(string)
			perspective, _ := srMap["perspective"].(string)

			// Default values if not found or invalid
			if !ok1 {
				scoreVal = 0.0
			}
			if !ok2 {
				confidence = 0.0
			}
			if perspective == "" {
				perspective = "unknown"
			}

			processedResults = append(processedResults, map[string]interface{}{
				"model":       model,
				"score":       scoreVal,
				"confidence":  confidence,
				"explanation": explanation,
				"perspective": perspective,
			})
		}

		aggregation, ok2 := meta["final_aggregation"].(map[string]interface{})
		if !ok2 {
			aggregation = map[string]interface{}{}
		}

		details = append(details, map[string]interface{}{
			"score":       score.Score,
			"sub_results": processedResults,
			"aggregation": aggregation,
			"created_at":  score.CreatedAt,
		})
	}
	return details
}

// @Summary Submit user feedback
// @Description Submit user feedback on an article's political bias analysis
// @Tags Feedback
// @Accept json
// @Produce json
// @Param request body FeedbackRequest true "Feedback information"
// @Success 200 {object} StandardResponse "Feedback received"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /api/feedback [post]
// @ID submitFeedback
func feedbackHandler(dbConn *sqlx.DB, llmClient *llm.LLMClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		var req struct {
			ArticleID        int64  `json:"article_id" form:"article_id"`
			UserID           string `json:"user_id" form:"user_id"`
			FeedbackText     string `json:"feedback_text" form:"feedback_text"`
			Category         string `json:"category" form:"category"`
			EnsembleOutputID *int64 `json:"ensemble_output_id" form:"ensemble_output_id"`
			Source           string `json:"source" form:"source"`
		}

		if err := c.ShouldBind(&req); err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid or missing feedback fields"))
			return
		}

		var missingFields []string
		if req.ArticleID == 0 {
			missingFields = append(missingFields, "article_id")
		}
		if req.FeedbackText == "" {
			missingFields = append(missingFields, "feedback_text")
		}
		if req.UserID == "" {
			missingFields = append(missingFields, "user_id")
		}
		if len(missingFields) > 0 {
			RespondError(c, NewAppError(ErrValidation, "Missing required fields: "+strings.Join(missingFields, ", ")))
			return
		}

		validCategories := map[string]bool{"agree": true, "disagree": true, "unclear": true, "other": true, "": true}
		if req.Category != "" && !validCategories[req.Category] {
			RespondError(c, ErrInvalidCategory)
			return
		}

		feedback := &db.Feedback{
			ArticleID:        req.ArticleID,
			UserID:           req.UserID,
			FeedbackText:     req.FeedbackText,
			Category:         req.Category,
			EnsembleOutputID: req.EnsembleOutputID,
			Source:           req.Source,
			CreatedAt:        time.Now(),
		}

		// Insert feedback
		err := db.InsertFeedback(dbConn, feedback)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, fmt.Sprintf("Failed to store feedback: %v", err)))
			return
		}

		// Update article confidence based on feedback
		scores, err := db.FetchLLMScores(dbConn, req.ArticleID)
		if err == nil {
			// Get config from the LLMClient associated with the handler
			config := llmClient.GetConfig()
			if config == nil {
				LogError(c, fmt.Errorf("LLM client config not loaded"), "feedbackHandler: LLM client config not loaded")
				RespondError(c, NewAppError(ErrInternal, "Internal processing error [config]"))
				return
			}

			score, confidence, err := llm.ComputeCompositeScoreWithConfidence(scores, config)
			if err != nil {
				log.Printf("[API DEBUG] Error computing composite score for article %d: %v", req.ArticleID, err)
			} else {
				// Adjust confidence based on feedback category
				if req.Category == "agree" {
					confidence = math.Min(1.0, confidence+0.1) // Increase confidence on agreement
				} else if req.Category == "disagree" {
					confidence = math.Max(0.0, confidence-0.1) // Decrease confidence on disagreement
				}

				// Update article with new confidence
				err = db.UpdateArticleScore(dbConn, req.ArticleID, score, confidence)
				if err != nil {
					// Log error but don't fail the request since feedback was saved
					LogError(c, err, "feedbackHandler: update article confidence")
				}
			}
		}

		RespondSuccess(c, map[string]string{"status": "feedback received"})
		LogPerformance("feedbackHandler", start)
	}
}

// @Summary Manually set article score
// @Description Updates an article's bias score manually
// @Tags Analysis
// @Param id path int true "Article ID" minimum(1)
// @Param request body ManualScoreRequest true "Score value between -1.0 and 1.0"
// @Success 200 {object} StandardResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/manual-score/{id} [post]
// @ID addManualScore
func manualScoreHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := getValidArticleID(c)
		if !ok {
			return
		}
		articleID := id

		// Read raw body for strict validation
		var raw map[string]interface{}
		var err error // Declare err at the top
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&raw); err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid JSON body"))
			LogError(c, err, "manualScoreHandler: invalid JSON body")
			return
		}
		// Only "score" is allowed
		if len(raw) != 1 || raw["score"] == nil {
			RespondError(c, NewAppError(ErrValidation, "Payload must contain only 'score' field"))
			LogError(c, nil, "manualScoreHandler: payload missing or has extra fields")
			return
		}
		// Validate score type and range
		scoreVal, ok := raw["score"].(float64)
		if !ok {
			// Accept integer as well
			if intVal, okInt := raw["score"].(int); okInt {
				scoreVal = float64(intVal)
			} else {
				RespondError(c, NewAppError(ErrValidation, "'score' must be a number"))
				LogError(c, nil, "manualScoreHandler: score not a number")
				return
			}
		}
		if scoreVal < -1.0 || scoreVal > 1.0 {
			RespondError(c, NewAppError(ErrValidation, "Score must be between -1.0 and 1.0"))
			LogError(c, nil, "manualScoreHandler: score out of range")
			return
		}

		_, err = db.FetchArticleByID(dbConn, articleID)
		if err != nil {
			if errors.Is(err, db.ErrArticleNotFound) {
				RespondError(c, NewAppError(ErrNotFound, "Article not found"))
				return

			}
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch article"))
			LogError(c, err, "manualScoreHandler: failed to fetch article")
			return
		}

		// Update score in DB
		err = db.UpdateArticleScore(dbConn, articleID, scoreVal, 1.0) // Set confidence to 1.0 for manual scores
		if err != nil {
			errMsg := err.Error()
			if errMsg != "" && (strings.Contains(errMsg, "constraint failed") ||
				strings.Contains(errMsg, "UNIQUE constraint failed") ||
				strings.Contains(errMsg, "CHECK constraint failed") ||
				strings.Contains(errMsg, "NOT NULL constraint failed") ||
				strings.Contains(errMsg, "FOREIGN KEY constraint failed") ||
				strings.Contains(errMsg, "invalid") ||
				strings.Contains(errMsg, "out of range") ||
				strings.Contains(errMsg, "data type mismatch")) {
				log.Printf("[manualScoreHandler] Constraint/validation error updating article score: %v", err)
				RespondError(c, NewAppError(ErrValidation, "Failed to update score due to invalid data or constraint violation"))
				return
			}
			log.Printf("[manualScoreHandler] Unexpected DB error updating article score: %v", err)
			RespondError(c, NewAppError(ErrInternal, "Failed to update article score"))
			LogError(c, err, "manualScoreHandler: failed to update article score")
			return
		}
		safeLogf("[manualScoreHandler] Article score updated successfully: articleID=%d, score=%f", articleID, scoreVal)
		RespondSuccess(c, map[string]interface{}{
			"status":     "score updated",
			"article_id": articleID,
			"score":      scoreVal,
		})
	}
}

// sanitizeForLog sanitizes user input to prevent log injection attacks
// It removes or escapes potentially dangerous characters that could be used for log injection
func sanitizeForLog(input string) string {
	// Remove newlines and carriage returns to prevent log injection
	re := regexp.MustCompile(`[\r\n\t]`)
	sanitized := re.ReplaceAllString(input, "_")

	// Limit length to prevent log spam
	if len(sanitized) > 100 {
		sanitized = sanitized[:100] + "..."
	}

	return sanitized
}

// safeLogf is a secure logging function that sanitizes user input
func safeLogf(format string, args ...interface{}) {
	// Sanitize the format string to prevent log injection through format parameter
	sanitizedFormat := sanitizeForLog(format)

	// Sanitize string arguments that might contain user input
	for i, arg := range args {
		if str, ok := arg.(string); ok {
			args[i] = sanitizeForLog(str)
		}
	}
	log.Printf(sanitizedFormat, args...)
}
