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
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

var (
	articlesCache     = NewSimpleCache()
	articlesCacheLock sync.RWMutex
)

// Progress tracking types and vars
type ProgressState struct {
	Step        string   `json:"step"`                  // Current detailed step
	Message     string   `json:"message"`               // User-friendly message
	Percent     int      `json:"percent"`               // Progress percentage
	Status      string   `json:"status"`                // Overall status
	Error       string   `json:"error,omitempty"`       // Error message if Status is "Error"
	FinalScore  *float64 `json:"final_score,omitempty"` // Final score if Status is "Success"
	LastUpdated int64    `json:"last_updated"`          // Timestamp
}

var (
	progressMap     = make(map[int64]*ProgressState)
	progressMapLock sync.RWMutex
)

func setProgress(articleID int64, step, message string, percent int, status string, errMsg string, finalScore *float64) {
	progressMapLock.Lock()
	defer progressMapLock.Unlock()
	progressMap[articleID] = &ProgressState{
		Step:        step,
		Message:     message,
		Percent:     percent,
		Status:      status, // Added status
		Error:       errMsg,
		FinalScore:  finalScore, // Added finalScore
		LastUpdated: time.Now().Unix(),
	}
}

func getProgress(articleID int64) *ProgressState {
	progressMapLock.RLock()
	defer progressMapLock.RUnlock()
	if p, ok := progressMap[articleID]; ok {
		return p
	}
	return nil
}

func RegisterRoutes(dbConn *sqlx.DB, rssCollector *rss.Collector, llmClient *llm.LLMClient) *gin.Engine {
	router := gin.Default()

	// Load HTML templates
	router.LoadHTMLGlob("web/*.html")

	// Serve static files from ./web
	router.Static("/static", "./web")

	// API routes
	router.GET("/api/articles", SafeHandler(getArticlesHandler(dbConn)))
	router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(dbConn)))
	router.POST("/api/articles", SafeHandler(createArticleHandler(dbConn)))
	router.POST("/api/refresh", SafeHandler(refreshHandler(rssCollector)))
	router.POST("/api/llm/reanalyze/:id", SafeHandler(reanalyzeHandler(llmClient, dbConn)))
	router.POST("/api/manual-score/:id", SafeHandler(manualScoreHandler(dbConn)))
	router.GET("/api/articles/:id/summary", SafeHandler(summaryHandler(dbConn)))
	router.GET("/api/articles/:id/bias", SafeHandler(biasHandler(dbConn)))
	router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(dbConn)))
	router.POST("/api/feedback", SafeHandler(feedbackHandler(dbConn)))
	router.GET("/api/feeds/healthz", SafeHandler(feedHealthHandler(rssCollector)))
	router.GET("/api/llm/score-progress/:id", SafeHandler(scoreProgressSSEHandler()))

	// Debug endpoints
	router.GET("/api/debug/schema", SafeHandler(debugSchemaHandler(dbConn)))

	return router
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
func createArticleHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Source  string `json:"source"`
			PubDate string `json:"pub_date"`
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
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
		article := &db.Article{
			Source:         req.Source,
			PubDate:        pubDate,
			URL:            req.URL,
			Title:          req.Title,
			Content:        req.Content,
			CreatedAt:      time.Now(),
			CompositeScore: &zero,
			Confidence:     &zero,
			ScoreSource:    "llm",
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
func getArticlesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		source := c.Query("source")
		leaning := c.Query("leaning")
		limitStr := c.DefaultQuery("limit", "20")
		offsetStr := c.DefaultQuery("offset", "0")

		// Input validation
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			RespondError(c, NewAppError(ErrValidation, "Invalid 'limit' parameter"))
			LogError("getArticlesHandler: invalid limit", err)
			return
		}
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			RespondError(c, NewAppError(ErrValidation, "Invalid 'offset' parameter"))
			LogError("getArticlesHandler: invalid offset", err)
			return
		}

		// Caching with improved key generation
		// Ensure source and leaning have default values for consistent cache keys
		sourceKey := source
		if sourceKey == "" {
			sourceKey = "all"
		}
		leaningKey := leaning
		if leaningKey == "" {
			leaningKey = "all"
		}

		cacheKey := fmt.Sprintf("articles:%s:%s:%s:%s", sourceKey, leaningKey, limitStr, offsetStr)
		articlesCacheLock.RLock()
		log.Printf("[getArticlesHandler] Checking cache for key: %s", cacheKey) // DEBUG LOG ADDED
		if cached, found := articlesCache.Get(cacheKey); found {
			articlesCacheLock.RUnlock()
			log.Printf("[getArticlesHandler] Cache HIT for key: %s. Serving cached data.", cacheKey) // DEBUG LOG ADDED
			// Optionally log cached data details if needed, be mindful of log volume
			// log.Printf("[getArticlesHandler] Cached data: %+v", cached)
			RespondSuccess(c, cached)
			LogPerformance("getArticlesHandler (cache hit)", start)
			return
		}
		articlesCacheLock.RUnlock()
		log.Printf("[getArticlesHandler] Cache MISS for key: %s. Fetching from DB.", cacheKey) // DEBUG LOG ADDED

		articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
		// Log fetched data *after* potential error check
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch articles"))
			LogError("getArticlesHandler: fetch articles", err)
			return
		}
		log.Printf("[getArticlesHandler] Fetched %d articles from DB for key: %s", len(articles), cacheKey) // DEBUG LOG ADDED
		if len(articles) > 0 {
			log.Printf("[getArticlesHandler] First article: %+v", articles[0])
		}
		// Optionally log specific article details if needed, e.g., the one being re-analyzed
		// for _, art := range articles { if art.ID == 1680 { log.Printf("[getArticlesHandler] DB Data for Article 1680: %+v", art) } }

		// Improved score fetching with better error handling
		for i := range articles {
			// Fetch the latest ensemble score directly
			ensembleScore, scoreErr := db.FetchLatestEnsembleScore(dbConn, articles[i].ID)
			if scoreErr != nil {
				// Log error fetching the specific ensemble score, but don't block response
				log.Printf("[getArticlesHandler] Error fetching latest ensemble score for article %d: %v", articles[i].ID, scoreErr)

				// Use a default score of 0.0 instead of nil for better consistency
				defaultScore := 0.0
				articles[i].CompositeScore = &defaultScore

				// Also set a default confidence value
				defaultConfidence := 0.0
				articles[i].Confidence = &defaultConfidence
			} else {
				// Take the address of the float64 to assign to *float64
				scoreCopy := ensembleScore // Create a copy to ensure its address is stable
				articles[i].CompositeScore = &scoreCopy

				// Optionally fetch confidence as well if available
				confidence, confErr := db.FetchLatestConfidence(dbConn, articles[i].ID)
				if confErr == nil {
					confCopy := confidence
					articles[i].Confidence = &confCopy
				} else {
					defaultConf := 0.0
					articles[i].Confidence = &defaultConf
				}
			}
			// Optional: Log the fetched ensemble score if needed for debugging
			// log.Printf("[getArticlesHandler] Fetched EnsembleScore for Article %d: %f", articles[i].ID, articles[i].CompositeScore)
		}

		sort.Slice(articles, func(i, j int) bool {
			// Safely dereference pointers for comparison, treat nil as 0
			scoreI := 0.0
			if articles[i].CompositeScore != nil {
				scoreI = *articles[i].CompositeScore
			}
			scoreJ := 0.0
			if articles[j].CompositeScore != nil {
				scoreJ = *articles[j].CompositeScore
			}
			return scoreI > scoreJ
		})

		// Cache the result for 30 seconds
		articlesCacheLock.Lock()
		articlesCache.Set(cacheKey, articles, 30*time.Second)
		articlesCacheLock.Unlock()

		RespondSuccess(c, articles)
		LogPerformance("getArticlesHandler", start)
	}
}

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

func refreshHandler(rssCollector *rss.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		go rssCollector.ManualRefresh()
		RespondSuccess(c, map[string]string{"status": "refresh started"})
		LogPerformance("refreshHandler", start)
	}
}

func reanalyzeHandler(llmClient *llm.LLMClient, dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			RespondError(c, ErrInvalidArticleID)
			return
		}
		articleID := int64(id)

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

		// Check for forbidden score field
		if _, hasScore := raw["score"]; hasScore {
			RespondError(c, NewAppError(ErrValidation, "Payload must not contain 'score' field"))
			return
		}

		// API-first: Pre-flight LLM provider check
		cfg, cfgErr := llm.LoadCompositeScoreConfig()
		if cfgErr != nil || len(cfg.Models) == 0 {
			RespondError(c, ErrLLMUnavailable)
			return
		}

		// Pre-flight health check with short timeout
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

		// Set initial progress: Queued
		setProgress(articleID, "Queued", "Scoring job queued", 0, "InProgress", "", nil)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errMsg := fmt.Sprintf("Internal panic: %v", r)
					log.Printf("[reanalyzeHandler %d] Recovered from panic: %s", articleID, errMsg)
					setProgress(articleID, "Error", "Internal error occurred", 0, "Error", errMsg, nil)
				}
			}()

			// Processing steps
			totalSteps := len(cfg.Models) + 3
			stepNum := 1

			// Step 1: Delete old scores
			setProgress(articleID, "Preparing", "Deleting old scores", percent(stepNum, totalSteps), "InProgress", "", nil)
			if err := llmClient.DeleteScores(articleID); err != nil {
				errMsg := fmt.Sprintf("Failed to delete old scores: %v", err)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}
			stepNum++

			// Step 2: Score with each model
			for _, m := range cfg.Models {
				label := fmt.Sprintf("Scoring with %s", m.ModelName)
				setProgress(articleID, label, label, percent(stepNum, totalSteps), "InProgress", "", nil)

				_, scoreErr := llmClient.ScoreWithModel(article, m.ModelName)
				if scoreErr != nil {
					userMsg := fmt.Sprintf("Error scoring with %s", m.ModelName)
					if errors.Is(scoreErr, llm.ErrBothLLMKeysRateLimited) {
						userMsg = "Rate limit exceeded"
					}
					setProgress(articleID, "Error", userMsg, percent(stepNum, totalSteps), "Error", scoreErr.Error(), nil)
					return
				}
				stepNum++
			}

			// Step 3: Calculate final score
			setProgress(articleID, "Calculating", "Computing final score", percent(stepNum, totalSteps), "InProgress", "", nil)
			scores, fetchErr := llmClient.FetchScores(articleID)
			if fetchErr != nil {
				errMsg := fmt.Sprintf("Failed to fetch scores: %v", fetchErr)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}

			finalScore, _, calcErr := llm.ComputeCompositeScoreWithConfidence(scores)
			if calcErr != nil {
				errMsg := fmt.Sprintf("Failed to calculate score: %v", calcErr)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}
			stepNum++

			// Step 4: Store results
			setProgress(articleID, "Storing", "Saving results", percent(stepNum, totalSteps), "InProgress", "", nil)
			actualScore, storeErr := llmClient.StoreEnsembleScore(article)
			if storeErr != nil {
				errMsg := fmt.Sprintf("Failed to store score: %v", storeErr)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, &finalScore)
				return
			}

			// Success: Clear cache and set final progress
			articlesCacheLock.Lock()
			for _, key := range []string{
				fmt.Sprintf("article:%d", articleID),
				fmt.Sprintf("ensemble:%d", articleID),
				fmt.Sprintf("bias:%d", articleID),
			} {
				articlesCache.Delete(key)
			}
			articlesCacheLock.Unlock()

			setProgress(articleID, "Complete", "Scoring complete", 100, "Success", "", &actualScore)
		}()

		RespondSuccess(c, gin.H{
			"status":     "reanalyze queued",
			"article_id": articleID,
		})
	}
}

// SSE handler for progress updates
func scoreProgressSSEHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
			return
		}
		articleID := int64(id)

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		lastStep := ""
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
				if progress.Step != lastStep || progress.Error != "" {
					data, _ := json.Marshal(progress)
					fmt.Fprintf(c.Writer, "data: %s\n\n", data)
					c.Writer.Flush()
					lastStep = progress.Step
					if progress.Step == "Complete" || progress.Error != "" {
						return
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

// feedHealthHandler returns the health status of all feed sources.
func feedHealthHandler(rssCollector *rss.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := rssCollector.CheckFeedHealth()
		c.JSON(200, status)
	}
}

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
	results := make([]map[string]interface{}, 0)
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
// If no valid LLM scores are available, the API responds with:
//   - "composite_score": null
//   - "status": "scoring_unavailable"
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

		minScore, err := strconv.ParseFloat(c.DefaultQuery("min_score", "-2"), 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid min_score"))
			LogError("biasHandler: invalid min_score", err)
			return
		}
		maxScore, err := strconv.ParseFloat(c.DefaultQuery("max_score", "2"), 64)
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
		cacheKey := "bias:" + idStr + ":" + c.DefaultQuery("min_score", "-2") + ":" + c.DefaultQuery("max_score", "2") + ":" + sortOrder
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
			compositeScore, confidence, _ := llm.ComputeCompositeScoreWithConfidence(scores)

			// Adjust confidence based on feedback category
			if req.Category == "agree" {
				confidence = math.Min(1.0, confidence+0.1) // Increase confidence on agreement
			} else if req.Category == "disagree" {
				confidence = math.Max(0.0, confidence-0.1) // Decrease confidence on disagreement
			}

			// Update article with new confidence
			err = db.UpdateArticleScore(dbConn, req.ArticleID, compositeScore, confidence)
			if err != nil {
				// Log error but don't fail the request since feedback was saved
				LogError("feedbackHandler: update article confidence", err)
			}
		}

		RespondSuccess(c, map[string]string{"status": "feedback received"})
		LogPerformance("feedbackHandler", start)
	}
}

// --- Manual Score Handler ---
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
		if err := c.ShouldBindJSON(&raw); err != nil {
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
