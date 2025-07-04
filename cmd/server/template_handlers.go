package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"log"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// TemplateHandlers contains the internal API client for template rendering
// These handlers use internal API calls instead of HTTP to maintain API-first architecture
// while avoiding circular dependencies
type TemplateHandlers struct {
	client *api.InternalAPIClient
}

// NewTemplateHandlers creates a new handler instance with the internal API client
func NewTemplateHandlers(dbConn *sqlx.DB) *TemplateHandlers {
	return &TemplateHandlers{
		client: api.NewInternalAPIClient(dbConn),
	}
}

// TemplateIndexHandler handles the articles listing page using internal API client
func (h *TemplateHandlers) TemplateIndexHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get query parameters for filtering
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
		// Build API parameters
		params := api.InternalArticlesParams{
			Limit:  limit,
			Offset: offset,
		}

		if source != "" {
			params.Source = source
		}

		if bias != "" {
			params.Leaning = bias // Use Leaning field instead of Bias
		}

		// Note: Query parameter is not supported in current internal API client
		// This would need to be added to the API later

		// Get articles from internal API client
		articles, err := h.client.GetArticles(ctx, params)
		if err != nil {
			log.Printf("[DEBUG] TemplateIndexHandler ERROR path - Error fetching articles: %v", err)
			c.HTML(http.StatusInternalServerError, "articles.html", gin.H{
				"Error":          "Error fetching articles: " + err.Error(),
				"Articles":       []api.InternalArticle{}, // Pass empty slice
				"Sources":        []string{},
				"SearchQuery":    query,
				"SelectedSource": source,
				"SelectedBias":   bias,
				"CurrentPage":    1,        // Default value
				"TotalPages":     1,        // Default value
				"Pages":          []int{1}, // Default value
				"PrevPage":       0,        // Default value
				"NextPage":       0,        // Default value
			})
			log.Printf("[DEBUG] TemplateIndexHandler ERROR path - Error fetching articles: %v. CurrentPage type: %T, value: %v", err, 1, 1) // DEBUG
			return
		}

		// Simplified pagination since we don't have a count endpoint yet
		// Estimate based on whether we got a full page
		totalCount := offset + len(articles)
		if len(articles) == limit {
			totalCount += 1 // Assume there's at least one more page
		}

		totalPages := (totalCount + limit - 1) / limit

		// Generate page numbers for pagination
		var pages []int
		start := maxInt(1, page-2)
		end := minInt(totalPages, page+2)
		for i := start; i <= end; i++ {
			pages = append(pages, i)
		}

		// Get available sources by analyzing current articles
		sourceSet := make(map[string]bool)
		for _, article := range articles {
			sourceSet[article.Source] = true
		}
		var sources []string
		for s := range sourceSet {
			sources = append(sources, s)
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
		log.Printf("[DEBUG] TemplateIndexHandler SUCCESS path - CurrentPage type: %T, value: %v", page, page) // DEBUG
	}
}

// TemplateArticleHandler handles the individual article detail page using internal API client
func (h *TemplateHandlers) TemplateArticleHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "article.html", gin.H{
				"Error": "Invalid article ID",
			})
			return
		}
		// Get the article from internal API
		article, err := h.client.GetArticle(ctx, int64(id))
		if err != nil {
			c.HTML(http.StatusNotFound, "article.html", gin.H{
				"Error": "Article not found",
			})
			return
		}

		// Get recent articles for sidebar (excluding current article)
		recentParams := api.InternalArticlesParams{
			Limit: 6, // Get 6 so we can filter out current and have 5
		}
		recentArticles, err := h.client.GetArticles(ctx, recentParams)
		if err != nil {
			recentArticles = []api.InternalArticle{} // Fallback to empty list
		}

		// Filter out current article from recent articles
		var filteredRecent []api.InternalArticle
		for _, recent := range recentArticles {
			if recent.ID != int64(id) && len(filteredRecent) < 5 {
				filteredRecent = append(filteredRecent, recent)
			}
		}

		// Basic statistics - simplified for now
		stats := map[string]interface{}{
			"totalArticles": len(recentArticles),
			"currentTime":   ctx.Value("time"),
		}

		c.HTML(http.StatusOK, "article.html", gin.H{
			"Article":        article,
			"RecentArticles": filteredRecent,
			"Stats":          stats,
		})
	}
}

// TemplateAdminHandler handles the admin dashboard page using API client
func (h *TemplateHandlers) TemplateAdminHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Get system statistics
		stats, err := h.getDetailedStats(ctx)
		if err != nil {
			stats = make(map[string]interface{}) // Fallback to empty stats
		}

		// Get system status
		systemStatus, err := h.getSystemStatus(ctx)
		if err != nil {
			systemStatus = make(map[string]bool) // Fallback to empty status
		}

		// Get recent activity
		recentActivity, err := h.getRecentActivity(ctx)
		if err != nil {
			recentActivity = []map[string]interface{}{} // Fallback to empty activity
		}

		c.HTML(http.StatusOK, "admin.html", gin.H{
			"Stats":          stats,
			"SystemStatus":   systemStatus,
			"RecentActivity": recentActivity,
		})
	}
}

// HTMX Fragment Handlers for dynamic loading

// TemplateArticlesFragmentHandler returns just the article list for HTMX updates
func (h *TemplateHandlers) TemplateArticlesFragmentHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get query parameters for filtering (same as main handler)
		source := c.Query("source")
		bias := c.Query("bias")
		if bias == "" {
			bias = c.Query("leaning")
		}
		query := c.Query("query")
		pageStr := c.DefaultQuery("page", "1")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limit := 20
		offset := (page - 1) * limit

		// Build API parameters
		params := api.InternalArticlesParams{
			Limit:  limit,
			Offset: offset,
		}

		if source != "" {
			params.Source = source
		}

		if bias != "" {
			params.Leaning = bias
		}
		// Get articles from API
		articles, err := h.client.GetArticles(ctx, params)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "article-list-fragment", gin.H{
				"Error": "Error fetching articles: " + err.Error(),
			})
			return
		}

		// Calculate pagination (simplified)
		totalCount := offset + len(articles)
		if len(articles) == limit {
			totalCount += 1
		}
		totalPages := (totalCount + limit - 1) / limit

		var pages []int
		start := maxInt(1, page-2)
		end := minInt(totalPages, page+2)
		for i := start; i <= end; i++ {
			pages = append(pages, i)
		}
		// Return just the fragment
		c.HTML(http.StatusOK, "article-list-fragment", gin.H{
			"Articles":       articles,
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

// TemplateArticleFragmentHandler returns the full article page for HTMX navigation
func (h *TemplateHandlers) TemplateArticleFragmentHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "article_htmx.html", gin.H{
				"Error": "Invalid article ID",
			})
			return
		}
		// Get the article from API
		article, err := h.client.GetArticle(ctx, int64(id))
		if err != nil {
			c.HTML(http.StatusNotFound, "article_htmx.html", gin.H{
				"Error": "Article not found",
			})
			return
		}

		// Get recent articles for sidebar
		recentParams := api.InternalArticlesParams{Limit: 6}
		recentArticles, err := h.client.GetArticles(ctx, recentParams)
		if err != nil {
			recentArticles = []api.InternalArticle{}
		}

		// Filter out current article
		var filteredRecent []api.InternalArticle
		for _, recent := range recentArticles {
			if recent.ID != int64(id) && len(filteredRecent) < 5 {
				filteredRecent = append(filteredRecent, recent)
			}
		}

		// Get stats
		stats, err := h.getBasicStats(ctx)
		if err != nil {
			stats = make(map[string]interface{})
		}

		c.HTML(http.StatusOK, "article_htmx.html", gin.H{
			"Article":        article,
			"RecentArticles": filteredRecent,
			"Stats":          stats,
		})
	}
}

// TemplateArticleSummaryFragmentHandler returns article summary fragment
func (h *TemplateHandlers) TemplateArticleSummaryFragmentHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error-fragment", gin.H{
				"Error": "Invalid article ID",
			})
			return
		}

		// Get article and extract summary
		article, err := h.client.GetArticle(ctx, int64(id))
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error-fragment", gin.H{
				"Error": "Error fetching article: " + err.Error(),
			})
			return
		}

		c.HTML(http.StatusOK, "summary-fragment", gin.H{
			"Summary": article.Summary,
		})
	}
}

// Helper functions that use the API client

func (h *TemplateHandlers) getBasicStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get a sample of articles to calculate stats from
	// Since we don't have count endpoints, we'll estimate from a larger sample
	allParams := api.InternalArticlesParams{
		Limit: 1000, // Get a large sample to calculate from
	}
	allArticles, err := h.client.GetArticles(ctx, allParams)
	if err != nil {
		return nil, err
	}

	totalCount := len(allArticles)
	stats["TotalArticles"] = totalCount

	// Count by bias from the articles we have
	leftCount := 0
	centerCount := 0
	rightCount := 0

	for _, article := range allArticles {
		// Use composite score to determine bias
		if article.CompositeScore < -0.1 {
			leftCount++
		} else if article.CompositeScore > 0.1 {
			rightCount++
		} else {
			centerCount++
		}
	}

	stats["LeftCount"] = leftCount
	stats["CenterCount"] = centerCount
	stats["RightCount"] = rightCount

	// Calculate percentages
	if totalCount > 0 {
		stats["LeftPercentage"] = (leftCount * 100) / totalCount
		stats["CenterPercentage"] = (centerCount * 100) / totalCount
		stats["RightPercentage"] = (rightCount * 100) / totalCount
	} else {
		stats["LeftPercentage"] = 0
		stats["CenterPercentage"] = 0
		stats["RightPercentage"] = 0
	}

	return stats, nil
}

func (h *TemplateHandlers) getDetailedStats(ctx context.Context) (map[string]interface{}, error) {
	// Use the new admin metrics endpoint for comprehensive statistics
	// Since template handlers run on the same server, use localhost
	metricsURL := "http://localhost:8080/api/admin/metrics"

	req, err := http.NewRequestWithContext(ctx, "GET", metricsURL, nil)
	if err != nil {
		log.Printf("[TEMPLATE] Failed to create metrics request: %v", err)
		// Fallback to basic stats
		return h.getBasicStats(ctx)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[TEMPLATE] Failed to fetch admin metrics: %v", err)
		// Fallback to basic stats
		return h.getBasicStats(ctx)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("[TEMPLATE] Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[TEMPLATE] Admin metrics endpoint returned status %d", resp.StatusCode)
		// Fallback to basic stats
		return h.getBasicStats(ctx)
	}

	var metricsResponse struct {
		Success bool `json:"success"`
		Data    struct {
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
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&metricsResponse); err != nil {
		log.Printf("[TEMPLATE] Failed to decode metrics response: %v", err)
		// Fallback to basic stats
		return h.getBasicStats(ctx)
	}

	if !metricsResponse.Success {
		log.Printf("[TEMPLATE] Admin metrics endpoint returned success=false")
		// Fallback to basic stats
		return h.getBasicStats(ctx)
	}

	// Convert to map[string]interface{} for template compatibility
	stats := map[string]interface{}{
		"TotalArticles":    metricsResponse.Data.TotalArticles,
		"ArticlesToday":    metricsResponse.Data.ArticlesToday,
		"PendingAnalysis":  metricsResponse.Data.PendingAnalysis,
		"ActiveSources":    metricsResponse.Data.ActiveSources,
		"DatabaseSize":     metricsResponse.Data.DatabaseSize,
		"LeftCount":        metricsResponse.Data.LeftCount,
		"CenterCount":      metricsResponse.Data.CenterCount,
		"RightCount":       metricsResponse.Data.RightCount,
		"LeftPercentage":   metricsResponse.Data.LeftPercentage,
		"CenterPercentage": metricsResponse.Data.CenterPercentage,
		"RightPercentage":  metricsResponse.Data.RightPercentage,
	}

	return stats, nil
}

func (h *TemplateHandlers) getSystemStatus(ctx context.Context) (map[string]bool, error) {
	// Use the new admin health check endpoint for comprehensive system status
	// Since template handlers run on the same server, use localhost
	healthURL := "http://localhost:8080/api/admin/health-check"

	req, err := http.NewRequestWithContext(ctx, "POST", healthURL, nil)
	if err != nil {
		log.Printf("[TEMPLATE] Failed to create health check request: %v", err)
		// Fallback to basic status
		return h.getBasicSystemStatus(ctx)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[TEMPLATE] Failed to fetch admin health check: %v", err)
		// Fallback to basic status
		return h.getBasicSystemStatus(ctx)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("[TEMPLATE] Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[TEMPLATE] Admin health check endpoint returned status %d", resp.StatusCode)
		// Fallback to basic status
		return h.getBasicSystemStatus(ctx)
	}

	var healthResponse struct {
		Success bool `json:"success"`
		Data    struct {
			DatabaseOK   bool `json:"database_ok"`
			LLMServiceOK bool `json:"llm_service_ok"`
			RSSServiceOK bool `json:"rss_service_ok"`
			ServerOK     bool `json:"server_ok"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
		log.Printf("[TEMPLATE] Failed to decode health check response: %v", err)
		// Fallback to basic status
		return h.getBasicSystemStatus(ctx)
	}

	if !healthResponse.Success {
		log.Printf("[TEMPLATE] Admin health check endpoint returned success=false")
		// Fallback to basic status
		return h.getBasicSystemStatus(ctx)
	}

	// Convert to map[string]bool for template compatibility
	status := map[string]bool{
		"DatabaseOK":   healthResponse.Data.DatabaseOK,
		"LLMServiceOK": healthResponse.Data.LLMServiceOK,
		"RSSServiceOK": healthResponse.Data.RSSServiceOK,
		"ServerOK":     healthResponse.Data.ServerOK,
	}

	return status, nil
}

// getBasicSystemStatus provides fallback system status checking
func (h *TemplateHandlers) getBasicSystemStatus(ctx context.Context) (map[string]bool, error) {
	status := make(map[string]bool)

	// Test API connectivity by making a simple request
	params := api.InternalArticlesParams{Limit: 1}
	_, err := h.client.GetArticles(ctx, params)
	status["DatabaseOK"] = err == nil

	// Basic checks for other services
	status["RSSServiceOK"] = true // Simplified for internal client
	status["LLMServiceOK"] = true // Simplified for internal client
	status["ServerOK"] = true     // If we're running, server is OK

	return status, nil
}

func (h *TemplateHandlers) getRecentActivity(ctx context.Context) ([]map[string]interface{}, error) {
	var activities []map[string]interface{}

	// Get recent articles as activity
	params := api.InternalArticlesParams{
		Limit: 5,
	}
	articles, err := h.client.GetArticles(ctx, params)
	if err != nil {
		return activities, err
	}

	for _, article := range articles {
		activities = append(activities, map[string]interface{}{
			"Message":   "New article: " + article.Title,
			"Timestamp": article.PubDate,
		})
	}

	return activities, nil
}

// Helper functions for min/max (renamed to avoid conflicts)
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TemplateArticlesLoadMoreHandler returns just article items for load more functionality
func (h *TemplateHandlers) TemplateArticlesLoadMoreHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get query parameters for filtering (same as main handler)
		source := c.Query("source")
		bias := c.Query("bias")
		if bias == "" {
			bias = c.Query("leaning")
		}
		pageStr := c.DefaultQuery("page", "1")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limit := 20
		offset := (page - 1) * limit

		// Build API parameters
		params := api.InternalArticlesParams{
			Limit:  limit,
			Offset: offset,
		}

		if source != "" {
			params.Source = source
		}

		if bias != "" {
			params.Leaning = bias
		}

		// Get articles from API
		articles, err := h.client.GetArticles(ctx, params)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "article-items-fragment", gin.H{
				"Error": "Error fetching articles: " + err.Error(),
			})
			return
		}

		// Return just the article items for appending
		c.HTML(http.StatusOK, "article-items-fragment", gin.H{
			"Articles": articles,
		})
	}
}

// TemplateIndexHTMXHandler handles the articles listing page with HTMX support
func (h *TemplateHandlers) TemplateIndexHTMXHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get query parameters for filtering
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

		// Build API parameters
		params := api.InternalArticlesParams{
			Limit:  limit,
			Offset: offset,
		}

		if source != "" {
			params.Source = source
		}

		if bias != "" {
			params.Leaning = bias // Use Leaning field instead of Bias
		}

		// Get articles from internal API client
		articles, err := h.client.GetArticles(ctx, params)
		if err != nil {
			log.Printf("[DEBUG] TemplateIndexHTMXHandler ERROR path - Error fetching articles: %v", err)
			c.HTML(http.StatusInternalServerError, "articles_htmx.html", gin.H{
				"Error":          "Error fetching articles: " + err.Error(),
				"Articles":       []api.InternalArticle{}, // Pass empty slice
				"Sources":        []string{},
				"SearchQuery":    query,
				"SelectedSource": source,
				"SelectedBias":   bias,
				"CurrentPage":    1,        // Default value
				"TotalPages":     1,        // Default value
				"Pages":          []int{1}, // Default value
				"PrevPage":       0,        // Default value
				"NextPage":       0,        // Default value
			})
			return
		}

		// Simplified pagination since we don't have a count endpoint yet
		// Estimate based on whether we got a full page
		totalCount := offset + len(articles)
		if len(articles) == limit {
			totalCount += 1 // Assume there's at least one more page
		}

		totalPages := (totalCount + limit - 1) / limit

		// Generate page numbers for pagination
		var pages []int
		start := maxInt(1, page-2)
		end := minInt(totalPages, page+2)
		for i := start; i <= end; i++ {
			pages = append(pages, i)
		}

		// Get available sources by analyzing current articles
		sourceSet := make(map[string]bool)
		for _, article := range articles {
			sourceSet[article.Source] = true
		}
		var sources []string
		for s := range sourceSet {
			sources = append(sources, s)
		}

		// Use HTMX template instead of regular template
		c.HTML(http.StatusOK, "articles_htmx.html", gin.H{
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
		log.Printf("[DEBUG] TemplateIndexHTMXHandler SUCCESS path - CurrentPage type: %T, value: %v", page, page)
	}
}
