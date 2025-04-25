package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
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
func RegisterRoutes(router *gin.Engine, dbConn *sqlx.DB, rssCollector *rss.Collector, llmClient *llm.LLMClient, scoreManager *llm.ScoreManager) {
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
	router.GET("/api/articles/:id/summary", SafeHandler(summaryHandler(dbConn)))

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
	router.POST("/api/feedback", SafeHandler(feedbackHandler(dbConn)))

	// Health checks
	// @Summary Feed health check
	// @Description Check the health status of RSS feeds
	// @Tags Health
	// @Accept json
	// @Produce json
	// @Success 200 {object} StandardResponse
	// @Router /feeds/healthz [get]
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
			RespondError(c, ErrDuplicateURL)
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
				RespondError(c, ErrDuplicateURL)
				return
			}
			RespondError(c, WrapError(err, ErrInternal, "Failed to create article"))
			return
		}

		RespondSuccess(c, map[string]interface{}{
			"status":     "created",
			"article_id": id,
		})
	}
}

// Utility function for handling article array results
type ArticleResult struct {
	Article  *db.Article            `json:"article"`
	Scores   []db.LLMScore          `json:"scores"`
	Metadata map[string]interface{} `json:"metadata"`
}

// handleArticleBatch processes a batch of articles
func handleArticleBatch(dbConn *sqlx.DB, articles []db.Article) ([]*ArticleResult, error) {
	results := make([]*ArticleResult, 0, len(articles))
	// ...existing code...
	return results, nil
}

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

		log.Printf("[INFO] getArticlesHandler: Successfully fetched %d articles", len(articles))
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    articles,
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

		// Get latest ensemble score and confidence
		ensembleScore, scoreErr := db.FetchLatestEnsembleScore(dbConn, id)
		if scoreErr != nil {
			log.Printf("[getArticleByIDHandler] Error fetching latest ensemble score for article %d: %v", id, scoreErr)
			ensembleScore = 0.0
		}

		confidence, confErr := db.FetchLatestConfidence(dbConn, id)
		if confErr != nil {
			log.Printf("[getArticleByIDHandler] Error fetching confidence for article %d: %v", id, confErr)
			confidence = 0.0
		}

		// Get all scores for detailed view
		scores, _ := db.FetchLLMScores(dbConn, id)

		result := map[string]interface{}{
			"article":         article,
			"scores":          scores,
			"composite_score": ensembleScore,
			"confidence":      confidence,
			"score_source":    article.ScoreSource,
		}

		// Cache the result for 30 seconds
		articlesCacheLock.Lock()
		articlesCache.Set(cacheKey, result, 30*time.Second)
		articlesCacheLock.Unlock()

		RespondSuccess(c, result)
		LogPerformance("getArticleByIDHandler", start)
	}
}

// @Summary Trigger RSS feed refresh
// @Description Initiates a manual RSS feed refresh job
// @Tags Feeds
// @Success 200 {object} StandardResponse "Refresh started"
// @Router /api/refresh [post]
func refreshHandler(rssCollector *rss.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		go rssCollector.ManualRefresh()
		RespondSuccess(c, map[string]string{"status": "refresh started"})
		LogPerformance("refreshHandler", start)
	}
}

// Refactored reanalyzeHandler to use ScoreManager for scoring, storage, and progress
// @Summary Reanalyze article
// @Description Initiates a reanalysis of an article's political bias or directly updates the score
// @Tags Analysis
// @Accept json
// @Produce json
// @Param id path int true "Article ID" minimum(1)
// @Param request body ManualScoreRequest false "Optional score to set directly"
// @Success 200 {object} StandardResponse "Success - reanalysis queued or score updated"
// @Failure 400 {object} ErrorResponse "Invalid article ID or score"
// @Failure 404 {object} ErrorResponse "Article not found"
// @Failure 429 {object} ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} ErrorResponse "Internal server error or LLM service unavailable"
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

		cfg, cfgErr := llm.LoadCompositeScoreConfig()
		if cfgErr != nil || len(cfg.Models) == 0 {
			RespondError(c, ErrLLMUnavailable)
			return
		}

		modelName := cfg.Models[0].ModelName
		originalTimeout := 10 * time.Second
		llmClient.SetHTTPLLMTimeout(2 * time.Second)
		_, healthErr := llmClient.ScoreWithModel(article, modelName)
		llmClient.SetHTTPLLMTimeout(originalTimeout)

		if healthErr != nil {
			if errors.Is(healthErr, llm.ErrBothLLMKeysRateLimited) {
				RespondError(c, ErrRateLimited)
				return
			}
			if errors.Is(healthErr, llm.ErrLLMServiceUnavailable) {
				RespondError(c, ErrLLMUnavailable)
				return
			}
			RespondError(c, WrapError(healthErr, ErrLLMService, "LLM provider error"))
			return
		}

		// Initial progress state
		if scoreManager != nil {
			scoreManager.SetProgress(articleID, &models.ProgressState{
				Step:        "Starting",
				Message:     "Scoring job queued",
				Percent:     0,
				Status:      "InProgress",
				LastUpdated: time.Now().Unix(),
			})
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errMsg := fmt.Sprintf("Internal panic: %v", r)
					setProgress(articleID, &models.ProgressState{
						Step:        "Error",
						Message:     "Internal error occurred",
						Percent:     0,
						Status:      "Error",
						Error:       errMsg,
						LastUpdated: time.Now().Unix(),
					})
					log.Printf("[Goroutine Panic] ArticleID=%d: %s", articleID, errMsg)
				}
			}()

			totalSteps := len(cfg.Models) + 3
			stepNum := 1

			if scoreManager != nil {
				scoreManager.SetProgress(articleID, &models.ProgressState{
					Step:        "Preparing",
					Message:     "Deleting old scores",
					Percent:     percent(stepNum, totalSteps),
					Status:      "InProgress",
					LastUpdated: time.Now().Unix(),
				})
			}
			if err := llmClient.DeleteScores(articleID); err != nil {
				errMsg := fmt.Sprintf("Failed to delete old scores: %v", err)
				setProgress(articleID, &models.ProgressState{
					Step:        "Error",
					Message:     errMsg,
					Percent:     percent(stepNum, totalSteps),
					Status:      "Error",
					Error:       errMsg,
					LastUpdated: time.Now().Unix(),
				})
				log.Printf("[SetProgress] ArticleID=%d: %s", articleID, errMsg)
				return
			}
			stepNum++

			for _, m := range cfg.Models {
				label := fmt.Sprintf("Scoring with %s", m.ModelName)
				if scoreManager != nil {
					scoreManager.SetProgress(articleID, &models.ProgressState{
						Step:        label,
						Message:     label,
						Percent:     percent(stepNum, totalSteps),
						Status:      "InProgress",
						LastUpdated: time.Now().Unix(),
					})
				}
				_, scoreErr := llmClient.ScoreWithModel(article, m.ModelName)
				if scoreErr != nil {
					userMsg := fmt.Sprintf("Error scoring with %s", m.ModelName)
					if errors.Is(scoreErr, llm.ErrBothLLMKeysRateLimited) {
						userMsg = "Rate limit exceeded"
					}
					setProgress(articleID, &models.ProgressState{
						Step:        "Error",
						Message:     userMsg,
						Percent:     percent(stepNum, totalSteps),
						Status:      "Error",
						Error:       scoreErr.Error(),
						LastUpdated: time.Now().Unix(),
					})
					log.Printf("[SetProgress] ArticleID=%d: %s", articleID, userMsg)
					return
				}
				stepNum++
			}

			if scoreManager != nil {
				scoreManager.SetProgress(articleID, &models.ProgressState{
					Step:        "Calculating",
					Message:     "Computing final score",
					Percent:     percent(stepNum, totalSteps),
					Status:      "InProgress",
					LastUpdated: time.Now().Unix(),
					FinalScore:  nil,
				})
			}
			scores, fetchErr := llmClient.FetchScores(articleID)
			if fetchErr != nil {
				errMsg := fmt.Sprintf("Failed to fetch scores: %v", fetchErr)
				setProgress(articleID, &models.ProgressState{
					Step:        "Error",
					Message:     errMsg,
					Percent:     percent(stepNum, totalSteps),
					Status:      "Error",
					Error:       errMsg,
					LastUpdated: time.Now().Unix(),
				})
				log.Printf("[SetProgress] ArticleID=%d: %s", articleID, errMsg)
				return
			}
			stepNum++

			if scoreManager != nil {
				scoreManager.SetProgress(articleID, &models.ProgressState{
					Step:        "Storing",
					Message:     "Saving results",
					Percent:     percent(stepNum, totalSteps),
					Status:      "InProgress",
					LastUpdated: time.Now().Unix(),
				})
			}
			finalScore, confidence, storeErr := scoreManager.UpdateArticleScore(articleID, scores, cfg)
			if storeErr != nil {
				errMsg := fmt.Sprintf("Failed to store score: %v", storeErr)
				setProgress(articleID, &models.ProgressState{
					Step:        "Error",
					Message:     errMsg,
					Percent:     percent(stepNum, totalSteps),
					Status:      "Error",
					Error:       errMsg,
					LastUpdated: time.Now().Unix(),
				})
				log.Printf("[SetProgress] ArticleID=%d: %s", articleID, errMsg)
				return
			}

			// Final success state with composite score
			confidencePercent := int(confidence * 100)
			message := fmt.Sprintf("Scoring complete (confidence: %d%%)", confidencePercent)
			setProgress(articleID, &models.ProgressState{
				Step:        "Complete",
				Message:     message,
				Percent:     100,
				Status:      "Success",
				FinalScore:  &finalScore,
				LastUpdated: time.Now().Unix(),
			})
			log.Printf("[SetProgress] ArticleID=%d: %s", articleID, message)
		}()

		RespondSuccess(c, gin.H{
			"status":     "reanalyze queued",
			"article_id": articleID,
		})
	}
}

// @Summary Score progress SSE stream
// @Description Server-Sent Events endpoint streaming scoring progress for an article
// @Tags Analysis
// @Param id path int true "Article ID" minimum(1)
// @Success 200 {string} string "event-stream"
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
						fmt.Fprintf(c.Writer, "data: %s\n\n", data)
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

// @Summary Get feed health status
// @Description Returns the health of configured RSS feed sources
// @Tags Feeds
// @Success 200 {object} map[string]bool
// @Router /api/feeds/healthz [get]
func feedHealthHandler(rssCollector *rss.Collector) gin.HandlerFunc {
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
func summaryHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		_, err = db.FetchArticleByID(dbConn, id)
		if err != nil {
			if errors.Is(err, db.ErrArticleNotFound) {
				RespondError(c, ErrArticleNotFound)
				return
			}
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch article"))
			return
		}

		scores, err := db.FetchLLMScores(dbConn, id)
		if err != nil {
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch article summary"))
			return
		}

		for _, score := range scores {
			if score.Model == "summarizer" {
				result := map[string]interface{}{
					"summary":    score.Metadata,
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
}

func parseArticleID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidArticleID})
		return 0, false
	}
	return int64(id), true
}

func filterAndTransformScores(scores []db.LLMScore, min, max float64) []map[string]interface{} {
	results := make([]map[string]interface{}, 0, len(scores))
	for _, score := range scores {
		if score.Model != "ensemble" {
			continue
		}
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(score.Metadata), &meta); err != nil {
			continue
		}
		agg, _ := meta["aggregation"].(map[string]interface{})
		weightedMean, _ := agg["weighted_mean"].(float64)
		if weightedMean < min || weightedMean > max {
			continue
		}
		result := map[string]interface{}{
			"score":      weightedMean,
			"metadata":   meta,
			"created_at": score.CreatedAt,
		}
		results = append(results, result)
	}
	return results
}

func sortResults(results []map[string]interface{}, order string) {
	if order == "asc" {
		sort.Slice(results, func(i, j int) bool {
			return results[i]["score"].(float64) < results[j]["score"].(float64)
		})
	} else {
		sort.Slice(results, func(i, j int) bool {
			return results[i]["score"].(float64) > results[j]["score"].(float64)
		})
	}
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
		sortOrder := c.DefaultQuery("sort", "desc")
		if sortOrder != "asc" && sortOrder != "desc" {
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

			if score.Model == "ensemble" {
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
			scoreI := individualResults[i]["score"].(float64)
			scoreJ := individualResults[j]["score"].(float64)
			if sortOrder == "asc" {
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
		if score.Model != "ensemble" {
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
func feedbackHandler(dbConn *sqlx.DB) gin.HandlerFunc {
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
			// Calculate new composite score and confidence
			score, confidence, compErr := llm.ComputeCompositeScoreWithConfidence(scores)
			if compErr != nil {
				LogError("feedbackHandler: composite score calculation", compErr)
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
		err = db.UpdateArticleScore(dbConn, articleID, scoreVal, 0)
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
