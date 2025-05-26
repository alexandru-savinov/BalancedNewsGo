package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// findTemplateGlob dynamically finds the template directory
func findTemplateGlob() string {
	// Try different possible paths relative to where tests might run
	candidates := []string{
		"../../web/templates/*", // from cmd/server/
		"./web/templates/*",     // from project root
		"web/templates/*",       // from project root without ./
	}

	for _, candidate := range candidates {
		// Check if the directory exists by removing the /* and checking the dir
		dir := strings.TrimSuffix(candidate, "/*")
		if _, err := os.Stat(dir); err == nil {
			// Double check that there are actually template files
			if matches, _ := filepath.Glob(candidate); len(matches) > 0 {
				return candidate
			}
		}
	}

	// For CI environments or when templates are missing, skip template tests
	return ""
}

// Test constants to avoid duplication
const (
	testSource          = "test-source"
	testSummary         = "This is a test summary of the article."
	testArticleTitle    = "Test Article"
	testArticleURL      = "https://example.com"
	testArticleContent  = "Test content"
	skipTemplateMessage = "Templates not available in this environment - skipping template tests"
	queryLLMScores      = "SELECT \\* FROM llm_scores WHERE article_id = \\?"
	queryCountArticles  = "SELECT COUNT\\(\\*\\) FROM articles"
	queryCountSources   = "SELECT COUNT\\(DISTINCT source\\) FROM articles"
	querySelectArticles = "SELECT \\* FROM articles"
	templateGlob        = "../../web/templates/*"
	articleIDRoute      = "/article/:id"
)

// setupTestDB creates a test database with sqlmock
func setupTestDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlite3")
	return sqlxDB, mock
}

// createTestArticle creates a sample article for testing
func createTestArticle(id int64) db.Article {
	now := time.Now()
	score := 0.2
	confidence := 0.8
	scoreSource := "test_model"

	return db.Article{
		ID:             id,
		Title:          fmt.Sprintf("Test Article %d", id),
		Content:        fmt.Sprintf("This is test content for article %d. It contains enough text to test excerpt functionality.", id),
		URL:            fmt.Sprintf("https://example.com/article-%d", id),
		Source:         testSource,
		PubDate:        now.Add(-time.Hour),
		CreatedAt:      now,
		CompositeScore: &score,
		Confidence:     &confidence,
		ScoreSource:    &scoreSource,
	}
}

// createTestLLMScore creates a sample LLM score for testing
func createTestLLMScore(articleID int64, model string, score float64) db.LLMScore {
	metadata := map[string]interface{}{
		"confidence": 0.8,
	}
	if model == "summarizer" {
		metadata["summary"] = testSummary
	}
	metadataJSON, _ := json.Marshal(metadata)

	return db.LLMScore{
		ID:        1,
		ArticleID: articleID,
		Model:     model,
		Score:     score,
		Metadata:  string(metadataJSON),
		CreatedAt: time.Now(),
	}
}

func TestExtractFilterParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryParams    map[string]string
		expectedParams FilterParams
	}{
		{
			name:        "default parameters",
			queryParams: map[string]string{},
			expectedParams: FilterParams{
				Source:      "",
				Leaning:     "",
				Query:       "",
				Limit:       20,
				Offset:      0,
				CurrentPage: 1,
			},
		},
		{
			name: "with all filters",
			queryParams: map[string]string{
				"source":  testSource,
				"leaning": "Left",
				"query":   "test query",
				"page":    "2",
			},
			expectedParams: FilterParams{
				Source:      testSource,
				Leaning:     "Left",
				Query:       "test query",
				Limit:       20,
				Offset:      20,
				CurrentPage: 2,
			},
		},
		{
			name: "bias parameter backward compatibility",
			queryParams: map[string]string{
				"bias": "Right",
			},
			expectedParams: FilterParams{
				Source:      "",
				Leaning:     "Right",
				Query:       "",
				Limit:       20,
				Offset:      0,
				CurrentPage: 1,
			},
		},
		{
			name: "leaning overrides bias",
			queryParams: map[string]string{
				"bias":    "Right",
				"leaning": "Left",
			},
			expectedParams: FilterParams{
				Source:      "",
				Leaning:     "Left",
				Query:       "",
				Limit:       20,
				Offset:      0,
				CurrentPage: 1,
			},
		},
		{
			name: "invalid page number",
			queryParams: map[string]string{
				"page": "invalid",
			},
			expectedParams: FilterParams{
				Source:      "",
				Leaning:     "",
				Query:       "",
				Limit:       20,
				Offset:      0,
				CurrentPage: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request with query parameters
			req := httptest.NewRequest("GET", "/", nil)
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			// Create gin context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Test the function
			params := extractFilterParams(c)
			assert.Equal(t, tt.expectedParams, params)
		})
	}
}

func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		name           string
		params         FilterParams
		expectedQuery  string
		expectedArgLen int
	}{
		{
			name: "basic query with no filters",
			params: FilterParams{
				Limit:  20,
				Offset: 0,
			},
			expectedQuery:  "SELECT * FROM articles WHERE 1=1 ORDER BY created_at DESC LIMIT ? OFFSET ?",
			expectedArgLen: 2,
		},
		{
			name: "with source filter",
			params: FilterParams{
				Source: testSource,
				Limit:  20,
				Offset: 0,
			},
			expectedQuery:  "SELECT * FROM articles WHERE 1=1 AND source = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
			expectedArgLen: 3,
		},
		{
			name: "with left leaning filter",
			params: FilterParams{
				Leaning: "Left",
				Limit:   20,
				Offset:  0,
			},
			expectedQuery:  "SELECT * FROM articles WHERE 1=1 AND composite_score < -0.1 ORDER BY created_at DESC LIMIT ? OFFSET ?",
			expectedArgLen: 2,
		},
		{
			name: "with right leaning filter",
			params: FilterParams{
				Leaning: "Right",
				Limit:   20,
				Offset:  0,
			},
			expectedQuery:  "SELECT * FROM articles WHERE 1=1 AND composite_score > 0.1 ORDER BY created_at DESC LIMIT ? OFFSET ?",
			expectedArgLen: 2,
		},
		{
			name: "with center leaning filter",
			params: FilterParams{
				Leaning: "Center",
				Limit:   20,
				Offset:  0,
			},
			expectedQuery:  "SELECT * FROM articles WHERE 1=1 AND composite_score BETWEEN -0.1 AND 0.1 ORDER BY created_at DESC LIMIT ? OFFSET ?",
			expectedArgLen: 2,
		},
		{
			name: "with search query",
			params: FilterParams{
				Query:  "test search",
				Limit:  20,
				Offset: 0,
			},
			expectedQuery:  "SELECT * FROM articles WHERE 1=1 AND (title LIKE ? OR content LIKE ?) ORDER BY created_at DESC LIMIT ? OFFSET ?",
			expectedArgLen: 4,
		},
		{
			name: "with all filters",
			params: FilterParams{
				Source:  testSource,
				Leaning: "Left",
				Query:   "test search",
				Limit:   20,
				Offset:  0,
			},
			expectedQuery: "SELECT * FROM articles WHERE 1=1 AND source = ? AND composite_score < -0.1 " +
				"AND (title LIKE ? OR content LIKE ?) ORDER BY created_at DESC LIMIT ? OFFSET ?",
			expectedArgLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := buildSearchQuery(tt.params)
			assert.Equal(t, tt.expectedQuery, query)
			assert.Len(t, args, tt.expectedArgLen)
		})
	}
}

func TestConvertToTemplateData(t *testing.T) {
	article := createTestArticle(1)

	templateData := convertToTemplateData(&article)

	assert.Equal(t, article.ID, templateData.ID)
	assert.Equal(t, article.Title, templateData.Title)
	assert.Equal(t, article.Content, templateData.Content)
	assert.Equal(t, article.URL, templateData.URL)
	assert.Equal(t, article.Source, templateData.Source)
	assert.Equal(t, article.PubDate, templateData.PubDate)
	assert.Equal(t, article.CreatedAt, templateData.CreatedAt)
	assert.Equal(t, *article.CompositeScore, templateData.CompositeScore)
	assert.Equal(t, *article.Confidence, templateData.Confidence)
	assert.Equal(t, *article.ScoreSource, templateData.ScoreSource)
	assert.Equal(t, article.PubDate.Format("January 2, 2006"), templateData.FormattedDate)
	assert.Contains(t, templateData.Excerpt, "This is test content")
	assert.Equal(t, "Center", templateData.BiasLabel)
}

func TestConvertToTemplateDataWithNilPointers(t *testing.T) {
	article := db.Article{
		ID:        1,
		Title:     testArticleTitle,
		Content:   "Short content",
		URL:       testArticleURL,
		Source:    testSource,
		PubDate:   time.Now(),
		CreatedAt: time.Now(),
		// Nil pointers
		CompositeScore: nil,
		Confidence:     nil,
		ScoreSource:    nil,
	}

	templateData := convertToTemplateData(&article)

	assert.Equal(t, float64(0), templateData.CompositeScore)
	assert.Equal(t, float64(0), templateData.Confidence)
	assert.Equal(t, "", templateData.ScoreSource)
	assert.Equal(t, "Short content", templateData.Excerpt) // No truncation
}

func TestConvertArticlesToTemplateData(t *testing.T) {
	articles := []db.Article{
		createTestArticle(1),
		createTestArticle(2),
	}

	templateArticles := convertArticlesToTemplateData(articles)

	assert.Len(t, templateArticles, 2)
	assert.Equal(t, int64(1), templateArticles[0].ID)
	assert.Equal(t, int64(2), templateArticles[1].ID)
}

func TestFetchArticleSummary(t *testing.T) {
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	articleID := int64(1)
	summaryScore := createTestLLMScore(articleID, "summarizer", 0.5)
	nonSummaryScore := createTestLLMScore(articleID, "other_model", 0.3)

	// Mock the query for LLM scores
	rows := sqlmock.NewRows([]string{"id", "article_id", "model", "score", "metadata", "created_at"}).
		AddRow(summaryScore.ID, summaryScore.ArticleID, summaryScore.Model, summaryScore.Score, summaryScore.Metadata, summaryScore.CreatedAt).
		AddRow(nonSummaryScore.ID, nonSummaryScore.ArticleID, nonSummaryScore.Model,
			nonSummaryScore.Score, nonSummaryScore.Metadata, nonSummaryScore.CreatedAt)

	mock.ExpectQuery(queryLLMScores).
		WithArgs(articleID).
		WillReturnRows(rows)

	summary := fetchArticleSummary(dbConn, articleID)

	assert.Equal(t, testSummary, summary)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFetchArticleSummaryNoSummary(t *testing.T) {
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	articleID := int64(1)

	// Mock empty result
	rows := sqlmock.NewRows([]string{"id", "article_id", "model", "score", "metadata", "created_at"})
	mock.ExpectQuery(queryLLMScores).
		WithArgs(articleID).
		WillReturnRows(rows)

	summary := fetchArticleSummary(dbConn, articleID)

	assert.Equal(t, "", summary)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCalculateCompositeScore(t *testing.T) {
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	articleID := int64(1)
	templateArticle := &ArticleTemplateData{ID: articleID}

	score1 := createTestLLMScore(articleID, "model1", 0.5)
	score2 := createTestLLMScore(articleID, "model2", -0.3)

	// Mock the query for LLM scores
	rows := sqlmock.NewRows([]string{"id", "article_id", "model", "score", "metadata", "created_at"}).
		AddRow(score1.ID, score1.ArticleID, score1.Model, score1.Score, score1.Metadata, score1.CreatedAt).
		AddRow(score2.ID, score2.ArticleID, score2.Model, score2.Score, score2.Metadata, score2.CreatedAt)

	mock.ExpectQuery(queryLLMScores).
		WithArgs(articleID).
		WillReturnRows(rows)

	calculateCompositeScore(dbConn, articleID, templateArticle)

	// Expected weighted average: (0.5 * 0.8 + (-0.3) * 0.8) / (0.8 + 0.8) = 0.16 / 1.6 = 0.1
	assert.InDelta(t, 0.1, templateArticle.CompositeScore, 0.0000001)
	assert.Equal(t, 0.8, templateArticle.Confidence)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStats(t *testing.T) {
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	// Mock total articles count
	mock.ExpectQuery(queryCountArticles).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	// Mock unique source count
	mock.ExpectQuery(queryCountSources).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	stats := getStats(dbConn)

	assert.Equal(t, 100, stats.TotalArticles)
	assert.Equal(t, 5, stats.SourceCount)
	assert.NotEmpty(t, stats.LastUpdate)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFetchArticlesWithFilters(t *testing.T) {
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	params := FilterParams{
		Source:      testSource,
		Leaning:     "Left",
		Query:       "test",
		Limit:       20,
		Offset:      0,
		CurrentPage: 1,
	}

	article1 := createTestArticle(1)
	article2 := createTestArticle(2)

	articleRows := sqlmock.NewRows([]string{"id", "title", "content", "url", "source", "pub_date", "created_at", "composite_score", "confidence", "score_source"}).
		AddRow(article1.ID, article1.Title, article1.Content, article1.URL, article1.Source, article1.PubDate, article1.CreatedAt, article1.CompositeScore, article1.Confidence, article1.ScoreSource).
		AddRow(article2.ID, article2.Title, article2.Content, article2.URL, article2.Source, article2.PubDate, article2.CreatedAt, article2.CompositeScore, article2.Confidence, article2.ScoreSource)

	actualSQLQuery, _ := buildSearchQuery(params) // We only need the query string for ExpectQuery
	t.Logf("DEBUG: Actual SQL Query from buildSearchQuery: %s", actualSQLQuery)
	// Args for WithArgs must match how buildSearchQuery constructs them
	t.Logf("DEBUG: Manual Args for WithArgs: %#v",
		[]interface{}{params.Source, "%" + params.Query + "%", "%" + params.Query + "%", params.Limit + 1, params.Offset})

	mock.ExpectQuery(regexp.QuoteMeta(actualSQLQuery)).
		WithArgs(params.Source, "%"+params.Query+"%", "%"+params.Query+"%", params.Limit+1, params.Offset).
		WillReturnRows(articleRows)

	articles, err := fetchArticlesWithFilters(dbConn, params)
	if err != nil {
		t.Logf("DEBUG: Error from fetchArticlesWithFilters: %v", err)
	}

	t.Logf("DEBUG: Fetched articles count: %d", len(articles))

	require.Len(t, articles, 2, "Should fetch 2 articles based on mock")
	assert.Equal(t, article1.ID, articles[0].ID)
	assert.Equal(t, article2.ID, articles[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFetchRecentArticles(t *testing.T) {
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	// Mock recent articles query
	article := createTestArticle(1)
	recentRows := sqlmock.NewRows([]string{"id", "title", "content", "url", "source",
		"pub_date", "created_at", "composite_score", "confidence", "score_source"})
	recentRows.AddRow(article.ID, article.Title, article.Content, article.URL, article.Source, article.PubDate, article.CreatedAt, article.CompositeScore, article.Confidence, article.ScoreSource)

	mock.ExpectQuery(querySelectArticles).
		WillReturnRows(recentRows)

	recentArticles := fetchRecentArticles(dbConn)

	assert.Len(t, recentArticles, 1)
	assert.Equal(t, article.ID, recentArticles[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnhanceTemplateArticle(t *testing.T) {
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	articleID := int64(1)
	templateArticle := &ArticleTemplateData{ID: articleID}

	// Mock LLM scores for composite score calculation
	score := createTestLLMScore(articleID, "test_model", 0.5)
	summaryScore := createTestLLMScore(articleID, "summarizer", 0.3)

	scoresRows := sqlmock.NewRows([]string{"id", "article_id", "model", "score", "metadata", "created_at"}).
		AddRow(score.ID, score.ArticleID, score.Model, score.Score, score.Metadata, score.CreatedAt)

	mock.ExpectQuery(queryLLMScores).
		WithArgs(articleID).
		WillReturnRows(scoresRows)

	// Mock LLM scores for summary
	summaryRows := sqlmock.NewRows([]string{"id", "article_id", "model", "score", "metadata", "created_at"}).
		AddRow(summaryScore.ID, summaryScore.ArticleID, summaryScore.Model, summaryScore.Score, summaryScore.Metadata, summaryScore.CreatedAt)

	mock.ExpectQuery(queryLLMScores).
		WithArgs(articleID).
		WillReturnRows(summaryRows)

	enhanceTemplateArticle(dbConn, templateArticle, articleID)

	assert.Equal(t, 0.5, templateArticle.CompositeScore)
	assert.Equal(t, 0.8, templateArticle.Confidence)
	assert.Equal(t, "Right Leaning", templateArticle.BiasLabel)
	assert.Equal(t, testSummary, templateArticle.Summary)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTemplateIndexHandler(t *testing.T) {
	templateGlob := findTemplateGlob()
	if templateGlob == "" {
		t.Skip(skipTemplateMessage)
	}

	gin.SetMode(gin.TestMode)
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	// Mock articles query
	article1 := createTestArticle(1)
	article2 := createTestArticle(2)

	articleRows := sqlmock.NewRows([]string{"id", "title", "content", "url", "source", "pub_date", "created_at", "composite_score", "confidence", "score_source"}).
		AddRow(article1.ID, article1.Title, article1.Content, article1.URL, article1.Source,
			article1.PubDate, article1.CreatedAt, article1.CompositeScore, article1.Confidence,
			article1.ScoreSource).
		AddRow(article2.ID, article2.Title, article2.Content, article2.URL, article2.Source,
			article2.PubDate, article2.CreatedAt, article2.CompositeScore, article2.Confidence,
			article2.ScoreSource)

	mock.ExpectQuery(querySelectArticles).
		WillReturnRows(articleRows)

	// Mock recent articles query
	recentRows := sqlmock.NewRows([]string{"id", "title", "content", "url", "source",
		"pub_date", "created_at", "composite_score", "confidence", "score_source"})
	recentRows.AddRow(article1.ID, article1.Title, article1.Content, article1.URL, article1.Source, article1.PubDate, article1.CreatedAt, article1.CompositeScore, article1.Confidence, article1.ScoreSource)

	mock.ExpectQuery(querySelectArticles).
		WillReturnRows(recentRows)

	// Mock stats queries
	mock.ExpectQuery(queryCountArticles).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
	mock.ExpectQuery(queryCountSources).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Setup router and handler
	router := gin.New()
	router.SetFuncMap(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"mul":   func(a, b float64) float64 { return a * b },
		"split": func(s, sep string) []string { return strings.Split(s, sep) },
	})
	router.LoadHTMLGlob(templateGlob)
	router.GET("/", templateIndexHandler(dbConn))

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTemplateIndexHandlerWithFilters(t *testing.T) {
	templateGlob := findTemplateGlob()
	if templateGlob == "" {
		t.Skip(skipTemplateMessage)
	}

	gin.SetMode(gin.TestMode)
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	// Mock search query with filters
	searchRows := sqlmock.NewRows([]string{"id", "title", "content", "url", "source",
		"pub_date", "created_at", "composite_score", "confidence", "score_source"}).
		AddRow(1, testArticleTitle, testArticleContent, testArticleURL, testSource, time.Now(), time.Now(), 0.2, 0.8, "test_model")

	mock.ExpectQuery("SELECT \\* FROM articles WHERE 1=1").
		WithArgs("%test%", "%test%", 21, 0).
		WillReturnRows(searchRows)

	// Mock recent articles query
	recentRows := sqlmock.NewRows([]string{"id", "title", "content", "url", "source",
		"pub_date", "created_at", "composite_score", "confidence", "score_source"})
	recentRows.AddRow(1, testArticleTitle, testArticleContent, testArticleURL, testSource, time.Now(), time.Now(), 0.2, 0.8, "test_model")
	mock.ExpectQuery(querySelectArticles).
		WillReturnRows(recentRows)

	// Mock stats queries
	mock.ExpectQuery(queryCountArticles).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
	mock.ExpectQuery(queryCountSources).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	// Setup router and handler
	router := gin.New()
	router.SetFuncMap(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"mul":   func(a, b float64) float64 { return a * b },
		"split": func(s, sep string) []string { return strings.Split(s, sep) },
	})
	router.LoadHTMLGlob(templateGlob)
	router.GET("/", templateIndexHandler(dbConn))

	// Create test request with query parameter
	req := httptest.NewRequest("GET", "/?query=test", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTemplateArticleHandler(t *testing.T) {
	templateGlob := findTemplateGlob()
	if templateGlob == "" {
		t.Skip(skipTemplateMessage)
	}

	gin.SetMode(gin.TestMode)
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	article := createTestArticle(1)

	// Mock article fetch
	articleRows := sqlmock.NewRows([]string{"id", "title", "content", "url", "source", "pub_date", "created_at", "composite_score", "confidence", "score_source"}).
		AddRow(article.ID, article.Title, article.Content, article.URL, article.Source,
			article.PubDate, article.CreatedAt, article.CompositeScore, article.Confidence,
			article.ScoreSource)

	mock.ExpectQuery("SELECT \\* FROM articles WHERE id = \\?").
		WithArgs(int64(1)).
		WillReturnRows(articleRows)

	// Mock LLM scores for enhancement
	scoresRows := sqlmock.NewRows([]string{"id", "article_id", "model", "score", "metadata", "created_at"})
	mock.ExpectQuery(queryLLMScores).
		WithArgs(int64(1)).
		WillReturnRows(scoresRows)

	// Mock LLM scores for summary
	summaryRows := sqlmock.NewRows([]string{"id", "article_id", "model", "score", "metadata", "created_at"})
	mock.ExpectQuery(queryLLMScores).
		WithArgs(int64(1)).
		WillReturnRows(summaryRows)

	// Mock recent articles query
	recentRows := sqlmock.NewRows([]string{"id", "title", "content", "url", "source",
		"pub_date", "created_at", "composite_score", "confidence", "score_source"})
	recentRows.AddRow(article.ID, article.Title, article.Content, article.URL, article.Source, article.PubDate, article.CreatedAt, article.CompositeScore, article.Confidence, article.ScoreSource)

	mock.ExpectQuery(querySelectArticles).
		WillReturnRows(recentRows)
	// Mock stats queries
	mock.ExpectQuery(queryCountArticles).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
	mock.ExpectQuery(queryCountSources).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Setup router and handler
	router := gin.New()
	router.SetFuncMap(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"mul":   func(a, b float64) float64 { return a * b },
		"split": func(s, sep string) []string { return strings.Split(s, sep) },
	})
	router.LoadHTMLGlob(templateGlob)
	router.GET(articleIDRoute, templateArticleHandler(dbConn))

	// Create test request
	req := httptest.NewRequest("GET", "/article/1", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTemplateArticleHandlerInvalidID(t *testing.T) {
	templateGlob := findTemplateGlob()
	if templateGlob == "" {
		t.Skip(skipTemplateMessage)
	}

	gin.SetMode(gin.TestMode)
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	// Setup router and handler
	router := gin.New()
	router.SetFuncMap(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"mul":   func(a, b float64) float64 { return a * b },
		"split": func(s, sep string) []string { return strings.Split(s, sep) }})
	router.LoadHTMLGlob(templateGlob)
	router.GET(articleIDRoute, templateArticleHandler(dbConn))

	// Create test request with invalid ID
	req := httptest.NewRequest("GET", "/article/invalid", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTemplateArticleHandlerNotFound(t *testing.T) {
	templateGlob := findTemplateGlob()
	if templateGlob == "" {
		t.Skip(skipTemplateMessage)
	}

	gin.SetMode(gin.TestMode)
	dbConn, mock := setupTestDB(t)
	defer dbConn.Close()

	// Mock article fetch returning no rows
	mock.ExpectQuery("SELECT \\* FROM articles WHERE id = \\?").
		WithArgs(int64(999)).
		WillReturnError(fmt.Errorf("sql: no rows in result set"))

	// Setup router and handler
	router := gin.New()
	router.SetFuncMap(template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"mul":   func(a, b float64) float64 { return a * b },
		"split": func(s, sep string) []string { return strings.Split(s, sep) }})
	router.LoadHTMLGlob(templateGlob)
	router.GET(articleIDRoute, templateArticleHandler(dbConn))

	// Create test request
	req := httptest.NewRequest("GET", "/article/999", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
