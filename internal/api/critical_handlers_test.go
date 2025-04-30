package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Critical tests for API handlers with low coverage:
// - createArticleHandler
// - getArticlesHandler
// - getArticleByIDHandler
// - biasHandler
// - summaryHandler

// ======= Article Creation Tests =======

// TestCreateArticleHandlerSuccess tests the successful creation of an article
func TestCreateArticleHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock DB
	mockDB := new(MockDBOperations)
	mockDB.On("ArticleExistsByURL", mock.Anything, mock.Anything).Return(false, nil)
	mockDB.On("InsertArticle", mock.Anything, mock.AnythingOfType("*db.Article")).Return(int64(1), nil)
	mockDB.On("FetchArticleByID", mock.Anything, int64(1)).Return(&db.Article{
		ID:     1,
		Source: "test-source",
		URL:    "https://example.com/article",
		Title:  "Test Article",
	}, nil)

	// Create router and handler
	router := gin.New()
	router.POST("/api/articles", SafeHandler(createArticleHandler(&sqlx.DB{})))

	// Create valid request
	validArticle := `{
		"source": "test-source",
		"pub_date": "2025-04-30T12:00:00Z",
		"url": "https://example.com/article",
		"title": "Test Article",
		"content": "This is a test article."
	}`
	req, _ := http.NewRequest("POST", "/api/articles", bytes.NewBuffer([]byte(validArticle)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["article_id"])
}

// TestCreateArticleHandlerValidationErrors tests article creation validation errors
func TestCreateArticleHandlerValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test cases
	tests := []struct {
		name        string
		requestBody string
		expectedMsg string
	}{
		{
			name: "Missing Fields",
			requestBody: `{
				"source": "test-source"
			}`,
			expectedMsg: "Missing required fields",
		},
		{
			name: "Invalid URL",
			requestBody: `{
				"source": "test-source",
				"pub_date": "2025-04-30T12:00:00Z",
				"url": "invalid-url",
				"title": "Test Article",
				"content": "This is a test article."
			}`,
			expectedMsg: "Invalid URL format",
		},
		{
			name: "Invalid Date",
			requestBody: `{
				"source": "test-source",
				"pub_date": "2025-04-30",
				"url": "https://example.com/article",
				"title": "Test Article",
				"content": "This is a test article."
			}`,
			expectedMsg: "Invalid pub_date format",
		},
	}

	// Create router and handler
	router := gin.New()
	router.POST("/api/articles", SafeHandler(createArticleHandler(&sqlx.DB{})))

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/articles", bytes.NewBuffer([]byte(tc.requestBody)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.False(t, response["success"].(bool))
			assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), tc.expectedMsg)
		})
	}
}

// TestCreateArticleDuplicateURL tests article creation with a duplicate URL
func TestCreateArticleDuplicateURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock DB
	mockDB := new(MockDBOperations)
	mockDB.On("ArticleExistsByURL", mock.Anything, mock.Anything).Return(true, nil)

	// Create router and handler
	router := gin.New()
	router.POST("/api/articles", SafeHandler(createArticleHandler(&sqlx.DB{})))

	// Create request with duplicate URL
	duplicateArticle := `{
		"source": "test-source",
		"pub_date": "2025-04-30T12:00:00Z",
		"url": "https://example.com/article",
		"title": "Test Article",
		"content": "This is a test article."
	}`
	req, _ := http.NewRequest("POST", "/api/articles", bytes.NewBuffer([]byte(duplicateArticle)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ======= Get Articles Tests =======

// TestGetArticlesHandlerSuccess tests the successful retrieval of articles
func TestGetArticlesHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock DB
	mockDB := new(MockDBOperations)
	articles := []db.Article{
		{ID: 1, Title: "Article 1", Source: "source1"},
		{ID: 2, Title: "Article 2", Source: "source2"},
	}
	mockDB.On("FetchArticles", mock.Anything, "", "", 20, 0).Return(articles, nil)
	mockDB.On("FetchLLMScores", mock.Anything, int64(1)).Return([]db.LLMScore{}, nil)
	mockDB.On("FetchLLMScores", mock.Anything, int64(2)).Return([]db.LLMScore{}, nil)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

	// Create request
	req, _ := http.NewRequest("GET", "/api/articles", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data, ok := response["data"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(data))
}

// TestGetArticlesHandlerFilters tests article retrieval with filters
func TestGetArticlesHandlerFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock DB
	mockDB := new(MockDBOperations)
	articles := []db.Article{
		{ID: 3, Title: "CNN Article", Source: "cnn"},
	}
	mockDB.On("FetchArticles", mock.Anything, "cnn", "left", 5, 10).Return(articles, nil)
	mockDB.On("FetchLLMScores", mock.Anything, int64(3)).Return([]db.LLMScore{}, nil)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

	// Create request with filters
	req, _ := http.NewRequest("GET", "/api/articles?source=cnn&leaning=left&limit=5&offset=10", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data, ok := response["data"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 1, len(data))
}

// TestGetArticlesHandlerErrors tests error cases for article retrieval
func TestGetArticlesHandlerErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

	// Test invalid limit
	t.Run("Invalid Limit", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/articles?limit=invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test invalid offset
	t.Run("Invalid Offset", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/articles?offset=invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test negative offset
	t.Run("Negative Offset", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/articles?offset=-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test database error
	t.Run("Database Error", func(t *testing.T) {
		mockDB := new(MockDBOperations)
		mockDB.On("FetchArticles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]db.Article{}, errors.New("database error"))

		router := gin.New()
		router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

		req, _ := http.NewRequest("GET", "/api/articles", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ======= Get Article By ID Tests =======

// TestGetArticleByIDHandlerSuccess tests successful article retrieval by ID
func TestGetArticleByIDHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup test data
	score := 0.75
	confidence := 0.85
	article := &db.Article{
		ID:             1,
		Title:          "Test Article",
		Content:        "Test content",
		URL:            "https://example.com/article",
		Source:         "test-source",
		PubDate:        time.Now(),
		CreatedAt:      time.Now(),
		CompositeScore: &score,
		Confidence:     &confidence,
	}

	// Setup mock DB
	mockDB := new(MockDBOperations)
	mockDB.On("FetchArticleByID", mock.Anything, int64(1)).Return(article, nil)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

	// Create request
	req, _ := http.NewRequest("GET", "/api/articles/1", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["article_id"])
	assert.Equal(t, "Test Article", data["Title"])
	assert.Equal(t, score, data["CompositeScore"])
	assert.Equal(t, confidence, data["Confidence"])
}

// TestGetArticleByIDHandlerErrors tests error cases for article retrieval by ID
func TestGetArticleByIDHandlerErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

	// Test invalid ID
	t.Run("Invalid ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/articles/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test article not found
	t.Run("Article Not Found", func(t *testing.T) {
		mockDB := new(MockDBOperations)
		mockDB.On("FetchArticleByID", mock.Anything, int64(999)).Return(nil, db.ErrArticleNotFound)

		router := gin.New()
		router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

		req, _ := http.NewRequest("GET", "/api/articles/999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Test database error
	t.Run("Database Error", func(t *testing.T) {
		mockDB := new(MockDBOperations)
		mockDB.On("FetchArticleByID", mock.Anything, int64(2)).Return(nil, errors.New("database error"))

		router := gin.New()
		router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

		req, _ := http.NewRequest("GET", "/api/articles/2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ======= Bias Handler Tests =======

// TestBiasHandlerWithEnsemble tests the bias handler with ensemble scores
func TestBiasHandlerWithEnsemble(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock DB
	mockDB := new(MockDBOperations)
	now := time.Now()
	mockDB.On("FetchLLMScores", mock.Anything, int64(1)).Return([]db.LLMScore{
		{
			Model:     "ensemble",
			Score:     0.75,
			Metadata:  `{"aggregation":{"weighted_mean":0.75,"confidence":0.8}}`,
			CreatedAt: now,
		},
		{
			Model:     "gpt",
			Score:     0.7,
			Metadata:  `{"Confidence":0.8,"Explanation":"Liberal leaning"}`,
			CreatedAt: now,
		},
		{
			Model:     "claude",
			Score:     0.8,
			Metadata:  `{"Confidence":0.9,"Explanation":"Somewhat liberal"}`,
			CreatedAt: now,
		},
	}, nil)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles/:id/bias", SafeHandler(biasHandler(&sqlx.DB{})))

	// Create request
	req, _ := http.NewRequest("GET", "/api/articles/1/bias", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 0.75, data["composite_score"])
	assert.IsType(t, []interface{}{}, data["results"])
}

// TestBiasHandlerWithoutEnsemble tests the bias handler without ensemble scores
func TestBiasHandlerWithoutEnsemble(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock DB
	mockDB := new(MockDBOperations)
	now := time.Now()
	mockDB.On("FetchLLMScores", mock.Anything, int64(2)).Return([]db.LLMScore{
		{
			Model:     "gpt",
			Score:     0.7,
			Metadata:  `{"Confidence":0.8,"Explanation":"Liberal leaning"}`,
			CreatedAt: now,
		},
	}, nil)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles/:id/bias", SafeHandler(biasHandler(&sqlx.DB{})))

	// Create request
	req, _ := http.NewRequest("GET", "/api/articles/2/bias", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Nil(t, data["composite_score"])
	assert.Equal(t, "scoring_unavailable", data["status"])
}

// ======= Summary Handler Tests =======

// TestSummaryHandlerSuccess tests the successful retrieval of an article summary
func TestSummaryHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock DB
	mockDB := new(MockDBOperations)
	mockDB.On("FetchArticleByID", mock.Anything, int64(1)).Return(&db.Article{ID: 1}, nil)
	mockDB.On("FetchLLMScores", mock.Anything, int64(1)).Return([]db.LLMScore{
		{Model: "summarizer", Metadata: `{"summary": "This is a test summary"}`, CreatedAt: time.Now()},
	}, nil)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles/:id/summary", SafeHandler(summaryHandler(&sqlx.DB{})))

	// Create request
	req, _ := http.NewRequest("GET", "/api/articles/1/summary", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "This is a test summary", data["summary"])
}

// TestSummaryHandlerNotFound tests the summary handler when no summary is available
func TestSummaryHandlerNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock DB
	mockDB := new(MockDBOperations)
	mockDB.On("FetchArticleByID", mock.Anything, int64(2)).Return(&db.Article{ID: 2}, nil)
	mockDB.On("FetchLLMScores", mock.Anything, int64(2)).Return([]db.LLMScore{
		{Model: "gpt", Metadata: `{}`, CreatedAt: time.Now()},
	}, nil)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles/:id/summary", SafeHandler(summaryHandler(&sqlx.DB{})))

	// Create request
	req, _ := http.NewRequest("GET", "/api/articles/2/summary", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "summary not available")
}

// TestSummaryHandlerErrors tests error cases for the summary handler
func TestSummaryHandlerErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create router and handler
	router := gin.New()
	router.GET("/api/articles/:id/summary", SafeHandler(summaryHandler(&sqlx.DB{})))

	// Test invalid ID
	t.Run("Invalid ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/articles/invalid/summary", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test article not found
	t.Run("Article Not Found", func(t *testing.T) {
		mockDB := new(MockDBOperations)
		mockDB.On("FetchArticleByID", mock.Anything, int64(999)).Return(nil, db.ErrArticleNotFound)

		router := gin.New()
		router.GET("/api/articles/:id/summary", SafeHandler(summaryHandler(&sqlx.DB{})))

		req, _ := http.NewRequest("GET", "/api/articles/999/summary", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
