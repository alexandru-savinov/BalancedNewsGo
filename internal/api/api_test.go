package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Import mock.Anything directly to make it accessible in test methods
var Anything = mock.Anything

// Constants for commonly used values
const (
	articlesEndpoint    = "/api/articles"
	contentTypeJSON     = "application/json"
	contentTypeKey      = "Content-Type"
	biasEndpoint        = "/api/articles/:id/bias"
	manualScoreEndpoint = "/api/manual-score/:id"
	feedbackEndpoint    = "/api/feedback"
	summaryEndpoint     = "/api/articles/:id/summary"
	ensembleEndpoint    = "/api/articles/:id/ensemble"
	feedsHealthEndpoint = "/api/feeds/healthz"
	reanalyzeEndpoint   = "/api/llm/reanalyze/:id"
)

// MockLLMClient is a mock implementation of the llm.Client interface
type MockLLMClient struct {
	mock.Mock
}

// AnalyzeArticle mocks the llm.Client.AnalyzeArticle method
func (m *MockLLMClient) AnalyzeArticle(ctx context.Context, article *db.Article) (*llm.ArticleAnalysis, error) {
	args := m.Called(ctx, article)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llm.ArticleAnalysis), args.Error(1)
}

// MockDBOperations is a mock implementation of the DBOperations interface
type MockDBOperations struct {
	mock.Mock
}

// GetArticleByID mocks the DBOperations.GetArticleByID method
func (m *MockDBOperations) GetArticleByID(ctx context.Context, id int64) (*db.Article, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Article), args.Error(1)
}

// FetchArticleByID mocks the DBOperations.FetchArticleByID method
func (m *MockDBOperations) FetchArticleByID(ctx context.Context, id int64) (*db.Article, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Article), args.Error(1)
}

// ArticleExistsByURL mocks the DBOperations.ArticleExistsByURL method
func (m *MockDBOperations) ArticleExistsByURL(ctx context.Context, url string) (bool, error) {
	args := m.Called(ctx, url)
	return args.Bool(0), args.Error(1)
}

// GetArticles mocks the DBOperations.GetArticles method
func (m *MockDBOperations) GetArticles(ctx context.Context, filter db.ArticleFilter) ([]*db.Article, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*db.Article), args.Error(1)
}

// FetchArticles mocks the DBOperations.FetchArticles method
func (m *MockDBOperations) FetchArticles(ctx context.Context, source, leaning string, limit, offset int) ([]*db.Article, error) {
	args := m.Called(ctx, source, leaning, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	
	// Handle both []db.Article and []*db.Article
	switch result := args.Get(0).(type) {
	case []db.Article:
		// Convert []db.Article to []*db.Article
		articles := make([]*db.Article, len(result))
		for i := range result {
			articles[i] = &result[i]
		}
		return articles, args.Error(1)
	case []*db.Article:
		return result, args.Error(1)
	default:
		return nil, fmt.Errorf("unexpected type for FetchArticles result: %T", args.Get(0))
	}
}

// InsertArticle mocks the DBOperations.InsertArticle method
func (m *MockDBOperations) InsertArticle(ctx context.Context, article *db.Article) (int64, error) {
	args := m.Called(ctx, article)
	return args.Get(0).(int64), args.Error(1)
}

// UpdateArticleScore mocks the DBOperations.UpdateArticleScore method
func (m *MockDBOperations) UpdateArticleScore(ctx context.Context, articleID int64, score float64, confidence float64) error {
	args := m.Called(ctx, articleID, score, confidence)
	return args.Error(0)
}

// UpdateArticleScoreObj mocks the DBOperations.UpdateArticleScoreObj method
func (m *MockDBOperations) UpdateArticleScoreObj(ctx context.Context, articleID int64, score *db.ArticleScore, confidence float64) error {
	args := m.Called(ctx, articleID, score, confidence)
	return args.Error(0)
}

// SaveArticleFeedback mocks the DBOperations.SaveArticleFeedback method
func (m *MockDBOperations) SaveArticleFeedback(ctx context.Context, feedback *db.ArticleFeedback) error {
	args := m.Called(ctx, feedback)
	return args.Error(0)
}

// InsertFeedback mocks the DBOperations.InsertFeedback method 
func (m *MockDBOperations) InsertFeedback(ctx context.Context, feedback *db.Feedback) error {
	args := m.Called(ctx, feedback)
	return args.Error(0)
}

// FetchLLMScores mocks the DBOperations.FetchLLMScores method
func (m *MockDBOperations) FetchLLMScores(ctx context.Context, articleID int64) ([]db.LLMScore, error) {
	args := m.Called(ctx, articleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.LLMScore), args.Error(1)
}

// Test helper functions
func setupTestRouter(mock db.DBOperations) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set up routes with the mock DB
	router.POST(articlesEndpoint, createArticleHandlerWithDB(mock))
	router.GET(articlesEndpoint, getArticlesHandlerWithDB(mock))
	// Add article detail route
	router.GET(articlesEndpoint+"/:id", getArticleByIDHandlerWithDB(mock))
	router.POST(manualScoreEndpoint, manualScoreHandlerWithDB(mock))
	router.POST(feedbackEndpoint, feedbackHandlerWithDB(mock))
	router.GET(biasEndpoint, biasHandlerWithDB(mock))
	router.GET(summaryEndpoint, summaryHandlerWithDB(mock))
	router.GET(ensembleEndpoint, ensembleDetailsHandlerWithDB(mock))
	router.GET(feedsHealthEndpoint, func(c *gin.Context) {
		RespondSuccess(c, map[string]interface{}{"status": "ok"})
	})

	return router
}

// Handler wrappers that use the DB interface
func createArticleHandlerWithDB(dbOps db.DBOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Source  string `json:"source"`
			PubDate string `json:"pub_date"`
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			RespondError(c, ErrInvalidPayload)
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
			RespondError(c, NewAppError(ErrValidation,
				fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", "))))
			return
		}

		// Validate URL format
		if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
			RespondError(c, NewAppError(ErrValidation, "Invalid URL format (must start with http:// or https://)"))
			return
		}

		// Check if article already exists
		exists, err := dbOps.ArticleExistsByURL(nil, req.URL)
		if err != nil {
			RespondError(c, WrapError(err, ErrInternal, "Failed to check for existing article"))
			return
		}
		if exists {
			RespondError(c, ErrDuplicateURL)
			return
		}

		// Parse pub_date
		pubDate, err := time.Parse(time.RFC3339, req.PubDate)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid pub_date format (expected RFC3339)"))
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
			ScoreSource:    strPtr("llm"),
		}

		id, err := dbOps.InsertArticle(nil, article)
		if err != nil {
			RespondError(c, WrapError(err, ErrInternal, "Failed to create article"))
			return
		}

		RespondSuccess(c, map[string]interface{}{
			"status":     "created",
			"article_id": id,
		})
	}
}

func getArticlesHandlerWithDB(dbOps db.DBOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		source := c.Query("source")
		leaning := c.Query("leaning")
		limitStr := c.DefaultQuery("limit", "20")
		offsetStr := c.DefaultQuery("offset", "0")

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			RespondError(c, NewAppError(ErrValidation, "Invalid 'limit' parameter"))
			return
		}
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			RespondError(c, NewAppError(ErrValidation, "Invalid 'offset' parameter"))
			return
		}

		articles, err := dbOps.FetchArticles(nil, source, leaning, limit, offset)
		if err != nil {
			RespondError(c, WrapError(err, ErrInternal, "Failed to fetch articles"))
			return
		}

		RespondSuccess(c, articles)
	}
}

func manualScoreHandlerWithDB(dbOps db.DBOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			RespondError(c, NewAppError(ErrValidation, "Invalid article ID"))
			return
		}
		articleID := int64(id)

		// Read raw body for strict validation
		var raw map[string]interface{}
		if err := c.ShouldBindJSON(&raw); err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid JSON body"))
			return
		}
		// Only "score" is allowed
		if len(raw) != 1 || raw["score"] == nil {
			RespondError(c, NewAppError(ErrValidation, "Payload must contain only 'score' field"))
			return
		}
		// Validate score type and range
		scoreVal, ok := raw["score"].(float64)
		if !ok {
			// Accept integer as well
			if intVal, okInt := raw["score"].(int); okInt {
				scoreVal = float64(intVal)
			} else {
				RespondError(c, NewAppError(ErrValidation, "'score' must be a number"))
				return
			}
		}
		if scoreVal < -1.0 || scoreVal > 1.0 {
			RespondError(c, NewAppError(ErrValidation, "Score must be between -1.0 and 1.0"))
			return
		}

		// Check if article exists
		_, err = dbOps.FetchArticleByID(nil, articleID)
		if err != nil {
			RespondError(c, NewAppError(ErrNotFound, "Article not found"))
			return
		}

		// Update score in DB
		err = dbOps.UpdateArticleScore(nil, articleID, scoreVal, 0)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to update article score"))
			return
		}

		RespondSuccess(c, map[string]interface{}{
			"status":     "score updated",
			"article_id": articleID,
			"score":      scoreVal,
		})
	}
}

func feedbackHandlerWithDB(dbOps db.DBOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ArticleID        int64  `json:"article_id" form:"article_id"`
			UserID           string `json:"user_id" form:"user_id"`
			FeedbackText     string `json:"feedback_text" form:"feedback_text"`
			Category         string `json:"category" form:"category"`
			EnsembleOutputID *int64 `json:"ensemble_output_id" form:"ensemble_output_id"`
			Source           string `json:"source" form:"source"`
		}

		if err := c.ShouldBind(&req); err != nil {
			RespondError(c, ErrInvalidPayload)
			return
		}

		// Validate all required fields
		var missingFields []string
		if req.ArticleID == 0 {
			missingFields = append(missingFields, "article_id")
		}
		if req.FeedbackText == "" {
			missingFields = append(missingFields, "feedback_text")
		}
		if req.UserID == "" {
			missingFields = append(missingFields, "user_id")
		}

		if len(missingFields) > 0 {
			RespondError(c, NewAppError(ErrValidation,
				fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", "))))
			return
		}

		// Validate Category if provided
		validCategories := map[string]bool{
			"agree":    true,
			"disagree": true,
			"unclear":  true,
			"other":    true,
			"":         true, // Allow empty for backward compatibility
		}

		if req.Category != "" && !validCategories[req.Category] {
			RespondError(c, NewAppError(ErrValidation, "Invalid category, allowed values: agree, disagree, unclear, other"))
			return
		}

		feedback := &db.Feedback{
			ArticleID:        req.ArticleID,
			UserID:           req.UserID,
			FeedbackText:     req.FeedbackText,
			Category:         req.Category,
			EnsembleOutputID: req.EnsembleOutputID,
			Source:           req.Source,
			CreatedAt:        time.Now(),
		}

		err := dbOps.InsertFeedback(nil, feedback)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to store feedback"))
			return
		}

		RespondSuccess(c, map[string]string{"status": "feedback received"})
	}
}

func biasHandlerWithDB(dbOps db.DBOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		articleID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || articleID < 1 {
			RespondError(c, NewAppError(ErrValidation, "Invalid article ID"))
			return
		}

		minScore, err := strconv.ParseFloat(c.DefaultQuery("min_score", "-2"), 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid min_score"))
			return
		}
		maxScore, err := strconv.ParseFloat(c.DefaultQuery("max_score", "2"), 64)
		if err != nil {
			RespondError(c, NewAppError(ErrValidation, "Invalid max_score"))
			return
		}

		scores, err := dbOps.FetchLLMScores(nil, articleID)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch bias data"))
			return
		}

		var latestEnsembleScore *db.LLMScore
		individualResults := make([]map[string]interface{}, 0)

		for _, score := range scores {
			if score.Model == "ensemble" {
				if latestEnsembleScore == nil || score.CreatedAt.After(latestEnsembleScore.CreatedAt) {
					latestEnsembleScore = &score
				}
			} else if score.Score >= minScore && score.Score <= maxScore {
				individualResults = append(individualResults, map[string]interface{}{
					"model":      score.Model,
					"score":      score.Score,
					"created_at": score.CreatedAt,
				})
			}
		}

		var compositeScoreValue interface{} = nil
		status := ""
		if latestEnsembleScore != nil {
			compositeScoreValue = latestEnsembleScore.Score
		} else {
			status = "scoring_unavailable"
		}

		resp := map[string]interface{}{
			"composite_score": compositeScoreValue,
			"results":         individualResults,
		}
		if status != "" {
			resp["status"] = status
		}

		RespondSuccess(c, resp)
	}
}

func summaryHandlerWithDB(dbOps db.DBOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id < 1 {
			RespondError(c, ErrInvalidArticleID)
			return
		}

		// Verify article exists
		_, err = dbOps.FetchArticleByID(nil, id)
		if err != nil {
			RespondError(c, ErrArticleNotFound)
			return
		}

		scores, err := dbOps.FetchLLMScores(nil, id)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch article summary"))
			return
		}

		for _, score := range scores {
			if score.Model == "summarizer" {
				result := map[string]interface{}{
					"summary":    score.Metadata,
					"created_at": score.CreatedAt,
				}
				RespondSuccess(c, result)
				return
			}
		}

		RespondError(c, NewAppError(ErrNotFound, "Article summary not available"))
	}
}

func ensembleDetailsHandlerWithDB(dbOps db.DBOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id < 1 {
			RespondError(c, NewAppError(ErrValidation, "Invalid article ID"))
			return
		}

		scores, err := dbOps.FetchLLMScores(nil, id)
		if err != nil {
			RespondError(c, NewAppError(ErrInternal, "Failed to fetch ensemble data"))
			return
		}

		details := make([]map[string]interface{}, 0)
		for _, score := range scores {
			if score.Model == "ensemble" {
				details = append(details, map[string]interface{}{
					"score":      score.Score,
					"metadata":   score.Metadata,
					"created_at": score.CreatedAt,
				})
			}
		}

		if len(details) == 0 {
			RespondError(c, NewAppError(ErrNotFound, "Ensemble data not found"))
			return
		}

		RespondSuccess(c, gin.H{"scores": details})
	}
}

// --- Tests ---
func TestCreateArticleValidation(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("ArticleExistsByURL", Anything, Anything).Return(false, nil)
	mock.On("InsertArticle", Anything, Anything).Return(int64(1), nil)

	router := setupTestRouter(mock)
	var w *httptest.ResponseRecorder

	// Missing fields
	body := `{"source": "src"}`
	req, _ := http.NewRequest("POST", articlesEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Missing required fields")

	// Invalid URL
	body = `{"source":"src","pub_date":"2022-01-01T00:00:00Z","url":"ftp://bad","title":"t","content":"c"}`
	req, _ = http.NewRequest("POST", articlesEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid URL format")

	// Invalid pub_date
	body = `{"source":"src","pub_date":"bad","url":"http://good","title":"t","content":"c"}`
	req, _ = http.NewRequest("POST", articlesEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid pub_date format")
}

func TestCreateArticleDuplicate(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("ArticleExistsByURL", Anything, Anything).Return(true, nil)
	router := setupTestRouter(mock)

	body := `{"source":"src","pub_date":"2022-01-01T00:00:00Z","url":"http://good","title":"t","content":"c"}`
	req, _ := http.NewRequest("POST", articlesEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already exists")
}

func TestGetArticlesLimitOffsetValidation(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("FetchArticles", Anything, Anything, Anything, Anything, Anything).Return([]db.Article{}, nil)
	router := setupTestRouter(mock)

	// Invalid limit
	req, _ := http.NewRequest("GET", articlesEndpoint+"?limit=bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Invalid offset
	req, _ = http.NewRequest("GET", articlesEndpoint+"?offset=-1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestManualScoreValidation(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("FetchArticleByID", Anything, Anything).Return(&db.Article{}, nil)
	mock.On("UpdateArticleScore", Anything, Anything, Anything, Anything).Return(nil)
	router := setupTestRouter(mock)

	// Invalid article ID
	req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpoint, ":id", "bad", 1), bytes.NewBuffer([]byte(`{"score":0}`)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Not a float or int
	body := `{"score":"bad"}`
	req, _ = http.NewRequest("POST", strings.Replace(manualScoreEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Out of range
	body = `{"score":2.0}`
	req, _ = http.NewRequest("POST", strings.Replace(manualScoreEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestManualScoreDatabaseErrors(t *testing.T) {
	// Article not found case
	mock1 := &MockDBOperations{}
	mock1.On("FetchArticleByID", Anything, Anything).Return(nil, db.ErrArticleNotFound)

	router := setupTestRouter(mock1)
	body := `{"score":0.5}`
	req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// DB error when updating score
	mock2 := &MockDBOperations{}
	mock2.On("FetchArticleByID", Anything, Anything).Return(&db.Article{}, nil)
	mock2.On("UpdateArticleScore", Anything, Anything, Anything, Anything).Return(fmt.Errorf("database error"))

	router = setupTestRouter(mock2)
	req, _ = http.NewRequest("POST", strings.Replace(manualScoreEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFeedbackValidation(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("InsertFeedback", Anything, Anything).Return(nil)
	mock.On("FetchLLMScores", Anything, Anything).Return(nil, nil)
	router := setupTestRouter(mock)

	// Missing fields
	body := `{"user_id":"u"}`
	req, _ := http.NewRequest("POST", feedbackEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeedbackValidationWithInvalidCategory(t *testing.T) {
	// Create a custom router with our modified handler
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register a custom feedback handler that validates categories
	router.POST(feedbackEndpoint, func(c *gin.Context) {
		var req struct {
			ArticleID        int64  `json:"article_id" form:"article_id"`
			UserID           string `json:"user_id" form:"user_id"`
			FeedbackText     string `json:"feedback_text" form:"feedback_text"`
			Category         string `json:"category" form:"category"`
			EnsembleOutputID *int64 `json:"ensemble_output_id" form:"ensemble_output_id"`
			Source           string `json:"source" form:"source"`
		}

		if err := c.ShouldBind(&req); err != nil {
			RespondError(c, ErrInvalidPayload)
			return
		}

		// Validate all required fields
		var missingFields []string
		if req.ArticleID == 0 {
			missingFields = append(missingFields, "article_id")
		}
		if req.FeedbackText == "" {
			missingFields = append(missingFields, "feedback_text")
		}
		if req.UserID == "" {
			missingFields = append(missingFields, "user_id")
		}

		if len(missingFields) > 0 {
			RespondError(c, NewAppError(ErrValidation,
				fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", "))))
			return
		}

		// Validate Category if provided
		validCategories := map[string]bool{
			"agree":    true,
			"disagree": true,
			"unclear":  true,
			"other":    true,
			"":         true, // Allow empty for backward compatibility
		}

		if req.Category != "" && !validCategories[req.Category] {
			RespondError(c, NewAppError(ErrValidation, "Invalid category, allowed values: agree, disagree, unclear, other"))
			return
		}

		// Skip the rest of the execution since we're only testing validation
		RespondSuccess(c, map[string]string{"status": "feedback received"})
	})

	// Test with invalid category
	body := `{"article_id":1,"user_id":"testuser","feedback_text":"test feedback","category":"invalid"}`
	req, _ := http.NewRequest("POST", feedbackEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid category")

	// Test with valid category
	body = `{"article_id":1,"user_id":"testuser","feedback_text":"test feedback","category":"agree"}`
	req, _ = http.NewRequest("POST", feedbackEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBiasInvalidId(t *testing.T) {
	router := setupTestRouter(nil)
	req, _ := http.NewRequest("GET", biasEndpoint, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBiasInvalidScoreParams(t *testing.T) {
	router := setupTestRouter(nil)
	req, _ := http.NewRequest("GET", biasEndpoint+"?min_score=bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBiasSuccessNoEnsemble(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("FetchLLMScores", Anything, Anything).Return([]db.LLMScore{
		{Model: "gpt", Score: 0.5, Metadata: "{}", CreatedAt: time.Now()},
	}, nil)
	router := setupTestRouter(mock)

	req, _ := http.NewRequest("GET", strings.Replace(biasEndpoint, ":id", "1", 1), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"composite_score":null`)
}

func TestFeedHealthHandler(t *testing.T) {
	router := setupTestRouter(nil)
	req, _ := http.NewRequest("GET", feedsHealthEndpoint, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSummaryInvalidId(t *testing.T) {
	router := setupTestRouter(nil)
	req, _ := http.NewRequest("GET", summaryEndpoint, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEnsembleDetailsInvalidId(t *testing.T) {
	router := setupTestRouter(nil)
	req, _ := http.NewRequest("GET", ensembleEndpoint, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEnsembleDetailsNotFound(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("FetchLLMScores", Anything, Anything).Return([]db.LLMScore{}, nil)
	router := setupTestRouter(mock)

	req, _ := http.NewRequest("GET", strings.Replace(ensembleEndpoint, ":id", "1", 1), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateArticleExtraFields(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("ArticleExistsByURL", Anything, Anything).Return(false, nil)
	mock.On("InsertArticle", Anything, Anything).Return(int64(1), nil)
	router := setupTestRouter(mock)
	// Extra field should be rejected
	body := `{"source":"src","pub_date":"2022-01-01T00:00:00Z","url":"http://good","title":"t","content":"c","extra":"field"}`
	req, _ := http.NewRequest("POST", articlesEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestManualScoreExtraFields(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("FetchArticleByID", Anything, Anything).Return(&db.Article{}, nil)
	mock.On("UpdateArticleScore", Anything, Anything, Anything, Anything).Return(nil)
	router := setupTestRouter(mock)
	// Extra field should be rejected
	body := `{"score":0.5,"extra":"field"}`
	req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateArticleMalformedJSON(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("ArticleExistsByURL", Anything, Anything).Return(false, nil)
	mock.On("InsertArticle", Anything, Anything).Return(int64(1), nil)
	router := setupTestRouter(mock)
	body := `{"source":"src","pub_date":"2022-01-01T00:00:00Z","url":"http://good","title":"t","content":"c",` // Malformed JSON
	req, _ := http.NewRequest("POST", articlesEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestManualScoreMalformedJSON(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("FetchArticleByID", Anything, Anything).Return(&db.Article{}, nil)
	mock.On("UpdateArticleScore", Anything, Anything, Anything, Anything).Return(nil)
	router := setupTestRouter(mock)
	body := `{"score":0.5,` // Malformed JSON
	req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateArticleWithStrictFieldValidation(t *testing.T) {
	// Create a custom router with a handler that checks for unknown fields
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register our custom handler directly
	router.POST(articlesEndpoint, func(c *gin.Context) {
		var req struct {
			Source  string `json:"source"`
			PubDate string `json:"pub_date"`
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			if strings.Contains(err.Error(), "unknown field") {
				RespondError(c, NewAppError(ErrValidation, "Request contains unknown or extra fields"))
				return
			}
			RespondError(c, ErrInvalidPayload)
			return
		}

		// For testing purposes, we just need to validate the fields
		RespondSuccess(c, map[string]interface{}{
			"status":     "created",
			"article_id": 1,
		})
	})

	// Test with unknown field - should be rejected with specific error
	body := `{"source":"src","pub_date":"2022-01-01T00:00:00Z","url":"http://good","title":"t","content":"c","unknown_field":"value"}`
	req, _ := http.NewRequest("POST", articlesEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "unknown or extra fields")
}

func TestFeedbackDatabaseError(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("InsertFeedback", Anything, Anything).Return(fmt.Errorf("database error"))
	router := setupTestRouter(mock)

	// Test DB error when inserting feedback
	body := `{"article_id":1,"user_id":"testuser","feedback_text":"test feedback","category":"agree"}`
	req, _ := http.NewRequest("POST", feedbackEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to store feedback")
}

func TestReanalyzeHandlerValidation(t *testing.T) {
	// Mock setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register a simplified reanalyze handler with only validation logic
	router.POST(reanalyzeEndpoint, func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": map[string]string{
					"code":    "validation_error",
					"message": "Invalid article ID",
				},
			})
			return
		}

		// Parse raw JSON body
		var raw map[string]interface{}
		if err := c.ShouldBindJSON(&raw); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": map[string]string{
					"code":    "validation_error",
					"message": "Invalid JSON payload",
				},
			})
			return
		}

		// Direct score update path - check if "score" field exists
		if scoreRaw, hasScore := raw["score"]; hasScore {
			var scoreFloat float64
			switch s := scoreRaw.(type) {
			case float64:
				scoreFloat = s
			case float32:
				scoreFloat = float64(s)
			case int:
				scoreFloat = float64(s)
			case int64:
				scoreFloat = float64(s)
			case string:
				var parseErr error
				scoreFloat, parseErr = strconv.ParseFloat(s, 64)
				if parseErr != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"success": false,
						"error": map[string]string{
							"code":    "validation_error",
							"message": "Invalid score value",
						},
					})
					return
				}
			default:
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error": map[string]string{
						"code":    "validation_error",
						"message": "Invalid score value",
					},
				})
				return
			}

			if scoreFloat < -1.0 || scoreFloat > 1.0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error": map[string]string{
						"code":    "validation_error",
						"message": "Score must be between -1.0 and 1.0",
					},
				})
				return
			}

			// Success case
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": map[string]interface{}{
					"status":     "score updated",
					"article_id": id,
					"score":      scoreFloat,
				},
			})
			return
		}

		// Success case for regular rescore
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": map[string]interface{}{
				"status":     "reanalyze queued",
				"article_id": id,
			},
		})
	})

	// Test with invalid article ID
	req, _ := http.NewRequest("POST", strings.Replace(reanalyzeEndpoint, ":id", "bad", 1), bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid article ID")

	// Test with malformed JSON
	req, _ = http.NewRequest("POST", strings.Replace(reanalyzeEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(`{"score":0.5`)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test with invalid score type
	req, _ = http.NewRequest("POST", strings.Replace(reanalyzeEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(`{"score":"invalid"}`)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid score value")

	// Test with out-of-range score
	req, _ = http.NewRequest("POST", strings.Replace(reanalyzeEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(`{"score":2.0}`)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Score must be between -1.0 and 1.0")

	// Test with valid score
	req, _ = http.NewRequest("POST", strings.Replace(reanalyzeEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(`{"score":0.5}`)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "score updated")

	// Test without score (normal reanalyze path)
	req, _ = http.NewRequest("POST", strings.Replace(reanalyzeEndpoint, ":id", "1", 1), bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "reanalyze queued")
}

// Test summaryHandlerWithDB
func TestSummaryHandlerWithDBSuccess(t *testing.T) {
	// Setup mock DB to return existing article and a summarizer score
	mock := &MockDBOperations{}
	mock.On("FetchArticleByID", Anything, Anything).Return(&db.Article{}, nil)
	mock.On("FetchLLMScores", Anything, Anything).Return([]db.LLMScore{{Model: "summarizer", Metadata: "my summary", CreatedAt: time.Now()}}, nil)
	r := setupTestRouter(mock)
	req, _ := http.NewRequest("GET", strings.Replace(summaryEndpoint, ":id", "42", 1), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "my summary")
}

func TestSummaryHandlerWithDBNotFound(t *testing.T) {
	// Article exists but no summarizer score
	mock := &MockDBOperations{}
	mock.On("FetchArticleByID", Anything, Anything).Return(&db.Article{}, nil)
	mock.On("FetchLLMScores", Anything, Anything).Return([]db.LLMScore{}, nil)
	r := setupTestRouter(mock)
	req, _ := http.NewRequest("GET", strings.Replace(summaryEndpoint, ":id", "100", 1), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test ensembleDetailsHandlerWithDB success
func TestEnsembleDetailsHandlerWithDBSuccess(t *testing.T) {
	mock := &MockDBOperations{}
	mock.On("FetchLLMScores", Anything, Anything).Return([]db.LLMScore{{Model: "ensemble", Score: 0.8, Metadata: "{}", CreatedAt: time.Now()}}, nil)
	r := setupTestRouter(mock)
	req, _ := http.NewRequest("GET", strings.Replace(ensembleEndpoint, ":id", "7", 1), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "0.8")
}

// Test getArticlesHandlerWithDB success and error
func TestGetArticlesHandlerWithDB(t *testing.T) {
	// Success case
	mock := &MockDBOperations{}
	mock.On("FetchArticles", Anything, Anything, Anything, Anything, Anything).Return([]db.Article{{ID: 1, Title: "x"}}, nil)
	r := setupTestRouter(mock)
	req, _ := http.NewRequest("GET", articlesEndpoint+"?source=a&leaning=right&limit=1&offset=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Len(t, body["data"].([]interface{}), 1)

	// Error case invalid limit
	req, _ = http.NewRequest("GET", articlesEndpoint+"?limit=0", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test getArticleByIDHandlerWithDB success and error
func TestGetArticleByIDHandlerWithDB(t *testing.T) {
	// Success case
	mockSuccess := &MockDBOperations{}
	mockSuccess.On("FetchArticleByID", Anything, Anything).Return(&db.Article{ID: 5, Title: "t"}, nil)
	mockSuccess.On("FetchLLMScores", Anything, Anything).Return([]db.LLMScore{{Model: "x", Score: 0.2, Metadata: "{}", CreatedAt: time.Now()}}, nil)
	mockSuccess.On("FetchLatestEnsembleScore", Anything, Anything).Return(0.2, nil)
	mockSuccess.On("FetchLatestConfidence", Anything, Anything).Return(0.3, nil)
	rSuccess := setupTestRouter(mockSuccess)
	req, _ := http.NewRequest("GET", strings.Replace(articlesEndpoint+"/:id", ":id", "5", 1), nil)
	w := httptest.NewRecorder()
	rSuccess.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.True(t, body["success"].(bool))

	// Not found case
	mockNotFound := &MockDBOperations{}
	mockNotFound.On("FetchArticleByID", Anything, Anything).Return(nil, db.ErrArticleNotFound)
	rNotFound := setupTestRouter(mockNotFound)
	req, _ = http.NewRequest("GET", strings.Replace(articlesEndpoint+"/:id", ":id", "99", 1), nil)
	w = httptest.NewRecorder()
	rNotFound.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// DB error case
	mockError := &MockDBOperations{}
	mockError.On("FetchArticleByID", Anything, Anything).Return(nil, fmt.Errorf("db error"))
	rError := setupTestRouter(mockError)
	req, _ = http.NewRequest("GET", strings.Replace(articlesEndpoint+"/:id", ":id", "7", 1), nil)
	w = httptest.NewRecorder()
	rError.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
