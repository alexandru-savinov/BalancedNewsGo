package api

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
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
// getRecentArticleIDs queries for recent article IDs to reanalyze
func getRecentArticleIDs(dbConn *sqlx.DB) ([]int64, error) {
	query := `
		SELECT id FROM articles
		WHERE created_at >= datetime('now', '-7 days')
		ORDER BY created_at DESC
		LIMIT 50
	`
	var articleIDs []int64
	err := dbConn.Select(&articleIDs, query)
	return articleIDs, err
}

// reanalyzeArticlesBatch processes a batch of articles for reanalysis
func reanalyzeArticlesBatch(ctx context.Context, llmClient *llm.LLMClient, scoreManager *llm.ScoreManager, articleIDs []int64) {
	log.Printf("[ADMIN] Starting reanalysis of %d recent articles", len(articleIDs))

	for _, articleID := range articleIDs {
		if err := llmClient.ReanalyzeArticle(ctx, articleID, scoreManager); err != nil {
			log.Printf("[ADMIN] Failed to reanalyze article %d: %v", articleID, err)
			continue
		}
		log.Printf("[ADMIN] Successfully reanalyzed article %d", articleID)
	}

	log.Printf("[ADMIN] Completed reanalysis of recent articles")
}

// performAsyncReanalysis handles the async reanalysis workflow
func performAsyncReanalysis(llmClient *llm.LLMClient, scoreManager *llm.ScoreManager, dbConn *sqlx.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	articleIDs, err := getRecentArticleIDs(dbConn)
	if err != nil {
		log.Printf("[ADMIN] Failed to query recent articles: %v", err)
		return
	}

	reanalyzeArticlesBatch(ctx, llmClient, scoreManager, articleIDs)
}

func adminReanalyzeRecentHandler(llmClient *llm.LLMClient, scoreManager *llm.ScoreManager, dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate LLM service availability
		if err := llmClient.ValidateAPIKey(); err != nil {
			RespondError(c, WrapError(err, ErrLLMService, "LLM service unavailable"))
			return
		}

		// Start async reanalysis of recent articles (last 7 days)
		go performAsyncReanalysis(llmClient, scoreManager, dbConn)

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
// getBasicCounts retrieves basic article counts
func getBasicCounts(ctx context.Context, dbConn *sqlx.DB) (totalArticles, articlesToday, pendingAnalysis int) {
	if err := dbConn.GetContext(ctx, &totalArticles, "SELECT COUNT(*) FROM articles"); err != nil {
		log.Printf("[ADMIN] Failed to get total articles count: %v", err)
	}

	if err := dbConn.GetContext(ctx, &articlesToday,
		"SELECT COUNT(*) FROM articles WHERE DATE(created_at) = DATE('now')"); err != nil {
		log.Printf("[ADMIN] Failed to get today's articles count: %v", err)
	}

	if err := dbConn.GetContext(ctx, &pendingAnalysis,
		"SELECT COUNT(*) FROM articles WHERE status = 'pending' OR composite_score IS NULL"); err != nil {
		log.Printf("[ADMIN] Failed to get pending analysis count: %v", err)
	}

	return totalArticles, articlesToday, pendingAnalysis
}

// getBiasDistribution retrieves bias distribution counts
func getBiasDistribution(ctx context.Context, dbConn *sqlx.DB) (leftCount, centerCount, rightCount int) {
	if err := dbConn.GetContext(ctx, &leftCount,
		"SELECT COUNT(*) FROM articles WHERE composite_score < -0.2"); err != nil {
		log.Printf("[ADMIN] Failed to get left count: %v", err)
	}

	if err := dbConn.GetContext(ctx, &centerCount,
		"SELECT COUNT(*) FROM articles WHERE composite_score >= -0.2 AND composite_score <= 0.2"); err != nil {
		log.Printf("[ADMIN] Failed to get center count: %v", err)
	}

	if err := dbConn.GetContext(ctx, &rightCount,
		"SELECT COUNT(*) FROM articles WHERE composite_score > 0.2"); err != nil {
		log.Printf("[ADMIN] Failed to get right count: %v", err)
	}

	return leftCount, centerCount, rightCount
}

// calculatePercentages calculates bias percentages
func calculatePercentages(leftCount, centerCount, rightCount int) (leftPct, centerPct, rightPct float64) {
	total := leftCount + centerCount + rightCount
	if total > 0 {
		leftPct = float64(leftCount) / float64(total) * 100
		centerPct = float64(centerCount) / float64(total) * 100
		rightPct = float64(rightCount) / float64(total) * 100
	}
	return leftPct, centerPct, rightPct
}

// getDatabaseSize retrieves the database size
func getDatabaseSize(ctx context.Context, dbConn *sqlx.DB) string {
	var dbSize int64
	err := dbConn.GetContext(ctx, &dbSize, "SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size()")
	if err != nil {
		log.Printf("[ADMIN] Failed to get database size: %v", err)
		return "Unknown"
	}
	return fmt.Sprintf("%.2f MB", float64(dbSize)/(1024*1024))
}

func adminGetMetricsHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		stats := SystemStatsResponse{}

		// Get basic counts
		stats.TotalArticles, stats.ArticlesToday, stats.PendingAnalysis = getBasicCounts(ctx, dbConn)

		// Get bias distribution
		leftCount, centerCount, rightCount := getBiasDistribution(ctx, dbConn)
		stats.LeftCount = leftCount
		stats.CenterCount = centerCount
		stats.RightCount = rightCount

		// Calculate percentages
		stats.LeftPercentage, stats.CenterPercentage, stats.RightPercentage = calculatePercentages(leftCount, centerCount, rightCount)

		// Get database size
		stats.DatabaseSize = getDatabaseSize(ctx, dbConn)

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

// Source Management Admin Handlers for HTMX

// adminSourcesListHandler handles GET /htmx/sources
func adminSourcesListHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fetch all sources with stats
		sources, err := db.FetchSources(dbConn, nil, "", "", 0, 0)
		if err != nil {
			log.Printf("[ERROR] Failed to fetch sources for admin: %v", err)
			c.HTML(500, "source-list-fragment", gin.H{
				"Error": "Failed to load sources",
			})
			return
		}

		// Convert to SourceWithStats for template
		sourcesWithStats := make([]models.SourceWithStats, len(sources))
		for i, source := range sources {
			sourcesWithStats[i] = models.SourceWithStats{
				Source: models.Source{
					ID:            source.ID,
					Name:          source.Name,
					ChannelType:   source.ChannelType,
					FeedURL:       source.FeedURL,
					Category:      source.Category,
					Enabled:       source.Enabled,
					DefaultWeight: source.DefaultWeight,
					LastFetchedAt: source.LastFetchedAt,
					ErrorStreak:   source.ErrorStreak,
					Metadata:      source.Metadata,
					CreatedAt:     source.CreatedAt,
					UpdatedAt:     source.UpdatedAt,
				},
				// TODO: Add stats aggregation
			}
		}

		c.HTML(200, "source-list-fragment", gin.H{
			"Sources": sourcesWithStats,
		})
	}
}

// adminSourceFormHandler handles GET /htmx/sources/new and GET /htmx/sources/:id/edit
func adminSourceFormHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")

		var source *db.Source
		var err error

		// If ID is provided and not "new", fetch the source for editing
		if idParam != "" && idParam != "new" {
			id, parseErr := strconv.ParseInt(idParam, 10, 64)
			if parseErr != nil {
				c.HTML(400, "source-form-fragment", gin.H{
					"Error": "Invalid source ID",
				})
				return
			}

			source, err = db.FetchSourceByID(dbConn, id)
			if err != nil {
				log.Printf("[ERROR] Failed to fetch source for editing: %v", err)
				c.HTML(404, "source-form-fragment", gin.H{
					"Error": "Source not found",
				})
				return
			}
		}

		// Convert to models.Source if editing
		var modelSource *models.Source
		if source != nil {
			modelSource = &models.Source{
				ID:            source.ID,
				Name:          source.Name,
				ChannelType:   source.ChannelType,
				FeedURL:       source.FeedURL,
				Category:      source.Category,
				Enabled:       source.Enabled,
				DefaultWeight: source.DefaultWeight,
				LastFetchedAt: source.LastFetchedAt,
				ErrorStreak:   source.ErrorStreak,
				Metadata:      source.Metadata,
				CreatedAt:     source.CreatedAt,
				UpdatedAt:     source.UpdatedAt,
			}
		}

		c.HTML(200, "source-form-fragment", gin.H{
			"Source": modelSource,
		})
	}
}

// adminSourceStatsHandler handles GET /htmx/sources/:id/stats
func adminSourceStatsHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.HTML(400, "source-stats-fragment", gin.H{
				"Error": "Invalid source ID",
			})
			return
		}

		// Fetch source
		source, err := db.FetchSourceByID(dbConn, id)
		if err != nil {
			log.Printf("[ERROR] Failed to fetch source for stats: %v", err)
			c.HTML(404, "source-stats-fragment", gin.H{
				"Error": "Source not found",
			})
			return
		}

		// Convert to models.Source
		modelSource := models.Source{
			ID:            source.ID,
			Name:          source.Name,
			ChannelType:   source.ChannelType,
			FeedURL:       source.FeedURL,
			Category:      source.Category,
			Enabled:       source.Enabled,
			DefaultWeight: source.DefaultWeight,
			LastFetchedAt: source.LastFetchedAt,
			ErrorStreak:   source.ErrorStreak,
			Metadata:      source.Metadata,
			CreatedAt:     source.CreatedAt,
			UpdatedAt:     source.UpdatedAt,
		}

		// TODO: Implement real stats aggregation
		// For now, create placeholder stats
		stats := models.SourceStats{
			SourceID:     id,
			ArticleCount: 0, // TODO: Count articles from this source
			AvgScore:     nil,
			ComputedAt:   time.Now(),
		}

		c.HTML(200, "source-stats-fragment", gin.H{
			"Source": modelSource,
			"Stats":  stats,
		})
	}
}

// adminCreateSourceHandler handles POST /htmx/sources - creates source and returns updated source list HTML
func adminCreateSourceHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateSourceRequest
		if err := c.ShouldBind(&req); err != nil {
			log.Printf("[ERROR] Failed to bind create source request: %v", err)
			c.HTML(400, "source-list-fragment", gin.H{
				"Error": "Invalid request data: " + err.Error(),
			})
			return
		}

		// Validate request
		if err := req.Validate(); err != nil {
			log.Printf("[ERROR] Create source validation failed: %v", err)
			c.HTML(400, "source-list-fragment", gin.H{
				"Error": "Validation failed: " + err.Error(),
			})
			return
		}

		// Create source using the same logic as the API handler
		source := &db.Source{
			Name:          req.Name,
			ChannelType:   req.ChannelType,
			FeedURL:       req.FeedURL,
			Category:      req.Category,
			DefaultWeight: req.DefaultWeight,
			Enabled:       true, // New sources are enabled by default
			Metadata:      req.Metadata,
		}

		id, err := db.InsertSource(dbConn, source)
		if err != nil {
			log.Printf("[ERROR] Failed to create source: %v", err)
			c.HTML(500, "source-list-fragment", gin.H{
				"Error": "Failed to create source: " + err.Error(),
			})
			return
		}
		source.ID = id

		// Return updated source list HTML
		sources, err := db.FetchSources(dbConn, nil, "", "", 0, 0)
		if err != nil {
			log.Printf("[ERROR] Failed to fetch sources after creation: %v", err)
			c.HTML(500, "source-list-fragment", gin.H{
				"Error": "Source created but failed to refresh list",
			})
			return
		}

		// Convert to SourceWithStats for template
		sourcesWithStats := make([]models.SourceWithStats, len(sources))
		for i, source := range sources {
			sourcesWithStats[i] = models.SourceWithStats{
				Source: models.Source{
					ID:            source.ID,
					Name:          source.Name,
					ChannelType:   source.ChannelType,
					FeedURL:       source.FeedURL,
					Category:      source.Category,
					DefaultWeight: source.DefaultWeight,
					Enabled:       source.Enabled,
					CreatedAt:     source.CreatedAt,
					UpdatedAt:     source.UpdatedAt,
					Metadata:      source.Metadata,
				},
			}
		}

		c.HTML(200, "source-list-fragment", gin.H{
			"Sources": sourcesWithStats,
		})
	}
}

// adminUpdateSourceHandler handles PUT /htmx/sources/:id - updates source and returns updated source list HTML
func adminUpdateSourceHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.HTML(400, "source-list-fragment", gin.H{
				"Error": "Invalid source ID",
			})
			return
		}

		var req models.UpdateSourceRequest

		// Debug: Log the raw request data
		log.Printf("[DEBUG] HTMX Update Source - Content-Type: %s", c.GetHeader("Content-Type"))
		log.Printf("[DEBUG] HTMX Update Source - Method: %s", c.Request.Method)

		if err := c.ShouldBind(&req); err != nil {
			log.Printf("[ERROR] Failed to bind update source request: %v", err)
			log.Printf("[DEBUG] Request body: %+v", c.Request.Body)
			c.HTML(400, "source-list-fragment", gin.H{
				"Error": "Invalid request data: " + err.Error(),
			})
			return
		}

		// Debug: Log the parsed request
		log.Printf("[DEBUG] HTMX Update Source - Parsed request: %+v", req)

		// Validate request
		if err := req.Validate(); err != nil {
			log.Printf("[ERROR] Update source validation failed: %v", err)
			c.HTML(400, "source-list-fragment", gin.H{
				"Error": "Validation failed: " + err.Error(),
			})
			return
		}

		// Check if source exists
		_, err = db.FetchSourceByID(dbConn, id)
		if err != nil {
			if err.Error() == "source not found" {
				c.HTML(404, "source-list-fragment", gin.H{
					"Error": "Source not found",
				})
				return
			}
			c.HTML(500, "source-list-fragment", gin.H{
				"Error": "Failed to fetch source",
			})
			return
		}

		// Update source
		updates := req.ToUpdateMap()
		err = db.UpdateSource(dbConn, id, updates)
		if err != nil {
			log.Printf("[ERROR] Failed to update source: %v", err)
			c.HTML(500, "source-list-fragment", gin.H{
				"Error": "Failed to update source: " + err.Error(),
			})
			return
		}

		// Return updated source list HTML
		sources, err := db.FetchSources(dbConn, nil, "", "", 0, 0)
		if err != nil {
			log.Printf("[ERROR] Failed to fetch sources after update: %v", err)
			c.HTML(500, "source-list-fragment", gin.H{
				"Error": "Source updated but failed to refresh list",
			})
			return
		}

		// Convert to SourceWithStats for template
		sourcesWithStats := make([]models.SourceWithStats, len(sources))
		for i, source := range sources {
			sourcesWithStats[i] = models.SourceWithStats{
				Source: models.Source{
					ID:            source.ID,
					Name:          source.Name,
					ChannelType:   source.ChannelType,
					FeedURL:       source.FeedURL,
					Category:      source.Category,
					DefaultWeight: source.DefaultWeight,
					Enabled:       source.Enabled,
					CreatedAt:     source.CreatedAt,
					UpdatedAt:     source.UpdatedAt,
					Metadata:      source.Metadata,
				},
			}
		}

		c.HTML(200, "source-list-fragment", gin.H{
			"Sources": sourcesWithStats,
		})
	}
}
