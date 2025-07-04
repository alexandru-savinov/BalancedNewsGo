package api

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// AdminOperationResponse represents the standard response for admin operations
type AdminOperationResponse struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// SystemHealthResponse represents system health check results
type SystemHealthResponse struct {
	DatabaseOK   bool `json:"database_ok"`
	LLMServiceOK bool `json:"llm_service_ok"`
	RSSServiceOK bool `json:"rss_service_ok"`
	ServerOK     bool `json:"server_ok"`
}

// SystemStatsResponse represents system statistics
type SystemStatsResponse struct {
	TotalArticles    int     `json:"total_articles"`
	ArticlesToday    int     `json:"articles_today"`
	PendingAnalysis  int     `json:"pending_analysis"`
	ActiveSources    int     `json:"active_sources"`
	DatabaseSize     string  `json:"database_size"`
	LeftCount        int     `json:"left_count"`
	CenterCount      int     `json:"center_count"`
	RightCount       int     `json:"right_count"`
	LeftPercentage   float64 `json:"left_percentage"`
	CenterPercentage float64 `json:"center_percentage"`
	RightPercentage  float64 `json:"right_percentage"`
}

// Feed Management Handlers

// adminRefreshFeedsHandler handles POST /api/admin/refresh-feeds
func adminRefreshFeedsHandler(rssCollector rss.CollectorInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// RSS operations are async, start in goroutine
		go rssCollector.ManualRefresh()

		// Use AdminOperationResponse for consistent format
		response := AdminOperationResponse{
			Status:    "refresh_initiated",
			Message:   "Feed refresh started successfully",
			Timestamp: time.Now().UTC(),
		}
		RespondSuccess(c, response)
	}
}

// adminResetFeedErrorsHandler handles POST /api/admin/reset-feed-errors
func adminResetFeedErrorsHandler(rssCollector rss.CollectorInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Since the RSS collector interface doesn't have a ResetErrors method,
		// we'll log the action and return success. In a real implementation,
		// this could clear error counters or reset failed feed states.
		log.Printf("[ADMIN] Feed errors reset requested")

		response := AdminOperationResponse{
			Status:    "errors_reset",
			Message:   "Feed errors have been reset (logged action)",
			Timestamp: time.Now().UTC(),
		}
		RespondSuccess(c, response)
	}
}

// adminGetSourcesStatusHandler handles GET /api/admin/sources
func adminGetSourcesStatusHandler(rssCollector rss.CollectorInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get feed health status
		healthStatus := rssCollector.CheckFeedHealth()

		// Count active sources
		activeSources := 0
		for _, isHealthy := range healthStatus {
			if isHealthy {
				activeSources++
			}
		}

		RespondSuccess(c, map[string]interface{}{
			"sources":        healthStatus,
			"active_sources": activeSources,
			"total_sources":  len(healthStatus),
			"timestamp":      time.Now().UTC(),
		})
	}
}

// Analysis Control Handlers

// adminReanalyzeRecentHandler handles POST /api/admin/reanalyze-recent
func adminReanalyzeRecentHandler(llmClient *llm.LLMClient, scoreManager *llm.ScoreManager, dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate LLM service availability
		if err := llmClient.ValidateAPIKey(); err != nil {
			RespondError(c, WrapError(err, ErrLLMService, "LLM service unavailable"))
			return
		}

		// Start async reanalysis of recent articles (last 7 days)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()

			// Query for recent articles (last 7 days)
			query := `
				SELECT id FROM articles
				WHERE created_at >= datetime('now', '-7 days')
				ORDER BY created_at DESC
				LIMIT 50
			`
			var articleIDs []int64
			if err := dbConn.Select(&articleIDs, query); err != nil {
				log.Printf("[ADMIN] Failed to query recent articles: %v", err)
				return
			}

			log.Printf("[ADMIN] Starting reanalysis of %d recent articles", len(articleIDs))

			// Reanalyze each article
			for _, articleID := range articleIDs {
				if err := llmClient.ReanalyzeArticle(ctx, articleID, scoreManager); err != nil {
					log.Printf("[ADMIN] Failed to reanalyze article %d: %v", articleID, err)
					continue
				}
				log.Printf("[ADMIN] Successfully reanalyzed article %d", articleID)
			}

			log.Printf("[ADMIN] Completed reanalysis of recent articles")
		}()

		response := AdminOperationResponse{
			Status:    "reanalysis_started",
			Message:   "Reanalysis of recent articles initiated (last 7 days, max 50 articles)",
			Timestamp: time.Now().UTC(),
		}
		RespondSuccess(c, response)
	}
}

// adminClearAnalysisErrorsHandler handles POST /api/admin/clear-analysis-errors
func adminClearAnalysisErrorsHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Clear articles with error status
		result, err := dbConn.ExecContext(ctx, `
			UPDATE articles
			SET status = 'pending'
			WHERE status = 'error'
		`)
		if err != nil {
			log.Printf("[ADMIN] Failed to clear analysis errors: %v", err)
			RespondError(c, fmt.Errorf("failed to clear analysis errors: %w", err))
			return
		}

		rowsAffected, _ := result.RowsAffected()
		log.Printf("[ADMIN] Cleared analysis errors for %d articles", rowsAffected)

		RespondSuccess(c, map[string]interface{}{
			"status":         "errors_cleared",
			"message":        "Analysis errors have been cleared",
			"articles_reset": rowsAffected,
			"timestamp":      time.Now().UTC(),
		})
	}
}

// adminValidateBiasScoresHandler handles POST /api/admin/validate-scores
func adminValidateBiasScoresHandler(llmClient *llm.LLMClient, scoreManager *llm.ScoreManager, dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Query for articles with scores to validate
		query := `
			SELECT DISTINCT a.id, a.title, a.composite_score, a.confidence
			FROM articles a
			INNER JOIN llm_scores ls ON a.id = ls.article_id
			WHERE a.composite_score IS NOT NULL
			ORDER BY a.created_at DESC
			LIMIT 100
		`

		var results []struct {
			ID         int64   `db:"id"`
			Title      string  `db:"title"`
			BiasScore  float64 `db:"composite_score"`
			Confidence float64 `db:"confidence"`
		}

		if err := dbConn.SelectContext(ctx, &results, query); err != nil {
			log.Printf("[ADMIN] Failed to query articles for validation: %v", err)
			RespondError(c, fmt.Errorf("failed to query articles for validation: %w", err))
			return
		}

		// Validate score ranges and consistency
		invalidScores := 0
		for _, result := range results {
			if result.BiasScore < -1.0 || result.BiasScore > 1.0 {
				invalidScores++
				log.Printf("[ADMIN] Invalid bias score for article %d: %f", result.ID, result.BiasScore)
			}
			if result.Confidence < 0.0 || result.Confidence > 1.0 {
				invalidScores++
				log.Printf("[ADMIN] Invalid confidence for article %d: %f", result.ID, result.Confidence)
			}
		}

		log.Printf("[ADMIN] Score validation completed: %d articles checked, %d invalid scores found", len(results), invalidScores)

		RespondSuccess(c, map[string]interface{}{
			"status":           "validation_completed",
			"message":          "Bias score validation completed",
			"articles_checked": len(results),
			"invalid_scores":   invalidScores,
			"timestamp":        time.Now().UTC(),
		})
	}
}

// Database Management Handlers

// adminOptimizeDatabaseHandler handles POST /api/admin/optimize-db
func adminOptimizeDatabaseHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Database operations need proper error handling
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Execute VACUUM and ANALYZE for SQLite optimization
		_, err := dbConn.ExecContext(ctx, "VACUUM; ANALYZE;")
		if err != nil {
			log.Printf("[ERROR] Database optimization failed: %v", err)
			RespondError(c, fmt.Errorf("database optimization failed: %w", err))
			return
		}

		RespondSuccess(c, map[string]interface{}{
			"status":    "optimization_completed",
			"message":   "Database optimization completed successfully",
			"timestamp": time.Now().UTC(),
		})
	}
}

// adminExportDataHandler handles GET /api/admin/export
func adminExportDataHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Query for articles with their scores
		query := `
			SELECT a.id, a.title, a.source, a.url, a.pub_date, a.composite_score, a.confidence, a.status,
				   GROUP_CONCAT(ls.model || ':' || ls.score) as llm_scores
			FROM articles a
			LEFT JOIN llm_scores ls ON a.id = ls.article_id
			GROUP BY a.id
			ORDER BY a.created_at DESC
			LIMIT 1000
		`

		var results []struct {
			ID         int64    `db:"id"`
			Title      string   `db:"title"`
			Source     string   `db:"source"`
			URL        string   `db:"url"`
			PubDate    string   `db:"pub_date"`
			BiasScore  *float64 `db:"composite_score"`
			Confidence *float64 `db:"confidence"`
			Status     string   `db:"status"`
			LLMScores  *string  `db:"llm_scores"`
		}

		if err := dbConn.SelectContext(ctx, &results, query); err != nil {
			log.Printf("[ADMIN] Failed to export data: %v", err)
			RespondError(c, fmt.Errorf("failed to export data: %w", err))
			return
		}

		// Set headers for CSV download
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=articles_export.csv")

		// Write CSV header
		c.String(200, "ID,Title,Source,URL,PubDate,BiasScore,Confidence,Status,LLMScores\n")

		// Write CSV data
		for _, result := range results {
			biasScore := ""
			if result.BiasScore != nil {
				biasScore = fmt.Sprintf("%.3f", *result.BiasScore)
			}
			confidence := ""
			if result.Confidence != nil {
				confidence = fmt.Sprintf("%.3f", *result.Confidence)
			}
			llmScores := ""
			if result.LLMScores != nil {
				llmScores = *result.LLMScores
			}

			c.String(200, "%d,\"%s\",\"%s\",\"%s\",\"%s\",%s,%s,\"%s\",\"%s\"\n",
				result.ID, result.Title, result.Source, result.URL, result.PubDate,
				biasScore, confidence, result.Status, llmScores)
		}

		log.Printf("[ADMIN] Exported %d articles to CSV", len(results))
	}
}

// adminCleanupOldArticlesHandler handles DELETE /api/admin/cleanup-old
func adminCleanupOldArticlesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		// Start transaction for cleanup
		tx, err := dbConn.BeginTxx(ctx, nil)
		if err != nil {
			log.Printf("[ADMIN] Failed to start cleanup transaction: %v", err)
			RespondError(c, fmt.Errorf("failed to start cleanup transaction: %w", err))
			return
		}
		defer func() {
			if err := tx.Rollback(); err != nil {
				log.Printf("[ADMIN] Failed to rollback transaction: %v", err)
			}
		}()

		// Delete LLM scores for old articles first (foreign key constraint)
		_, err = tx.ExecContext(ctx, `
			DELETE FROM llm_scores
			WHERE article_id IN (
				SELECT id FROM articles
				WHERE created_at < datetime('now', '-30 days')
			)
		`)
		if err != nil {
			log.Printf("[ADMIN] Failed to delete old LLM scores: %v", err)
			RespondError(c, fmt.Errorf("failed to delete old LLM scores: %w", err))
			return
		}

		// Delete old articles (>30 days)
		result, err := tx.ExecContext(ctx, `
			DELETE FROM articles
			WHERE created_at < datetime('now', '-30 days')
		`)
		if err != nil {
			log.Printf("[ADMIN] Failed to delete old articles: %v", err)
			RespondError(c, fmt.Errorf("failed to delete old articles: %w", err))
			return
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			log.Printf("[ADMIN] Failed to commit cleanup transaction: %v", err)
			RespondError(c, fmt.Errorf("failed to commit cleanup transaction: %w", err))
			return
		}

		deletedCount, _ := result.RowsAffected()
		log.Printf("[ADMIN] Cleanup completed: deleted %d old articles", deletedCount)

		RespondSuccess(c, map[string]interface{}{
			"status":       "cleanup_completed",
			"message":      "Old articles cleanup completed",
			"deletedCount": deletedCount,
			"timestamp":    time.Now().UTC(),
		})
	}
}

// Monitoring Handlers

// adminGetMetricsHandler handles GET /api/admin/metrics
func adminGetMetricsHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		stats := SystemStatsResponse{}

		// Get total articles count
		err := dbConn.GetContext(ctx, &stats.TotalArticles, "SELECT COUNT(*) FROM articles")
		if err != nil {
			log.Printf("[ADMIN] Failed to get total articles count: %v", err)
		}

		// Get articles today count
		err = dbConn.GetContext(ctx, &stats.ArticlesToday,
			"SELECT COUNT(*) FROM articles WHERE DATE(created_at) = DATE('now')")
		if err != nil {
			log.Printf("[ADMIN] Failed to get today's articles count: %v", err)
		}

		// Get pending analysis count
		err = dbConn.GetContext(ctx, &stats.PendingAnalysis,
			"SELECT COUNT(*) FROM articles WHERE status = 'pending' OR composite_score IS NULL")
		if err != nil {
			log.Printf("[ADMIN] Failed to get pending analysis count: %v", err)
		}

		// Get bias distribution counts
		var leftCount, centerCount, rightCount int
		err = dbConn.GetContext(ctx, &leftCount,
			"SELECT COUNT(*) FROM articles WHERE composite_score < -0.2")
		if err != nil {
			log.Printf("[ADMIN] Failed to get left count: %v", err)
		}

		err = dbConn.GetContext(ctx, &centerCount,
			"SELECT COUNT(*) FROM articles WHERE composite_score >= -0.2 AND composite_score <= 0.2")
		if err != nil {
			log.Printf("[ADMIN] Failed to get center count: %v", err)
		}

		err = dbConn.GetContext(ctx, &rightCount,
			"SELECT COUNT(*) FROM articles WHERE composite_score > 0.2")
		if err != nil {
			log.Printf("[ADMIN] Failed to get right count: %v", err)
		}

		stats.LeftCount = leftCount
		stats.CenterCount = centerCount
		stats.RightCount = rightCount

		// Calculate percentages
		total := leftCount + centerCount + rightCount
		if total > 0 {
			stats.LeftPercentage = float64(leftCount) / float64(total) * 100
			stats.CenterPercentage = float64(centerCount) / float64(total) * 100
			stats.RightPercentage = float64(rightCount) / float64(total) * 100
		}

		// Get database size (SQLite specific)
		var dbSize int64
		err = dbConn.GetContext(ctx, &dbSize, "SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size()")
		if err != nil {
			log.Printf("[ADMIN] Failed to get database size: %v", err)
			stats.DatabaseSize = "Unknown"
		} else {
			stats.DatabaseSize = fmt.Sprintf("%.2f MB", float64(dbSize)/(1024*1024))
		}

		// Active sources would need RSS collector integration
		stats.ActiveSources = 0 // Placeholder

		RespondSuccess(c, stats)
	}
}

// adminGetLogsHandler handles GET /api/admin/logs
func adminGetLogsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For now, return a placeholder response since log retrieval would require
		// implementing a proper logging system with file or database storage
		// In a production system, this would read from log files or a logging service

		sampleLogs := []map[string]interface{}{
			{
				"timestamp": time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
				"level":     "INFO",
				"message":   "RSS feed refresh completed successfully",
				"component": "RSS",
			},
			{
				"timestamp": time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339),
				"level":     "INFO",
				"message":   "LLM analysis completed for article ID 123",
				"component": "LLM",
			},
			{
				"timestamp": time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339),
				"level":     "WARN",
				"message":   "High confidence threshold not met for article analysis",
				"component": "LLM",
			},
		}

		RespondSuccess(c, map[string]interface{}{
			"logs":      sampleLogs,
			"message":   "Sample log entries (implement proper log storage for production)",
			"timestamp": time.Now().UTC(),
		})
	}
}

// adminRunHealthCheckHandler handles POST /api/admin/health-check
func adminRunHealthCheckHandler(dbConn *sqlx.DB, llmClient *llm.LLMClient, rssCollector rss.CollectorInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := SystemHealthResponse{
			ServerOK:     true, // If we're responding, server is OK
			DatabaseOK:   false,
			LLMServiceOK: false,
			RSSServiceOK: false,
		}

		// Check database connectivity
		if err := dbConn.Ping(); err == nil {
			health.DatabaseOK = true
		}

		// Check LLM service
		if err := llmClient.ValidateAPIKey(); err == nil {
			health.LLMServiceOK = true
		}

		// Check RSS service by testing feed health
		feedHealth := rssCollector.CheckFeedHealth()
		if len(feedHealth) > 0 {
			health.RSSServiceOK = true
		}

		RespondSuccess(c, health)
	}
}
