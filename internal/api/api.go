package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

const errInvalidArticleID = "Invalid article ID"

func RegisterRoutes(router *gin.Engine, dbConn *sqlx.DB, rssCollector *rss.Collector, llmClient *llm.LLMClient) {
	router.GET("/api/articles", getArticlesHandler(dbConn))
	router.GET("/api/articles/:id", getArticleByIDHandler(dbConn))
	router.POST("/api/refresh", refreshHandler(rssCollector))
	router.POST("/api/llm/reanalyze/:id", reanalyzeHandler(llmClient))
	router.GET("/api/articles/:id/summary", summaryHandler(dbConn))
	router.GET("/api/articles/:id/bias", biasHandler(dbConn))
	router.GET("/api/articles/:id/ensemble", ensembleDetailsHandler(dbConn))
	router.POST("/api/feedback", feedbackHandler(dbConn))
}

func getArticlesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		source := c.Query("source")
		leaning := c.Query("leaning")
		limitStr := c.DefaultQuery("limit", "20")
		offsetStr := c.DefaultQuery("offset", "0")
		limit, _ := strconv.Atoi(limitStr)
		offset, _ := strconv.Atoi(offsetStr)

		articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles"})
			return
		}

		for i := range articles {
			scores, _ := db.FetchLLMScores(dbConn, articles[i].ID)
			articles[i].CompositeScore = llm.ComputeCompositeScore(scores)
		}

		sort.Slice(articles, func(i, j int) bool {
			return articles[i].CompositeScore > articles[j].CompositeScore
		})

		c.JSON(http.StatusOK, articles)
	}
}

func getArticleByIDHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidArticleID})

			return
		}

		article, err := db.FetchArticleByID(dbConn, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})

			return
		}

		scores, _ := db.FetchLLMScores(dbConn, id)
		composite := llm.ComputeCompositeScore(scores)
		c.JSON(http.StatusOK, gin.H{
			"article":         article,
			"scores":          scores,
			"composite_score": composite,
		})
	}
}

func refreshHandler(rssCollector *rss.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		go rssCollector.ManualRefresh()
		c.JSON(http.StatusOK, gin.H{"status": "refresh started"})
	}
}

func reanalyzeHandler(llmClient *llm.LLMClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")

		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidArticleID})
			return
		}

		err = llmClient.ReanalyzeArticle(int64(id))
		if err != nil {
			log.Printf("ReanalyzeArticle error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "reanalyze failed",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     "reanalyze complete",
			"article_id": id,
		})
	}
}

func summaryHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")

		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidArticleID})

			return
		}

		scores, err := db.FetchLLMScores(dbConn, int64(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch summary"})

			return
		}

		for _, score := range scores {
			if score.Model == "summarizer" {
				c.JSON(http.StatusOK, gin.H{"summary": score.Metadata})

				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Summary not found"})
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
		articleID, ok := parseArticleID(c)
		if !ok {
			return
		}

		minScore, _ := strconv.ParseFloat(c.DefaultQuery("min_score", "-2"), 64)
		maxScore, _ := strconv.ParseFloat(c.DefaultQuery("max_score", "2"), 64)
		sortOrder := c.DefaultQuery("sort", "desc")

		scores, err := db.FetchLLMScores(dbConn, articleID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bias data"})
			return
		}

		ensembleResults := filterAndTransformScores(scores, minScore, maxScore)
		sortResults(ensembleResults, sortOrder)

		var detailedResults []map[string]interface{}
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

			detailedResults = append(detailedResults, map[string]interface{}{
				"model":       s.Model,
				"score":       s.Score,
				"confidence":  parsed.Confidence,
				"explanation": parsed.Explanation,
			})
		}

		var composite interface{}
		status := ""
		if totalWeight > 0 {
			composite = weightedSum / totalWeight
		} else {
			composite = nil
			status = "scoring_unavailable"
		}
		resp := gin.H{
			"composite_score": composite,
			"results":         ensembleResults,
		}
		if status != "" {
			resp["status"] = status
		}
		c.JSON(http.StatusOK, resp)
	}
}

func ensembleDetailsHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidArticleID})
			return
		}

		scores, err := db.FetchLLMScores(dbConn, int64(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ensemble data"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Ensemble data not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ensembles": details,
		})
	}
}

func feedbackHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ArticleID        int64  `json:"article_id" form:"article_id"`
			UserID           string `json:"user_id" form:"user_id"`
			FeedbackText     string `json:"feedback_text" form:"feedback_text"`
			Category         string `json:"category" form:"category"`
			EnsembleOutputID *int64 `json:"ensemble_output_id" form:"ensemble_output_id"`
			Source           string `json:"source" form:"source"`
		}

		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})

			return
		}

		if req.ArticleID == 0 || req.FeedbackText == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})

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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feedback"})

			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "feedback received"})
	}
}
