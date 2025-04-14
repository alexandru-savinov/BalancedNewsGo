package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// This is a fixed version of the getArticleByIDHandler function
// that uses the article's stored composite score instead of recalculating it
func getArticleByIDHandlerFixed(dbConn *sqlx.DB) gin.HandlerFunc {
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
		// Use the article's stored composite score instead of recalculating it
		// This ensures consistency with manual rescoring
		result := map[string]interface{}{
			"article":         article,
			"scores":          scores,
			"composite_score": article.CompositeScore,
			"confidence":      article.Confidence,
		}

		// Cache the result for 30 seconds
		articlesCacheLock.Lock()
		articlesCache.Set(cacheKey, result, 30*time.Second)
		articlesCacheLock.Unlock()

		RespondSuccess(c, result)
		LogPerformance("getArticleByIDHandler", start)
	}
}

// This function should be called in RegisterRoutes instead of the original getArticleByIDHandler
func UpdateRoutes(router *gin.Engine, dbConn *sqlx.DB) {
	// Replace the original handler with the fixed one
	router.GET("/api/articles/:id", getArticleByIDHandlerFixed(dbConn))
}
