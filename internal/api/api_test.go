package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

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
)

// DBOperations defines the interface for database operations
type DBOperations interface {
	ArticleExistsByURL(*sqlx.DB, string) (bool, error)
	InsertArticle(*sqlx.DB, *db.Article) (int64, error)
	FetchArticles(*sqlx.DB, string, string, int, int) ([]db.Article, error)
	FetchArticleByID(*sqlx.DB, int64) (*db.Article, error)
	FetchLLMScores(*sqlx.DB, int64) ([]db.LLMScore, error)
	FetchLatestEnsembleScore(*sqlx.DB, int64) (float64, error)
	FetchLatestConfidence(*sqlx.DB, int64) (float64, error)
	UpdateArticleScore(*sqlx.DB, int64, float64, int) error
	InsertFeedback(*sqlx.DB, *db.Feedback) error
}

// mockDB implements DBOperations for testing
type mockDB struct {
	ArticleExistsByURLFunc       func(*sqlx.DB, string) (bool, error)
	InsertArticleFunc            func(*sqlx.DB, *db.Article) (int64, error)
	FetchArticlesFunc            func(*sqlx.DB, string, string, int, int) ([]db.Article, error)
	FetchArticleByIDFunc         func(*sqlx.DB, int64) (*db.Article, error)
	FetchLLMScoresFunc           func(*sqlx.DB, int64) ([]db.LLMScore, error)
	FetchLatestEnsembleScoreFunc func(*sqlx.DB, int64) (float64, error)
	FetchLatestConfidenceFunc    func(*sqlx.DB, int64) (float64, error)
	UpdateArticleScoreFunc       func(*sqlx.DB, int64, float64, int) error
	InsertFeedbackFunc           func(*sqlx.DB, *db.Feedback) error
}

// Mock implementations
func (m *mockDB) ArticleExistsByURL(db *sqlx.DB, url string) (bool, error) {
	if m.ArticleExistsByURLFunc == nil {
		return false, nil
	}
	return m.ArticleExistsByURLFunc(db, url)
}

func (m *mockDB) InsertArticle(db *sqlx.DB, a *db.Article) (int64, error) {
	if m.InsertArticleFunc == nil {
		return 0, nil
	}
	return m.InsertArticleFunc(db, a)
}

func (m *mockDB) FetchArticles(db *sqlx.DB, source, leaning string, limit, offset int) ([]db.Article, error) {
	if m.FetchArticlesFunc == nil {
		return nil, nil
	}
	return m.FetchArticlesFunc(db, source, leaning, limit, offset)
}

func (m *mockDB) FetchArticleByID(db *sqlx.DB, id int64) (*db.Article, error) {
	if m.FetchArticleByIDFunc == nil {
		return nil, nil
	}
	return m.FetchArticleByIDFunc(db, id)
}

func (m *mockDB) FetchLLMScores(db *sqlx.DB, id int64) ([]db.LLMScore, error) {
	if m.FetchLLMScoresFunc == nil {
		return nil, nil
	}
	return m.FetchLLMScoresFunc(db, id)
}

func (m *mockDB) FetchLatestEnsembleScore(db *sqlx.DB, id int64) (float64, error) {
	if m.FetchLatestEnsembleScoreFunc == nil {
		return 0, nil
	}
	return m.FetchLatestEnsembleScoreFunc(db, id)
}

func (m *mockDB) FetchLatestConfidence(db *sqlx.DB, id int64) (float64, error) {
	if m.FetchLatestConfidenceFunc == nil {
		return 0, nil
	}
	return m.FetchLatestConfidenceFunc(db, id)
}

func (m *mockDB) UpdateArticleScore(db *sqlx.DB, id int64, score float64, conf int) error {
	if m.UpdateArticleScoreFunc == nil {
		return nil
	}
	return m.UpdateArticleScoreFunc(db, id, score, conf)
}

func (m *mockDB) InsertFeedback(db *sqlx.DB, f *db.Feedback) error {
	if m.InsertFeedbackFunc == nil {
		return nil
	}
	return m.InsertFeedbackFunc(db, f)
}

// Test helper functions
func setupTestRouter(mock DBOperations) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set up routes with the mock DB
	router.POST(articlesEndpoint, createArticleHandlerWithDB(mock))
	router.GET(articlesEndpoint, getArticlesHandlerWithDB(mock))
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
func createArticleHandlerWithDB(dbOps DBOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Source  string `json:"source"`
			PubDate string `json:"pub_date"`
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
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
			ScoreSource:    "llm",
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

func getArticlesHandlerWithDB(dbOps DBOperations) gin.HandlerFunc {
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

func manualScoreHandlerWithDB(dbOps DBOperations) gin.HandlerFunc {
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

func feedbackHandlerWithDB(dbOps DBOperations) gin.HandlerFunc {
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

func biasHandlerWithDB(dbOps DBOperations) gin.HandlerFunc {
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

func summaryHandlerWithDB(dbOps DBOperations) gin.HandlerFunc {
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

func ensembleDetailsHandlerWithDB(dbOps DBOperations) gin.HandlerFunc {
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
	mock := &mockDB{
		ArticleExistsByURLFunc: func(_ *sqlx.DB, url string) (bool, error) { return false, nil },
		InsertArticleFunc:      func(_ *sqlx.DB, a *db.Article) (int64, error) { return 1, nil },
	}

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
	mock := &mockDB{
		ArticleExistsByURLFunc: func(_ *sqlx.DB, url string) (bool, error) { return true, nil },
	}
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
	mock := &mockDB{
		FetchArticlesFunc: func(_ *sqlx.DB, source, leaning string, limit, offset int) ([]db.Article, error) {
			return []db.Article{}, nil
		},
	}
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
	mock := &mockDB{
		FetchArticleByIDFunc: func(_ *sqlx.DB, id int64) (*db.Article, error) {
			return &db.Article{ID: id}, nil
		},
		UpdateArticleScoreFunc: func(_ *sqlx.DB, id int64, score float64, conf int) error {
			return nil
		},
	}
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

func TestFeedbackValidation(t *testing.T) {
	mock := &mockDB{
		InsertFeedbackFunc: func(_ *sqlx.DB, f *db.Feedback) error { return nil },
		FetchLLMScoresFunc: func(_ *sqlx.DB, id int64) ([]db.LLMScore, error) { return nil, nil },
	}
	router := setupTestRouter(mock)

	// Missing fields
	body := `{"user_id":"u"}`
	req, _ := http.NewRequest("POST", feedbackEndpoint, bytes.NewBuffer([]byte(body)))
	req.Header.Set(contentTypeKey, contentTypeJSON)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	mock := &mockDB{
		FetchLLMScoresFunc: func(_ *sqlx.DB, id int64) ([]db.LLMScore, error) {
			return []db.LLMScore{
				{Model: "gpt", Score: 0.5, Metadata: "{}", CreatedAt: time.Now()},
			}, nil
		},
	}
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
	mock := &mockDB{
		FetchLLMScoresFunc: func(_ *sqlx.DB, id int64) ([]db.LLMScore, error) {
			return []db.LLMScore{}, nil
		},
	}
	router := setupTestRouter(mock)

	req, _ := http.NewRequest("GET", strings.Replace(ensembleEndpoint, ":id", "1", 1), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
