package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// Template file constants
const (
	indexTemplate   = "articles.html"
	articleTemplate = "article.html"
	adminTemplate   = "admin.html"
)

// TemplateData represents the data structure for templates
type TemplateData struct {
	Title          string
	SearchQuery    string
	Articles       []ArticleTemplateData
	Article        *ArticleTemplateData
	RecentArticles []ArticleTemplateData
	Stats          StatsData
	TotalArticles  int
	CurrentPage    int
	HasMore        bool
	Filters        FilterData
}

// ArticleTemplateData represents article data for templates
type ArticleTemplateData struct {
	ID             int64     `json:"id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	URL            string    `json:"url"`
	Source         string    `json:"source"`
	PubDate        time.Time `json:"pub_date"`
	CreatedAt      time.Time `json:"created_at"`
	CompositeScore float64   `json:"composite_score"`
	Confidence     float64   `json:"confidence"`
	ScoreSource    string    `json:"score_source"`
	Summary        string    `json:"summary,omitempty"`
	BiasLabel      string    `json:"bias_label,omitempty"`
	FormattedDate  string    `json:"formatted_date"`
	Excerpt        string    `json:"excerpt,omitempty"`
}

// StatsData represents statistics for the sidebar
type StatsData struct {
	TotalArticles int
	SourceCount   int
	LastUpdate    string
}

// FilterData represents filtering options
type FilterData struct {
	Source  string
	Leaning string
	Query   string
}

// AdminData represents data structure for admin dashboard
type AdminData struct {
	Title       string
	SystemStats SystemStatsData
	FeedHealth  map[string]bool
	Metrics     MetricsData
}

// SystemStatsData represents system statistics for admin dashboard
type SystemStatsData struct {
	TotalArticles     int
	TotalSources      int
	LastUpdate        string
	DatabaseSize      string
	ServerUptime      string
	ActiveConnections int
}

// MetricsData represents system metrics for admin dashboard
type MetricsData struct {
	ArticleProcessingRate float64
	AvgResponseTime       float64
	ErrorRate             float64
	CacheHitRate          float64
}

// FilterParams holds filter and pagination parameters
type FilterParams struct {
	Source      string
	Leaning     string
	Query       string
	Limit       int
	Offset      int
	CurrentPage int
}

// extractFilterParams extracts and validates filter parameters from gin context
func extractFilterParams(c *gin.Context) FilterParams {
	params := FilterParams{
		Source:  c.Query("source"),
		Leaning: c.Query("leaning"),
		Query:   c.Query("query"),
		Limit:   20,
		Offset:  0,
	}

	// Map bias to leaning for backward compatibility
	if bias := c.Query("bias"); bias != "" && params.Leaning == "" {
		params.Leaning = bias
	}

	// Handle pagination
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 1 {
			params.Offset = (page - 1) * params.Limit
			params.CurrentPage = page
		}
	}
	if params.CurrentPage == 0 {
		params.CurrentPage = 1
	}

	return params
}

// buildSearchQuery constructs SQL query for search with filters
func buildSearchQuery(params FilterParams) (string, []interface{}) {
	sqlQuery := `SELECT * FROM articles WHERE 1=1`
	var args []interface{}

	if params.Source != "" {
		sqlQuery += " AND source = ?"
		args = append(args, params.Source)
	}

	if params.Leaning != "" {
		switch params.Leaning {
		case "Left", "left":
			sqlQuery += " AND composite_score < -0.1"
		case "Right", "right":
			sqlQuery += " AND composite_score > 0.1"
		case "Center", "center":
			sqlQuery += " AND composite_score BETWEEN -0.1 AND 0.1"
		}
	}

	if params.Query != "" {
		sqlQuery += " AND (title LIKE ? OR content LIKE ?)"
		searchTerm := "%" + params.Query + "%"
		args = append(args, searchTerm, searchTerm)
	}

	sqlQuery += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit+1, params.Offset)

	return sqlQuery, args
}

// fetchArticlesWithFilters fetches articles based on filter parameters
func fetchArticlesWithFilters(dbConn *sqlx.DB, params FilterParams) ([]db.Article, error) {
	if params.Query != "" {
		sqlQuery, args := buildSearchQuery(params)
		var articles []db.Article
		err := dbConn.Select(&articles, sqlQuery, args...)
		return articles, err
	}

	return db.FetchArticles(dbConn, params.Source, params.Leaning, params.Limit+1, params.Offset)
}

// fetchRecentArticles fetches recent articles for sidebar
func fetchRecentArticles(dbConn *sqlx.DB) []ArticleTemplateData {
	recentArticles, err := db.FetchArticles(dbConn, "", "", 5, 0)
	if err != nil {
		log.Printf("Error fetching recent articles: %v", err)
		return []ArticleTemplateData{}
	}

	recentTemplateArticles := make([]ArticleTemplateData, len(recentArticles))
	for i, article := range recentArticles {
		recentTemplateArticles[i] = convertToTemplateData(&article)
	}
	return recentTemplateArticles
}

// convertArticlesToTemplateData converts db articles to template data
func convertArticlesToTemplateData(articles []db.Article) []ArticleTemplateData {
	templateArticles := make([]ArticleTemplateData, len(articles))
	for i, article := range articles {
		templateArticles[i] = convertToTemplateData(&article)
	}
	return templateArticles
}

// templateIndexHandler serves the main articles list page using Editorial template
func templateIndexHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := extractFilterParams(c)

		// Fetch articles from database
		articles, err := fetchArticlesWithFilters(dbConn, params)
		if err != nil {
			log.Printf("Error fetching articles for template: %v", err)
			c.HTML(http.StatusInternalServerError, indexTemplate, TemplateData{
				Title: "Error - NewsBalancer",
			})
			return
		}

		// Check if there are more articles
		hasMore := len(articles) > params.Limit
		if hasMore {
			articles = articles[:params.Limit] // Remove the extra article
		}

		// Convert to template data
		templateArticles := convertArticlesToTemplateData(articles)
		recentTemplateArticles := fetchRecentArticles(dbConn)

		// Prepare template data
		data := TemplateData{
			Title:          "NewsBalancer - Balanced News Analysis",
			SearchQuery:    params.Query,
			Articles:       templateArticles,
			RecentArticles: recentTemplateArticles,
			Stats:          getStats(dbConn),
			TotalArticles:  len(templateArticles),
			CurrentPage:    params.CurrentPage,
			HasMore:        hasMore,
			Filters: FilterData{
				Source:  params.Source,
				Leaning: params.Leaning,
				Query:   params.Query,
			},
		}

		c.HTML(http.StatusOK, indexTemplate, data)
	}
}

// calculateCompositeScore calculates composite score and confidence from LLM scores
func calculateCompositeScore(dbConn *sqlx.DB, articleID int64, templateArticle *ArticleTemplateData) {
	scores, err := db.FetchLLMScores(dbConn, articleID)
	if err != nil || len(scores) == 0 {
		return
	}

	var weightedSum, sumWeights float64
	for _, s := range scores {
		var meta struct {
			Confidence float64 `json:"confidence"`
		}
		_ = json.Unmarshal([]byte(s.Metadata), &meta)
		weightedSum += s.Score * meta.Confidence
		sumWeights += meta.Confidence
	}

	if sumWeights > 0 {
		templateArticle.CompositeScore = weightedSum / sumWeights
		templateArticle.Confidence = sumWeights / float64(len(scores))
	}
}

// fetchArticleSummary fetches article summary from LLM scores
func fetchArticleSummary(dbConn *sqlx.DB, articleID int64) string {
	allScores, err := db.FetchLLMScores(dbConn, articleID)
	if err != nil {
		return ""
	}

	for _, score := range allScores {
		if score.Model == "summarizer" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(score.Metadata), &meta); err == nil {
				if summaryText, ok := meta["summary"].(string); ok {
					return summaryText
				}
			}
		}
	}
	return ""
}

// enhanceTemplateArticle enhances template article with scores and summary
func enhanceTemplateArticle(dbConn *sqlx.DB, templateArticle *ArticleTemplateData, articleID int64) {
	calculateCompositeScore(dbConn, articleID, templateArticle)
	templateArticle.BiasLabel = getBiasLabel(templateArticle.CompositeScore)
	templateArticle.Summary = fetchArticleSummary(dbConn, articleID)
}

// templateArticleHandler serves individual article page using Editorial template
func templateArticleHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.HTML(http.StatusBadRequest, articleTemplate, TemplateData{
				Title: "Invalid Article ID - NewsBalancer",
			})
			return
		}

		// Fetch article from database
		article, err := db.FetchArticleByID(dbConn, id)
		if err != nil {
			log.Printf("Error fetching article %d: %v", id, err)
			c.HTML(http.StatusNotFound, articleTemplate, TemplateData{
				Title: "Article Not Found - NewsBalancer",
			})
			return
		}

		// Convert to template data and enhance with scores
		templateArticle := convertToTemplateData(article)
		enhanceTemplateArticle(dbConn, &templateArticle, article.ID)

		// Fetch recent articles and prepare data
		recentTemplateArticles := fetchRecentArticles(dbConn)

		// Prepare template data
		data := TemplateData{
			Title:          templateArticle.Title + " - NewsBalancer",
			Article:        &templateArticle,
			RecentArticles: recentTemplateArticles,
			Stats:          getStats(dbConn),
		}

		c.HTML(http.StatusOK, articleTemplate, data)
	}
}

// convertToTemplateData converts db.Article to ArticleTemplateData
func convertToTemplateData(article *db.Article) ArticleTemplateData {
	templateData := ArticleTemplateData{
		ID:            article.ID,
		Title:         article.Title,
		Content:       article.Content,
		URL:           article.URL,
		Source:        article.Source,
		PubDate:       article.PubDate,
		CreatedAt:     article.CreatedAt,
		ScoreSource:   getStringValue(article.ScoreSource),
		FormattedDate: article.PubDate.Format("January 2, 2006"),
	}

	// Handle pointers safely
	if article.CompositeScore != nil {
		templateData.CompositeScore = *article.CompositeScore
	}
	if article.Confidence != nil {
		templateData.Confidence = *article.Confidence
	}

	// Create excerpt from content (first 200 characters)
	if len(article.Content) > 200 {
		templateData.Excerpt = article.Content[:200] + "..."
	} else {
		templateData.Excerpt = article.Content
	}

	// Set bias label
	templateData.BiasLabel = getBiasLabel(templateData.CompositeScore)

	return templateData
}

// getStringValue safely extracts string value from pointer
func getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

// getBiasLabel converts numeric bias score to human-readable label
func getBiasLabel(score float64) string {
	if score < -0.3 {
		return "Left Leaning"
	} else if score > 0.3 {
		return "Right Leaning"
	}
	return "Center"
}

// getStats fetches system statistics for templates
func getStats(dbConn *sqlx.DB) StatsData {
	var totalArticles int
	var sourceCount int

	// Get total articles count
	err := dbConn.Get(&totalArticles, "SELECT COUNT(*) FROM articles")
	if err != nil {
		log.Printf("Error fetching total articles count: %v", err)
		totalArticles = 0
	}

	// Get unique source count
	err = dbConn.Get(&sourceCount, "SELECT COUNT(DISTINCT source) FROM articles")
	if err != nil {
		log.Printf("Error fetching source count: %v", err)
		sourceCount = 0
	}

	// Get current timestamp for last update
	lastUpdate := time.Now().Format("January 2, 2006 at 3:04 PM")

	return StatsData{
		TotalArticles: totalArticles,
		SourceCount:   sourceCount,
		LastUpdate:    lastUpdate,
	}
}

// templateAdminHandler serves the admin dashboard page
func templateAdminHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fetch system statistics
		stats := getStats(dbConn)

		// Get detailed system stats for admin
		systemStats := SystemStatsData{
			TotalArticles:     stats.TotalArticles,
			TotalSources:      stats.SourceCount,
			LastUpdate:        stats.LastUpdate,
			DatabaseSize:      "Unknown", // Could implement db size calculation
			ServerUptime:      "Unknown", // Could implement uptime tracking
			ActiveConnections: 1,         // Placeholder
		}

		// Mock feed health data (would integrate with actual feed health check)
		feedHealth := map[string]bool{
			"Reuters":  true,
			"BBC News": true,
			"CNN":      false, // Example unhealthy feed
		}

		// Mock metrics data (would integrate with actual metrics)
		metrics := MetricsData{
			ArticleProcessingRate: 15.2,
			AvgResponseTime:       8.5,
			ErrorRate:             2.1,
			CacheHitRate:          85.3,
		}

		// Prepare admin template data
		data := AdminData{
			Title:       "Admin Dashboard - NewsBalancer",
			SystemStats: systemStats,
			FeedHealth:  feedHealth,
			Metrics:     metrics,
		}

		c.HTML(http.StatusOK, adminTemplate, data)
	}
}

// Change these handler functions to exported (capitalized) so they can be used in main.go
func TemplateIndexHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return templateIndexHandler(dbConn)
}
func TemplateArticleHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return templateArticleHandler(dbConn)
}
func TemplateAdminHandler(dbConn *sqlx.DB) gin.HandlerFunc {
	return templateAdminHandler(dbConn)
}
