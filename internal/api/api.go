package api

import (
	"net/http"
	"strconv"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
	"github.com/jmoiron/sqlx"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all API endpoints to the Gin router
func RegisterRoutes(router *gin.Engine, dbConn *sqlx.DB, rssCollector *rss.Collector, llmClient *llm.LLMClient) {
	router.GET("/api/articles", func(c *gin.Context) {
		// Optional filters
		source := c.Query("source")
		leaning := c.Query("leaning")
		// Pagination params
		limitStr := c.DefaultQuery("limit", "20")
		offsetStr := c.DefaultQuery("offset", "0")
		limit, _ := strconv.Atoi(limitStr)
		offset, _ := strconv.Atoi(offsetStr)

		articles, err := db.FetchArticles(dbConn, source, leaning, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles"})
			return
		}
		c.JSON(http.StatusOK, articles)
	})

	router.GET("/api/articles/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
			return
		}
		article, err := db.FetchArticleByID(dbConn, int64(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
			return
		}
		scores, _ := db.FetchLLMScores(dbConn, int64(id))
		c.JSON(http.StatusOK, gin.H{
			"article": article,
			"scores":  scores,
		})
	})

	router.POST("/api/refresh", func(c *gin.Context) {
		go rssCollector.ManualRefresh()
		c.JSON(http.StatusOK, gin.H{"status": "refresh started"})
	})

	router.POST("/api/llm/reanalyze/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
			return
		}
		go llmClient.ReanalyzeArticle(int64(id))
		c.JSON(http.StatusOK, gin.H{"status": "reanalyze started"})
	})
}
