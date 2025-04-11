package api

import (
	"encoding/json"
	"fmt"
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
	Step        string `json:"step"`
	Message     string `json:"message"`
	Percent     int    `json:"percent"`
	Error       string `json:"error,omitempty"`
	LastUpdated int64  `json:"last_updated"`
}

var (
	progressMap     = make(map[int64]*ProgressState)
	progressMapLock sync.RWMutex
)

func setProgress(articleID int64, step, message string, percent int, errMsg string) {
	progressMapLock.Lock()
	defer progressMapLock.Unlock()
	progressMap[articleID] = &ProgressState{
		Step:        step,
		Message:     message,
		Percent:     percent,
		Error:       errMsg,
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
		if cached, found := articlesCache.Get(cacheKey); found {
			articlesCacheLock.RUnlock()
			RespondSuccess(c, cached)
			LogPerformance("getArticlesHandler (cache hit)", start)
			return
		}
		articlesCacheLock.RUnlock()

		articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrInternal, "Failed to fetch articles")
			LogError("getArticlesHandler: fetch articles", err)
			return
		}

		for i := range articles {
			scores, _ := db.FetchLLMScores(dbConn, articles[i].ID)
			articles[i].CompositeScore = llm.ComputeCompositeScore(scores)
		}

		sort.Slice(articles, func(i, j int) bool {
			return articles[i].CompositeScore > articles[j].CompositeScore
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

		setProgress(articleID, "Queued", "Scoring job queued", 0, "")

		go func() {
			defer func() {
				if r := recover(); r != nil {
					setProgress(articleID, "Error", "Internal error occurred", 0, fmt.Sprintf("%v", r))
				}
			}()

			setProgress(articleID, "Starting", "Starting scoring process", 0, "")

			cfg, err := llm.LoadCompositeScoreConfig()
			if err != nil {
				setProgress(articleID, "Error", "Failed to load scoring config", 0, err.Error())
				return
			}

			article, err := llmClient.GetArticle(articleID)
			if err != nil {
				setProgress(articleID, "Error", "Failed to load article", 0, err.Error())
				return
			}

			totalSteps := len(cfg.Models) + 2 // +1 for storing, +1 for complete
			stepNum := 1

			// Delete old scores
			err = llmClient.DeleteScores(articleID)
			if err != nil {
				setProgress(articleID, "Error", "Failed to delete old scores", percent(stepNum, totalSteps), err.Error())
				return
			}

			// Score with each model
			for _, m := range cfg.Models {
				label := fmt.Sprintf("Scoring with %s", m.ModelName)
				setProgress(articleID, label, label, percent(stepNum, totalSteps), "")
				_, err := llmClient.ScoreWithModel(article, m.ModelName, m.URL)
				if err != nil {
					setProgress(articleID, "Error", fmt.Sprintf("Error scoring with %s", m.ModelName), percent(stepNum, totalSteps), err.Error())
					return
				}
				stepNum++
			}

			// Store ensemble score
			setProgress(articleID, "Storing results", "Storing ensemble score", percent(stepNum, totalSteps), "")
			err = llmClient.StoreEnsembleScore(article)
			if err != nil {
				setProgress(articleID, "Error", "Error storing ensemble score", percent(stepNum, totalSteps), err.Error())
				return
			}
			stepNum++

			setProgress(articleID, "Complete", "Scoring complete", 100, "")
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

		ensembleResults := filterAndTransformScores(scores, minScore, maxScore)
		sortResults(ensembleResults, sortOrder)

		var weightedSum float64
		var totalWeight float64

		for _, s := range scores {
			var parsed struct {
				Confidence  float64 `json:"Confidence"`
				Explanation string  `json:"Explanation"`
			}
			_ = json.Unmarshal([]byte(s.Metadata), &parsed)

			weightedSum += s.Score * parsed.Confidence
			totalWeight += parsed.Confidence
		}

		var composite interface{}
		status := ""
		if totalWeight > 0 {
			composite = weightedSum / totalWeight
		} else {
			composite = nil
			status = "scoring_unavailable"
		}
		resp := map[string]interface{}{
			"composite_score": composite,
			"results":         ensembleResults,
		}
		if status != "" {
			resp["status"] = status
		}

		// Cache the result for 30 seconds
		articlesCacheLock.Lock()
		articlesCache.Set(cacheKey, resp, 30*time.Second)
		articlesCacheLock.Unlock()

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
