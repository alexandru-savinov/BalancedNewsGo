package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
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
var (
	progressMap     = make(map[int64]*models.ProgressState)
	progressMapLock sync.RWMutex
)

func setProgress(articleID int64, state *models.ProgressState) {
	progressMapLock.Lock()
	defer progressMapLock.Unlock()
	progressMap[articleID] = state
	log.Printf("[SetProgress] ArticleID=%d Status=%s Step=%s Message=%s",
		articleID, state.Status, state.Step, state.Message)
}

func getProgress(articleID int64) *models.ProgressState {
	progressMapLock.RLock()
	defer progressMapLock.RUnlock()
	if p, ok := progressMap[articleID]; ok {
		return p
	}
	return nil
}

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
	// @Success 200 {array} models.Article
	// @Failure 500 {object} ErrorResponse
	// @Router /articles [get]
	router.GET("/api/articles", SafeHandler(getArticlesHandler(dbConn)))

	// @Summary Get article by ID
	// @Description Get detailed information about a specific article
	// @Tags Articles
	// @Accept json
	// @Produce json
	// @Param id path integer true "Article ID"
	// @Success 200 {object} models.Article
	// @Failure 404 {object} ErrorResponse
	// @Router /articles/{id} [get]
	router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(dbConn)))

	// @Summary Create article
	// @Description Create a new article
	// @Tags Articles
	// @Accept json
	// @Produce json
	// @Param article body models.CreateArticleRequest true "Article object"
	// @Success 200 {object} StandardResponse{data=models.Article}
	// @Failure 400 {object} ErrorResponse
	// @Router /articles [post]
	router.POST("/api/articles", SafeHandler(createArticleHandler(dbConn)))

	// Feed management
	// @Summary Refresh feeds
	// @Description Trigger a refresh of all RSS feeds
	// @Tags Feeds
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Failure 500 {object} ErrorResponse
	// @Router /refresh [post]
	router.POST("/api/refresh", SafeHandler(refreshHandler(rssCollector)))

	// LLM Analysis
	// @Summary Reanalyze article
	// @Description Trigger a new LLM analysis for a specific article
	// @Tags LLM
	// @Accept json
	// @Produce json
	// @Param id path integer true "Article ID"
	// @Success 202 {object} StandardResponse
	// @Failure 404 {object} ErrorResponse
	// @Router /llm/reanalyze/{id} [post]
	router.POST("/api/llm/reanalyze/:id", SafeHandler(reanalyzeHandler(llmClient, dbConn, scoreManager)))

	// Scoring
	// @Summary Add manual score
	// @Description Add a manual bias score for an article
	// @Tags Scoring
	// @Accept json
	// @Produce json
	// @Param id path integer true "Article ID"
	// @Param score body models.ManualScoreRequest true "Score information"
	// @Success 200 {object} StandardResponse
	// @Failure 400 {object} ErrorResponse
	// @Router /manual-score/{id} [post]
	router.POST("/api/manual-score/:id", SafeHandler(manualScoreHandler(dbConn)))

	// Article analysis
	// @Summary Get article summary
	// @Description Get the summary analysis for an article
	// @Tags Analysis
	// @Accept json
	// @Produce json
	// @Param id path integer true "Article ID"
	// @Success 200 {object} models.SummaryResponse
	// @Failure 404 {object} ErrorResponse
	// @Router /articles/{id}/summary [get]
	handler := NewSummaryHandler(&db.DBInstance{DB: dbConn})
	router.GET("/api/articles/:id/summary", SafeHandler(handler.Handle))

	// @Summary Get bias analysis
	// @Description Get the bias analysis for an article
	// @Tags Analysis
	// @Accept json
	// @Produce json
	// @Param id path integer true "Article ID"
	// @Success 200 {object} models.BiasResponse
	// @Failure 404 {object} ErrorResponse
	// @Router /articles/{id}/bias [get]
	router.GET("/api/articles/:id/bias", SafeHandler(biasHandler(dbConn)))

	// @Summary Get ensemble details
	// @Description Get detailed ensemble analysis results for an article
	// @Tags Analysis
	// @Param id path integer true "Article ID"
	// @Success 200 {object} models.EnsembleResponse
	// @Failure 404 {object} ErrorResponse
	// @Router /articles/{id}/ensemble [get]
	router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(dbConn)))

	// Feedback
	// @Summary Submit feedback
	// @Description Submit user feedback for an article analysis
	// @Tags Feedback
	// @Accept json
	// @Produce json
	// @Param feedback body models.FeedbackRequest true "Feedback information"
	// @Success 200 {object} StandardResponse
	// @Failure 400 {object} ErrorResponse
	// @Router /feedback [post]
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
	router.GET("/api/feeds/healthz", SafeHandler(feedHealthHandler(rssCollector)))

	// Progress tracking
	// @Summary Score progress
	// @Description Get real-time progress updates for article scoring
	// @Tags LLM
	// @Accept json
	// @Produce text/event-stream
	// @Param id path integer true "Article ID"
	// @Success 200 {object} models.ProgressResponse
	// @Router /llm/score-progress/{id} [get]
	router.GET("/api/llm/score-progress/:id", SafeHandler(scoreProgressSSEHandler()))
}

// SafeHandler wraps a handler function with panic recovery to prevent server crashes
func SafeHandler(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic
				log.Printf("[PANIC RECOVERED] %v\n%s", r, string(debug.Stack()))
				// Return an error response
				RespondError(c, NewAppError(ErrInternal, fmt.Sprintf("Internal server error: %v", r)))
			}
		}()
		handler(c)
	}
}

// Helper: Convert db.Article to Postman schema (TitleCase fields)
func articleToPostmanSchema(a *db.Article) map[string]interface{} {
	return map[string]interface{}{
		"article_id":     a.ID,
		"Title":          a.Title,
		"Content":        a.Content,
		"URL":            a.URL,
		"Source":         a.Source,
		"CompositeScore": a.CompositeScore,
		"Confidence":     a.Confidence,
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
// @Router /articles [post]
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

		// Check if article already exists
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

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data":    articleToPostmanSchema(createdArticle),
		})
	}
}

// Utility function for handling article array results
// ... remove handleArticleBatch function ...

// getArticlesHandler handles GET /articles
// @Summary Get articles
// @Description Fetches a list of articles with optional filters
// @Tags Articles
// @Accept json
// @Produce json
// @Param source query string false "Filter by source (e.g., CNN, Fox)"
// @Param leaning query string false "Filter by political leaning"
// @Param limit query int false "Maximum number of articles to return" default(20) minimum(1) maximum(100)
// @Param offset query int false "Number of articles to skip" default(0) minimum(0)
// @Success 200 {object} StandardResponse{data=[]db.Article} "Success"
// @Failure 400 {object} ErrorResponse "Invalid parameters"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /articles [get]
func getArticlesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		source := c.Query("source")
		leaning := c.Query("leaning")
		limitStr := c.DefaultQuery("limit", "20")
		offsetStr := c.DefaultQuery("offset", "0")

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			log.Printf("[ERROR] getArticlesHandler: invalid limit parameter: %v", err)
			RespondError(c, NewAppError(ErrValidation, "Invalid 'limit' parameter"))
			return
		}
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			log.Printf("[ERROR] getArticlesHandler: invalid offset parameter: %v", err)
			RespondError(c, NewAppError(ErrValidation, "Invalid 'offset' parameter"))
			return
		}

		log.Printf("[INFO] getArticlesHandler: Fetching articles (source=%s, leaning=%s, limit=%d, offset=%d)", source, leaning, limit, offset)
		articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
		if err != nil {
			log.Printf("[ERROR] getArticlesHandler: Database error fetching articles: %+v", err)
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch articles"))
			return
		}

		// Enhance articles with composite scores and confidence
		for i := range articles {
			scores, _ := db.FetchLLMScores(dbConn, articles[i].ID)
			if len(scores) > 0 {
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
					compositeScore := weightedSum / sumWeights
					avgConfidence := sumWeights / float64(len(scores))
					articles[i].CompositeScore = &compositeScore
					articles[i].Confidence = &avgConfidence
				}
			}
		}

		// Map to Postman schema
		var out []map[string]interface{}
		for i := range articles {
			out = append(out, articleToPostmanSchema(&articles[i]))
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    out,
		})
		LogPerformance("getArticlesHandler", start)
	}
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
// @Router /articles/{id} [get]
func getArticleByIDHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		idStr := c.Param("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id < 1 {
			RespondError(c, ErrInvalidArticleID)
			return
		}

		// Caching
		cacheKey := "article:" + idStr
		articlesCacheLock.RLock()
		if cached, found := articlesCache.Get(cacheKey); found {
			articlesCacheLock.RUnlock()
			RespondSuccess(c, cached)
			LogPerformance("getArticleByIDHandler (cache hit)", start)
			return
		}
		articlesCacheLock.RUnlock()

		article, err := db.FetchArticleByID(dbConn, id)
		if err != nil {
			if errors.Is(err, db.ErrArticleNotFound) {
				RespondError(c, ErrArticleNotFound)
				return
			}
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch article"))
			LogError("getArticleByIDHandler: fetch article", err)
			return
		}

		// Use the same schema as other endpoints
		result := articleToPostmanSchema(article)

		// Cache the result for 30 seconds
		articlesCacheLock.Lock()
		articlesCache.Set(cacheKey, result, 30*time.Second)
		articlesCacheLock.Unlock()

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    result,
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
func refreshHandler(rssCollector rss.CollectorInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		go rssCollector.ManualRefresh()
		RespondSuccess(c, map[string]string{"status": "refresh started"})
		LogPerformance("refreshHandler", start)
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
// @Failure 404 {object} ErrorResponse "Article not found"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /llm/reanalyze/{id} [post]
func reanalyzeHandler(llmClient *llm.LLMClient, dbConn *sqlx.DB, scoreManager *llm.ScoreManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			RespondError(c, ErrInvalidArticleID)
			return
		}
		articleID := int64(id)
		log.Printf("[POST /api/llm/reanalyze] ArticleID=%d", articleID)

		// Check if article exists
		article, err := db.FetchArticleByID(dbConn, articleID)
		if err != nil {
			if errors.Is(err, db.ErrArticleNotFound) {
				RespondError(c, ErrArticleNotFound)
				return
			}
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch article"))
			return
		}

		// Parse raw JSON body
		var raw map[string]interface{}
		if err := c.ShouldBindJSON(&raw); err != nil {
			RespondError(c, ErrInvalidPayload)
			return
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
					RespondError(c, NewAppError(ErrValidation, "Invalid score value"))
					LogError("reanalyzeHandler: invalid score format", parseErr)
					return
				}
			default:
				RespondError(c, NewAppError(ErrValidation, "Invalid score value"))
				LogError("reanalyzeHandler: invalid score value", nil)
				return
			}

			if scoreFloat < -1.0 || scoreFloat > 1.0 {
				RespondError(c, NewAppError(ErrValidation, "Score must be between -1.0 and 1.0"))
				LogError("reanalyzeHandler: invalid score value", nil)
				return
			}

			confidence := 1.0 // Use maximum confidence for direct score updates
			err = db.UpdateArticleScoreLLM(dbConn, articleID, scoreFloat, confidence)
			if err != nil {
				RespondError(c, NewAppError(ErrInternal, "Failed to update article score"))
				LogError("reanalyzeHandler: failed to update article score", err)
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
			RespondError(c, ErrLLMUnavailable)
			return
		}

		// Try each model in sequence until we find one that works
		var workingModel string
		var healthErr error

		if os.Getenv("NO_AUTO_ANALYZE") == "true" {
			log.Printf("[reanalyzeHandler %d] NO_AUTO_ANALYZE is set, skipping working model health check.", articleID)
			if len(cfg.Models) > 0 {
				workingModel = cfg.Models[0].ModelName // Assume the first model would work for queuing purposes
			} else {
				log.Printf("[reanalyzeHandler %d] No models configured, cannot proceed even with NO_AUTO_ANALYZE.", articleID)
				RespondError(c, ErrLLMUnavailable) // Or a more specific error
				return
			}
		} else {
			originalTimeout := 10 * time.Second          // TODO: Make this configurable or use actual client timeout
			llmClient.SetHTTPLLMTimeout(2 * time.Second) // Short timeout for health check

			for _, modelConfig := range cfg.Models {
				log.Printf("[reanalyzeHandler %d] Trying model: %s", articleID, modelConfig.ModelName)
				_, healthErr = llmClient.ScoreWithModel(article, modelConfig.ModelName)
				if healthErr == nil {
					workingModel = modelConfig.ModelName
					break
				}
				// If it's not a rate limit error, don't try other models for health check, assume primary service issue
				if !errors.Is(healthErr, llm.ErrBothLLMKeysRateLimited) {
					break
				}
			}
			llmClient.SetHTTPLLMTimeout(originalTimeout) // Restore original timeout
		}

		if workingModel == "" {
			log.Printf("[reanalyzeHandler %d] No working models found (health check failed or skipped with no models): %v", articleID, healthErr)
			RespondError(c, ErrLLMUnavailable)
			return
		}

		// Start the reanalysis process
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{
				Status:  "InProgress",
				Step:    "Starting",
				Message: fmt.Sprintf("Starting analysis with model %s", workingModel),
			})

			// Check for an environment variable to skip auto-analysis during tests
			if os.Getenv("NO_AUTO_ANALYZE") != "true" {
				go func() {
					err := llmClient.ReanalyzeArticle(articleID)
					if err != nil {
						log.Printf("[reanalyzeHandler %d] Error during reanalysis: %v", articleID, err)
						scoreManager.SetProgress(articleID, &models.ProgressState{
							Status:  "Error",
							Step:    "Error",
							Message: fmt.Sprintf("Error during analysis: %v", err),
						})
						return
					}
					scoreManager.SetProgress(articleID, &models.ProgressState{
						Status:  "Complete",
						Step:    "Done",
						Message: "Analysis complete",
					})
				}()
			} else {
				log.Printf("[reanalyzeHandler %d] NO_AUTO_ANALYZE is set, skipping background reanalysis.", articleID)
				// Optionally, set progress to complete or a specific "skipped" state
				scoreManager.SetProgress(articleID, &models.ProgressState{
					Status:  "Skipped",
					Step:    "Skipped",
					Message: "Automatic reanalysis skipped by test configuration.",
				})
			}
		}

		RespondSuccess(c, map[string]interface{}{
			"status":     "reanalyze queued",
			"article_id": articleID,
		})
	}
}

// @Summary Score progress SSE stream
// @Description Server-Sent Events endpoint streaming real-time scoring progress for an article
// @Tags LLM
// @Accept json
// @Produce text/event-stream
// @Param id path int true "Article ID" minimum(1)
// @Success 200 {object} models.ProgressState "event-stream containing progress updates"
// @Failure 400 {object} ErrorResponse "Invalid article ID"
// @Router /api/llm/score-progress/{id} [get]
func scoreProgressSSEHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
			return
		}
		articleID := int64(id)
		log.Printf("[SSE GET /api/llm/score-progress] ArticleID=%d", articleID)

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		lastProgress := ""
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-ticker.C:
				progress := getProgress(articleID)
				if progress == nil {
					continue
				}

				// Always send updates when status changes or on final states
				if data, err := json.Marshal(progress); err == nil {
					currentProgress := string(data)
					if currentProgress != lastProgress {
						if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
							// Client disconnected or error writing
							return
						}
						c.Writer.Flush()
						lastProgress = currentProgress

						// Close connection on final states
						if progress.Status == "Success" || progress.Status == "Error" {
							return
						}
					}
				}
			}
		}
	}
}

// Helper to calculate percent complete
func percent(step, total int) int {
	if total == 0 {
		return 0
	}
	p := (step * 100) / total
	if p > 100 {
		return 100
	}
	return p
}

// @Summary Get RSS feed health status
// @Description Returns the health status of all configured RSS feeds
// @Tags Feeds
// @Accept json
// @Produce json
// @Success 200 {object} map[string]bool "Feed health status mapping feed names to boolean status"
// @Failure 500 {object} ErrorResponse "Server error"
// @Router /api/feeds/healthz [get]
func feedHealthHandler(rssCollector rss.CollectorInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := rssCollector.CheckFeedHealth()
		c.JSON(200, status)
	}
}

// @Summary Get article summary
// @Description Retrieves the text summary for an article
// @Tags Summary
// @Param id path int true "Article ID" minimum(1)
// @Success 200 {object} StandardResponse
// @Failure 404 {object} ErrorResponse "Summary not available"
// @Router /api/articles/{id}/summary [get]
type SummaryHandler struct {
	db db.DBOperations
}

func NewSummaryHandler(db db.DBOperations) *SummaryHandler {
	return &SummaryHandler{db: db}
}

func (h *SummaryHandler) Handle(c *gin.Context) {
	start := time.Now()
	idStr := c.Param("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 1 {
		RespondError(c, ErrInvalidArticleID)
		return
	}

	// Caching
	cacheKey := "summary:" + idStr
	articlesCacheLock.RLock()
	if cached, found := articlesCache.Get(cacheKey); found {
		articlesCacheLock.RUnlock()
		RespondSuccess(c, cached)
		LogPerformance("summaryHandler (cache hit)", start)
		return
	}
	articlesCacheLock.RUnlock()

	// Verify article exists
	_, err = h.db.FetchArticleByID(c, id)
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
// @Router /articles/{id}/bias [get]
// If no valid LLM scores are available, the API responds with:
//   - "composite_score": null
//   - "status": "scoring_unavailable"
//
// instead of defaulting to zero values.
// This indicates that scoring data is currently unavailable.
func biasHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		idStr := c.Param("id")
		articleID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || articleID < 1 {
			RespondError(c, NewAppError(ErrValidation, "Invalid article ID"))
			LogError("biasHandler: invalid id", err)
			return
		}

		minScore, err := strconv.ParseFloat(c.DefaultQuery("min_score", "-1"), 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid min_score"))
			LogError("biasHandler: invalid min_score", err)
			return
		}
		maxScore, err := strconv.ParseFloat(c.DefaultQuery("max_score", "1"), 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid max_score"))
			LogError("biasHandler: invalid max_score", err)
			return
		}
		sortOrder := c.DefaultQuery("sort", SortDesc)
		if sortOrder != SortAsc && sortOrder != SortDesc {
			RespondError(c, NewAppError(ErrValidation, "Invalid sort order"))
			LogError("biasHandler: invalid sort order", nil)
			return
		}

		// Caching
		cacheKey := "bias:" + idStr + ":" + c.DefaultQuery("min_score", "-1") + ":" + c.DefaultQuery("max_score", "1") + ":" + sortOrder
		articlesCacheLock.RLock()
		if cached, found := articlesCache.Get(cacheKey); found {
			articlesCacheLock.RUnlock()
			RespondSuccess(c, cached)
			LogPerformance("biasHandler (cache hit)", start)
			return
		}
		articlesCacheLock.RUnlock()

		scores, err := db.FetchLLMScores(dbConn, articleID)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch bias data"))
			LogError("biasHandler: fetch scores", err)
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
		if articleID == 1646 {
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
func ensembleDetailsHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			RespondError(c, NewAppError(ErrValidation, "Invalid article ID"))
			LogError("ensembleDetailsHandler: invalid id", err)
			return
		}

		// Skip cache if _t query param exists (cache busting)
		if _, skipCache := c.GetQuery("_t"); skipCache {
			log.Printf("[ensembleDetailsHandler] Cache busting requested for article %d", id)
			scores, err := db.FetchLLMScores(dbConn, int64(id))
			if err != nil {
				RespondError(c, NewAppError(ErrInternal, "Failed to fetch ensemble data"))
				LogError("ensembleDetailsHandler: fetch scores", err)
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
		cacheKey := "ensemble:" + idStr
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
			LogError("ensembleDetailsHandler: fetch scores", err)
			return
		}

		details := processEnsembleScores(scores)
		if len(details) == 0 {
			RespondError(c, NewAppError(ErrNotFound, "Ensemble data not found"))
			LogPerformance("ensembleDetailsHandler", start)
			return
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

		subResults, ok1 := meta["sub_results"].([]interface{})
		if !ok1 {
			subResults = []interface{}{}
		}

		aggregation, ok2 := meta["aggregation"].(map[string]interface{})
		if !ok2 {
			aggregation = map[string]interface{}{}
		}

		details = append(details, map[string]interface{}{
			"score":       score.Score,
			"sub_results": subResults,
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
// @Router /feedback [post]
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
			RespondError(c, ErrInvalidPayload)
			return
		}

		// Validate all required fields
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
			RespondError(c, NewAppError(ErrValidation,
				fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", "))))
			return
		}

		// Validate Category if provided
		validCategories := map[string]bool{
			"agree":    true,
			"disagree": true,
			"unclear":  true,
			"other":    true,
			"":         true, // Allow empty for backward compatibility
		}
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
				LogError("feedbackHandler: LLM client config not loaded", fmt.Errorf("LLM client config not loaded"))
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
					LogError("feedbackHandler: update article confidence", err)
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
func manualScoreHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			RespondError(c, NewAppError(ErrValidation, "Invalid article ID"))
			LogError("manualScoreHandler: invalid id", err)
			return
		}
		articleID := int64(id)

		// Read raw body for strict validation
		var raw map[string]interface{}
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&raw); err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid JSON body"))
			LogError("manualScoreHandler: invalid JSON body", err)
			return
		}
		// Only "score" is allowed
		if len(raw) != 1 || raw["score"] == nil {
			RespondError(c, NewAppError(ErrValidation, "Payload must contain only 'score' field"))
			LogError("manualScoreHandler: payload missing or has extra fields", nil)
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
				LogError("manualScoreHandler: score not a number", nil)
				return
			}
		}
		if scoreVal < -1.0 || scoreVal > 1.0 {
			RespondError(c, NewAppError(ErrValidation, "Score must be between -1.0 and 1.0"))
			LogError("manualScoreHandler: score out of range", nil)
			return
		}

		// Check if article exists
		_, err = db.FetchArticleByID(dbConn, articleID)
		if err != nil {
			if errors.Is(err, db.ErrArticleNotFound) {
				RespondError(c, NewAppError(ErrNotFound, "Article not found"))
				return
			}
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch article"))
			LogError("manualScoreHandler: failed to fetch article", err)
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
			LogError("manualScoreHandler: failed to update article score", err)
			return
		}
		log.Printf("[manualScoreHandler] Article score updated successfully: articleID=%d, score=%f", articleID, scoreVal)
		RespondSuccess(c, map[string]interface{}{
			"status":     "score updated",
			"article_id": articleID,
			"score":      scoreVal,
		})
	}
}

// Helper function to convert string to *string
func strPtr(s string) *string {
	return &s
}
