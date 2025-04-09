package api

import (
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

func RegisterRoutes(router *gin.Engine, dbConn *sqlx.DB, rssCollector *rss.Collector, llmClient *llm.LLMClient) {
	router.GET("/api/articles", getArticlesHandler(dbConn))
	router.GET("/api/articles/:id", getArticleByIDHandler(dbConn))
	router.POST("/api/refresh", refreshHandler(rssCollector))
	router.POST("/api/llm/reanalyze/:id", reanalyzeHandler(llmClient))
	router.GET("/api/articles/:id/summary", summaryHandler(dbConn))
	router.GET("/api/articles/:id/bias", biasHandler(dbConn))
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})

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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})

			return
		}

		go func() {
			if err := llmClient.ReanalyzeArticle(int64(id)); err != nil {
				log.Printf("ReanalyzeArticle error: %v", err)
			}
		}()
		c.JSON(http.StatusOK, gin.H{"status": "reanalyze started"})
	}
}

func summaryHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")

		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})

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

func biasHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")

		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})

			return
		}

		scores, err := db.FetchLLMScores(dbConn, int64(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bias data"})

			return
		}

		for _, score := range scores {
			if score.Model == "bias-detector" {
				c.JSON(http.StatusOK, gin.H{
					"bias_score": score.Score,
					"metadata":   score.Metadata,
				})

				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Bias data not found"})
	}
}

func feedbackHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ArticleID    int64  `json:"article_id"`
			UserID       string `json:"user_id"`
			FeedbackText string `json:"feedback_text"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})

			return
		}

		if req.ArticleID == 0 || req.FeedbackText == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})

			return
		}

		feedback := &db.Feedback{
			ArticleID:    req.ArticleID,
			UserID:       req.UserID,
			FeedbackText: req.FeedbackText,
		}

		_, err := db.InsertFeedback(dbConn, feedback)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feedback"})

			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "feedback received"})
	}
}
