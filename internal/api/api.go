package api

import (
	"encoding/json"
	"fmt"
	"log" // Added log package
	"net/http"
	"sort"
	"strconv"
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

const errInvalidArticleID = "Invalid article ID"

// --- Progress Tracking ---

type ProgressState struct {
	Step        string   `json:"step"`                  // Current detailed step (e.g., "Scoring with model X")
	Message     string   `json:"message"`               // User-friendly message for the step
	Percent     int      `json:"percent"`               // Progress percentage
	Status      string   `json:"status"`                // Overall status: "InProgress", "Success", "Error"
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

func RegisterRoutes(router *gin.Engine, dbConn *sqlx.DB, rssCollector *rss.Collector, llmClient *llm.LLMClient) {
	router.GET("/api/articles", getArticlesHandler(dbConn))
	router.GET("/api/articles/:id", getArticleByIDHandler(dbConn))
	router.POST("/api/refresh", refreshHandler(rssCollector))
	router.POST("/api/llm/reanalyze/:id", reanalyzeHandler(llmClient))
	router.GET("/api/articles/:id/summary", summaryHandler(dbConn))
	router.GET("/api/articles/:id/bias", biasHandler(dbConn))
	router.GET("/api/articles/:id/ensemble", ensembleDetailsHandler(dbConn))
	router.POST("/api/feedback", feedbackHandler(dbConn))
	router.GET("/api/feeds/healthz", feedHealthHandler(rssCollector))
	router.GET("/api/llm/score-progress/:id", scoreProgressSSEHandler())
}

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
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid 'limit' parameter")
			LogError("getArticlesHandler: invalid limit", err)
			return
		}
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid 'offset' parameter")
			LogError("getArticlesHandler: invalid offset", err)
			return
		}

		// Caching
		cacheKey := "articles:" + source + ":" + leaning + ":" + limitStr + ":" + offsetStr
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
			RespondError(c, http.StatusInternalServerError, ErrInternal, "Failed to fetch articles")
			LogError("getArticlesHandler: fetch articles", err)
			return
		}
		log.Printf("[getArticlesHandler] Fetched %d articles from DB for key: %s", len(articles), cacheKey) // DEBUG LOG ADDED
		// Optionally log specific article details if needed, e.g., the one being re-analyzed
		// for _, art := range articles { if art.ID == 1680 { log.Printf("[getArticlesHandler] DB Data for Article 1680: %+v", art) } }

		for i := range articles {
			// Fetch the latest ensemble score directly
			ensembleScore, scoreErr := db.FetchLatestEnsembleScore(dbConn, articles[i].ID)
			if scoreErr != nil {
				// Log error fetching the specific ensemble score, but don't block response
				log.Printf("[getArticlesHandler] Error fetching latest ensemble score for article %d: %v", articles[i].ID, scoreErr)
				articles[i].CompositeScore = nil // Default to nil if fetch fails
			} else {
				// Take the address of the float64 to assign to *float64
				scoreCopy := ensembleScore // Create a copy to ensure its address is stable
				articles[i].CompositeScore = &scoreCopy
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
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid article ID")
			LogError("getArticleByIDHandler: invalid id", err)
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
			RespondError(c, http.StatusNotFound, ErrNotFound, "Article not found")
			LogError("getArticleByIDHandler: fetch article", err)
			return
		}

		scores, _ := db.FetchLLMScores(dbConn, id)
		composite, confidence, _ := llm.ComputeCompositeScoreWithConfidence(scores)
		result := map[string]interface{}{
			"article":         article,
			"scores":          scores,
			"composite_score": composite,
			"confidence":      confidence,
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
func reanalyzeHandler(llmClient *llm.LLMClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid article ID")
			LogError("reanalyzeHandler: invalid id", err)
			return
		}
		articleID := int64(id)

		// Set initial progress: Queued
		setProgress(articleID, "Queued", "Scoring job queued", 0, "InProgress", "", nil)

		go func() {
			// Use recover for panics
			defer func() {
				if r := recover(); r != nil {
					errMsg := fmt.Sprintf("Internal panic: %v", r)
					log.Printf("[reanalyzeHandler %d] Recovered from panic: %s", articleID, errMsg)
					setProgress(articleID, "Error", "Internal error occurred", 0, "Error", errMsg, nil)
				}
			}()

			// Set progress: Starting
			setProgress(articleID, "Starting", "Starting scoring process", 0, "InProgress", "", nil)

			// Load configuration
			cfg, err := llm.LoadCompositeScoreConfig()
			if err != nil {
				errMsg := fmt.Sprintf("Failed to load scoring config: %v", err)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, 0, "Error", errMsg, nil)
				return
			}

			// Load article
			article, err := llmClient.GetArticle(articleID)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to load article: %v", err)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, 0, "Error", errMsg, nil)
				return
			}

			// Define total steps for progress calculation
			totalSteps := len(cfg.Models) + 3 // +1 delete, +N models, +1 calculate, +1 store
			stepNum := 1

			// Step 1: Delete old scores
			setProgress(articleID, "Preparing", "Deleting old scores", percent(stepNum, totalSteps), "InProgress", "", nil)
			err = llmClient.DeleteScores(articleID)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to delete old scores: %v", err)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}
			stepNum++

			// Step 2: Score with each model
			for _, m := range cfg.Models {
				label := fmt.Sprintf("Scoring with %s", m.ModelName)
				setProgress(articleID, label, label, percent(stepNum, totalSteps), "InProgress", "", nil)

				_, scoreErr := llmClient.ScoreWithModel(article, m.ModelName) // Renamed err to scoreErr for clarity
				log.Printf("[reanalyzeHandler %d] Model %s scoring result: err=%v", articleID, m.ModelName, scoreErr)

				if scoreErr != nil {
					log.Printf("[reanalyzeHandler %d] Actual error received from ScoreWithModel for model %s: (%T) %v", articleID, m.ModelName, scoreErr, scoreErr)
					log.Printf("[reanalyzeHandler %d] Error scoring with model %s, stopping analysis: %v", articleID, m.ModelName, scoreErr)

					errorMsg := scoreErr.Error()
					userMsg := fmt.Sprintf("Error scoring with %s", m.ModelName)
					if scoreErr == llm.ErrBothLLMKeysRateLimited {
						userMsg = llm.LLMRateLimitErrorMessage // Use specific user message for rate limit
						errorMsg = userMsg                     // Log the user message as the error too
					}
					setProgress(articleID, "Error", userMsg, percent(stepNum, totalSteps), "Error", errorMsg, nil)
					return // Exit the goroutine on first model error
				}
				stepNum++
			}
			log.Printf("[reanalyzeHandler %d] Scoring loop finished successfully.", articleID)

			// Step 3: Fetch scores and Calculate Final Composite Score
			setProgress(articleID, "Calculating", "Fetching scores for final calculation", percent(stepNum, totalSteps), "InProgress", "", nil)
			scores, fetchErr := llmClient.FetchScores(articleID) // Corrected: Use exported method from LLMClient
			if fetchErr != nil {
				errMsg := fmt.Sprintf("Failed to fetch scores for calculation: %v", fetchErr)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}
			finalScoreValue, _, calcErr := llm.ComputeCompositeScoreWithConfidence(scores)
			if calcErr != nil {
				errMsg := fmt.Sprintf("Failed to calculate final score: %v", calcErr)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, nil)
				return
			}
			log.Printf("[reanalyzeHandler %d] Calculated final score: %f", articleID, finalScoreValue)
			stepNum++

			// Step 4: Store ensemble score (Note: Ensure StoreEnsembleScore uses the calculated score if needed, or update article object)
			// Assuming StoreEnsembleScore implicitly uses the latest scores from DB or updates the article object passed to it.
			// If StoreEnsembleScore needs the calculated value explicitly, the call needs modification.
			log.Printf("[reanalyzeHandler %d] Attempting to store ensemble score.", articleID)
			actualFinalScore, storeErr := llmClient.StoreEnsembleScore(article) // Capture both return values
			if storeErr != nil {
				errMsg := fmt.Sprintf("Error storing ensemble score: %v", storeErr)
				log.Printf("[reanalyzeHandler %d] %s", articleID, errMsg)
				// Even if storing failed, report the score that was calculated before the failure
				setProgress(articleID, "Error", errMsg, percent(stepNum, totalSteps), "Error", errMsg, &actualFinalScore)
				return
			}
			// Send "Storing results" message AFTER successful storage
			setProgress(articleID, "Storing results", "Storing ensemble score", percent(stepNum, totalSteps), "InProgress", "", nil)
			// stepNum++ // No need to increment stepNum here, as the next step is the final one (100%)

			// Step 5: Final success step
			log.Printf("[reanalyzeHandler %d] Scoring complete. Final score reported: %f", articleID, actualFinalScore) // Log the score being reported
			setProgress(articleID, "Complete", "Scoring complete", 100, "Success", "", &actualFinalScore)               // Use actualFinalScore
		}()

		RespondSuccess(c, map[string]interface{}{
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

		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid article ID")
			LogError("summaryHandler: invalid id", err)
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

		scores, err := db.FetchLLMScores(dbConn, int64(id))
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrInternal, "Failed to fetch summary")
			LogError("summaryHandler: fetch scores", err)
			return
		}

		for _, score := range scores {
			if score.Model == "summarizer" {
				result := map[string]interface{}{"summary": score.Metadata}
				articlesCacheLock.Lock()
				articlesCache.Set(cacheKey, result, 30*time.Second)
				articlesCacheLock.Unlock()
				RespondSuccess(c, result)
				LogPerformance("summaryHandler", start)
				return
			}
		}

		RespondError(c, http.StatusNotFound, ErrNotFound, "Summary not found")
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
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid article ID")
			LogError("biasHandler: invalid id", err)
			return
		}

		minScore, err := strconv.ParseFloat(c.DefaultQuery("min_score", "-2"), 64)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid min_score")
			LogError("biasHandler: invalid min_score", err)
			return
		}
		maxScore, err := strconv.ParseFloat(c.DefaultQuery("max_score", "2"), 64)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid max_score")
			LogError("biasHandler: invalid max_score", err)
			return
		}
		sortOrder := c.DefaultQuery("sort", "desc")
		if sortOrder != "asc" && sortOrder != "desc" {
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid sort order")
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
			RespondError(c, http.StatusInternalServerError, ErrInternal, "Failed to fetch bias data")
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
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid article ID")
			LogError("ensembleDetailsHandler: invalid id", err)
			return
		}

		// Caching
		cacheKey := "ensemble:" + idStr
		articlesCacheLock.RLock()
		if cached, found := articlesCache.Get(cacheKey); found {
			articlesCacheLock.RUnlock()
			RespondSuccess(c, cached)
			LogPerformance("ensembleDetailsHandler (cache hit)", start)
			return
		}
		articlesCacheLock.RUnlock()

		scores, err := db.FetchLLMScores(dbConn, int64(id))
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrInternal, "Failed to fetch ensemble data")
			LogError("ensembleDetailsHandler: fetch scores", err)
			return
		}

		var details = make([]map[string]interface{}, 0, len(scores))
		for _, score := range scores {
			if score.Model != "ensemble" {
				continue
			}
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(score.Metadata), &meta); err != nil {
				continue
			}
			subResults, _ := meta["sub_results"].([]interface{})
			aggregation, _ := meta["aggregation"].(map[string]interface{})
			details = append(details, map[string]interface{}{
				"score":       score.Score,
				"sub_results": subResults,
				"aggregation": aggregation,
				"created_at":  score.CreatedAt,
			})
		}

		if len(details) == 0 {
			RespondError(c, http.StatusNotFound, ErrNotFound, "Ensemble data not found")
			LogPerformance("ensembleDetailsHandler", start)
			return
		}

		result := map[string]interface{}{"ensembles": details}
		articlesCacheLock.Lock()
		articlesCache.Set(cacheKey, result, 30*time.Second)
		articlesCacheLock.Unlock()

		RespondSuccess(c, result)
		LogPerformance("ensembleDetailsHandler", start)
	}
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
			RespondError(c, http.StatusBadRequest, ErrValidation, "Invalid request")
			LogError("feedbackHandler: bind", err)
			return
		}

		if req.ArticleID == 0 || req.FeedbackText == "" {
			RespondError(c, http.StatusBadRequest, ErrValidation, "Missing required fields")
			LogError("feedbackHandler: missing required fields", nil)
			return
		}

		feedback := &db.Feedback{
			ArticleID:        req.ArticleID,
			UserID:           req.UserID,
			FeedbackText:     req.FeedbackText,
			Category:         req.Category,
			EnsembleOutputID: req.EnsembleOutputID,
			Source:           req.Source,
		}

		_, err := db.InsertFeedback(dbConn, feedback)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrInternal, "Failed to save feedback")
			LogError("feedbackHandler: insert", err)
			return
		}

		RespondSuccess(c, map[string]string{"status": "feedback received"})
		LogPerformance("feedbackHandler", start)
	}
}
