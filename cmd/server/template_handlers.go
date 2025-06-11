package main

import (
	"net/http"
	"strconv"
	"strings"

	dbpkg "github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// TemplateIndexHandler handles the articles listing page
func TemplateIndexHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) { // Get query parameters for filtering
		source := c.Query("source")
		bias := c.Query("bias")
		if bias == "" {
			bias = c.Query("leaning") // Support both parameter names for backward compatibility
		}
		query := c.Query("query")
		pageStr := c.DefaultQuery("page", "1")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limit := 20 // Articles per page
		offset := (page - 1) * limit

		// Build query conditions
		var whereConditions []string
		var args []interface{}
		argIndex := 1

		if source != "" {
			whereConditions = append(whereConditions, "source = $"+strconv.Itoa(argIndex))
			args = append(args, source)
			argIndex++
		}

		// Implement bias filtering using composite_score thresholds
		if bias != "" {
			switch strings.ToLower(bias) {
			case "left":
				whereConditions = append(whereConditions, "composite_score < $"+strconv.Itoa(argIndex))
				args = append(args, -0.1)
				argIndex++
			case "right":
				whereConditions = append(whereConditions, "composite_score > $"+strconv.Itoa(argIndex))
				args = append(args, 0.1)
				argIndex++
			case "center":
				whereConditions = append(whereConditions, "composite_score >= $"+strconv.Itoa(argIndex)+" AND composite_score <= $"+strconv.Itoa(argIndex+1))
				args = append(args, -0.1, 0.1)
				argIndex += 2
			}
		}

		if query != "" {
			whereConditions = append(whereConditions, "(title LIKE $"+strconv.Itoa(argIndex)+" OR content LIKE $"+strconv.Itoa(argIndex+1)+")")
			args = append(args, "%"+query+"%", "%"+query+"%")
			argIndex += 2
		}

		whereClause := ""
		if len(whereConditions) > 0 {
			whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
		}
		// Get articles
		sqlQuery := `
			SELECT id, title, source, url, pub_date, content, 
			       composite_score, confidence, score_source, created_at
			FROM articles ` + whereClause + `
			ORDER BY pub_date DESC
			LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)

		args = append(args, limit, offset)

		rows, err := db.Query(sqlQuery, args...)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "articles.html", gin.H{
				"Error": "Error fetching articles: " + err.Error(),
			})
			return
		}
		defer rows.Close()
		var articles []dbpkg.Article
		for rows.Next() {
			var article dbpkg.Article
			err := rows.Scan(
				&article.ID, &article.Title, &article.Source, &article.URL,
				&article.PubDate, &article.Content, &article.CompositeScore,
				&article.Confidence, &article.ScoreSource, &article.CreatedAt,
			)
			if err != nil {
				continue
			}
			articles = append(articles, article)
		}

		// Get total count for pagination
		countQuery := "SELECT COUNT(*) FROM articles " + whereClause
		countArgs := args[:len(args)-2] // Remove limit and offset

		var totalCount int
		err = db.Get(&totalCount, countQuery, countArgs...)
		if err != nil {
			totalCount = 0
		}

		totalPages := (totalCount + limit - 1) / limit

		// Generate page numbers for pagination
		var pages []int
		start := max(1, page-2)
		end := min(totalPages, page+2)
		for i := start; i <= end; i++ {
			pages = append(pages, i)
		}

		// Get available sources for filter
		var sources []string
		err = db.Select(&sources, "SELECT DISTINCT source FROM articles ORDER BY source")
		if err != nil {
			sources = []string{}
		}

		c.HTML(http.StatusOK, "articles.html", gin.H{
			"Articles":       articles,
			"Sources":        sources,
			"SearchQuery":    query,
			"SelectedSource": source,
			"SelectedBias":   bias,
			"CurrentPage":    page,
			"TotalPages":     totalPages,
			"Pages":          pages,
			"PrevPage":       page - 1,
			"NextPage":       page + 1,
		})
	}
}

// TemplateArticleHandler handles the individual article detail page
func TemplateArticleHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "article.html", gin.H{
				"Error": "Invalid article ID",
			})
			return
		}
		// Get the article
		var article dbpkg.Article
		err = db.Get(&article, `
			SELECT id, title, source, url, pub_date, content,
			       composite_score, confidence, score_source, created_at,
				   bias_label, analysis_notes
			FROM articles WHERE id = $1`, id)

		if err != nil {
			c.HTML(http.StatusNotFound, "article.html", gin.H{
				"Error": "Article not found",
			})
			return
		}

		// Get recent articles for sidebar
		var recentArticles []dbpkg.Article
		err = db.Select(&recentArticles, `
			SELECT id, title, source
			FROM articles 
			WHERE id != $1
			ORDER BY pub_date DESC 
			LIMIT 5`, id)
		if err != nil {
			recentArticles = []dbpkg.Article{}
		}

		// Get basic statistics
		stats := getBasicStats(db)

		c.HTML(http.StatusOK, "article.html", gin.H{
			"Article":        article,
			"RecentArticles": recentArticles,
			"Stats":          stats,
		})
	}
}

// TemplateAdminHandler handles the admin dashboard page
func TemplateAdminHandler(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get system statistics
		stats := getDetailedStats(db)

		// Get system status
		systemStatus := getSystemStatus(db)

		// Get recent activity (this could be expanded to include actual activity logs)
		recentActivity := getRecentActivity(db)

		c.HTML(http.StatusOK, "admin.html", gin.H{
			"Stats":          stats,
			"SystemStatus":   systemStatus,
			"RecentActivity": recentActivity,
		})
	}
}

// Helper functions

func formatBiasScore(score float64) string {
	return strconv.FormatFloat(score, 'f', 2, 64)
}

func getBasicStats(db *sqlx.DB) map[string]interface{} {
	stats := make(map[string]interface{})

	// Total articles
	var totalArticles int
	db.Get(&totalArticles, "SELECT COUNT(*) FROM articles")
	stats["TotalArticles"] = totalArticles

	// Count by bias
	var leftCount, centerCount, rightCount int
	db.Get(&leftCount, "SELECT COUNT(*) FROM articles WHERE bias_label = 'left'")
	db.Get(&centerCount, "SELECT COUNT(*) FROM articles WHERE bias_label = 'center'")
	db.Get(&rightCount, "SELECT COUNT(*) FROM articles WHERE bias_label = 'right'")

	stats["LeftCount"] = leftCount
	stats["CenterCount"] = centerCount
	stats["RightCount"] = rightCount

	// Calculate percentages
	if totalArticles > 0 {
		stats["LeftPercentage"] = (leftCount * 100) / totalArticles
		stats["CenterPercentage"] = (centerCount * 100) / totalArticles
		stats["RightPercentage"] = (rightCount * 100) / totalArticles
	} else {
		stats["LeftPercentage"] = 0
		stats["CenterPercentage"] = 0
		stats["RightPercentage"] = 0
	}

	return stats
}

func getDetailedStats(db *sqlx.DB) map[string]interface{} {
	stats := getBasicStats(db)

	// Articles today
	var articlesToday int
	db.Get(&articlesToday, "SELECT COUNT(*) FROM articles WHERE date(published_date) = date('now')")
	stats["ArticlesToday"] = articlesToday

	// Pending analysis (articles without scores)
	var pendingAnalysis int
	db.Get(&pendingAnalysis, "SELECT COUNT(*) FROM articles WHERE composite_score IS NULL")
	stats["PendingAnalysis"] = pendingAnalysis

	// Active sources
	var activeSources int
	db.Get(&activeSources, "SELECT COUNT(DISTINCT source) FROM articles")
	stats["ActiveSources"] = activeSources

	// Database size (approximation)
	stats["DatabaseSize"] = "~2.5MB" // This could be calculated more accurately

	return stats
}

func getSystemStatus(db *sqlx.DB) map[string]bool {
	status := make(map[string]bool)

	// Database connectivity
	err := db.Ping()
	status["DatabaseOK"] = err == nil

	// LLM service (simplified check)
	status["LLMServiceOK"] = true // This should check actual LLM service status

	// RSS service (simplified check)
	status["RSSServiceOK"] = true // This should check RSS feed status

	return status
}

func getRecentActivity(db *sqlx.DB) []map[string]interface{} {
	// This is a simplified implementation
	// In a real system, you'd have an activity log table
	var activities []map[string]interface{}

	// Get recent articles as activity
	rows, err := db.Query(`
		SELECT title, published_date 
		FROM articles 
		ORDER BY published_date DESC 
		LIMIT 5`)

	if err != nil {
		return activities
	}
	defer rows.Close()

	for rows.Next() {
		var title string
		var publishedDate string
		rows.Scan(&title, &publishedDate)

		activities = append(activities, map[string]interface{}{
			"Message":   "New article: " + title,
			"Timestamp": publishedDate,
		})
	}

	return activities
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
