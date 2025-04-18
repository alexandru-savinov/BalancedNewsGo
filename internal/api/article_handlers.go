package api

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Handler for POST /api/articles
func createArticleHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Source  string `json:"source"`
			PubDate string `json:"pub_date"` // ISO8601 or RFC3339 string
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			appErr := handleError(err, "invalid request body")
			appErr.Code = ErrInvalidInput
			RespondError(c, appErr)
			LogError("createArticleHandler", appErr)
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
			appErr := &AppError{
				Message: fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", ")),
				Code:    ErrValidation,
			}
			RespondError(c, appErr)
			LogError("createArticleHandler", appErr)
			return
		}

		// Validate URL format
		if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
			appErr := &AppError{
				Message: "Invalid URL format (must start with http:// or https://)",
				Code:    ErrValidation,
			}
			RespondError(c, appErr)
			LogError("createArticleHandler", appErr)
			return
		}

		// Check if article with this URL already exists
		exists, err := db.ArticleExistsByURL(dbConn, req.URL)
		if err != nil {
			appErr := handleError(err, "database check failed")
			appErr.Code = ErrDatabaseOperation
			RespondError(c, appErr)
			LogError("createArticleHandler", appErr)
			return
		}
		if exists {
			appErr := &AppError{
				Message: "Article with this URL already exists",
				Code:    ErrValidation,
			}
			RespondError(c, appErr)
			LogError("createArticleHandler", appErr)
			return
		}

		// Parse pub_date
		pubDate, err := time.Parse(time.RFC3339, req.PubDate)
		if err != nil {
			appErr := handleError(err, "invalid date format")
			appErr.Code = ErrValidation
			RespondError(c, appErr)
			LogError("createArticleHandler", appErr)
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
			ScoreSource:    sql.NullString{String: "llm", Valid: true},
		}

		id, err := db.InsertArticle(dbConn, article)
		if err != nil {
			appErr := handleError(err, "failed to create article")
			appErr.Code = ErrDatabaseOperation
			RespondError(c, appErr)
			LogError("createArticleHandler", appErr)
			return
		}

		RespondSuccess(c, map[string]interface{}{
			"status":     "created",
			"article_id": id,
		})
	}
}
