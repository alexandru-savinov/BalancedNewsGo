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

// Tests for article-related handlers with low coverage:
// - createArticleHandler
// - getArticlesHandler
// - getArticleByIDHandler

// TestCreateArticleHandler tests the createArticleHandler function
func TestCreateArticleHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock expectations
		mockDB.On("ArticleExistsByURL", mock.Anything, "https://example.com/article").Return(false, nil)
		mockDB.On("InsertArticle", mock.Anything, mock.AnythingOfType("*db.Article")).Return(int64(1), nil)
		mockDB.On("FetchArticleByID", mock.Anything, int64(1)).Return(&db.Article{
			ID:     1,
			Source: "test-source",
			URL:    "https://example.com/article",
			Title:  "Test Article",
		}, nil)

		// Create router
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

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["article_id"])

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Missing Required Fields", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router
		router := gin.New()
		router.POST("/api/articles", SafeHandler(createArticleHandler(&sqlx.DB{})))

		// Create request with missing fields
		invalidArticle := `{
			"source": "test-source",
			"url": "https://example.com/article"
		}`
		req, _ := http.NewRequest("POST", "/api/articles", bytes.NewBuffer([]byte(invalidArticle)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Missing required fields")
	})

	t.Run("Invalid URL Format", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router
		router := gin.New()
		router.POST("/api/articles", SafeHandler(createArticleHandler(&sqlx.DB{})))

		// Create request with invalid URL format
		invalidArticle := `{
			"source": "test-source",
			"pub_date": "2025-04-30T12:00:00Z",
			"url": "invalid-url",
			"title": "Test Article",
			"content": "This is a test article."
		}`
		req, _ := http.NewRequest("POST", "/api/articles", bytes.NewBuffer([]byte(invalidArticle)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Invalid URL format")
	})

	t.Run("Invalid pub_date Format", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router
		router := gin.New()
		router.POST("/api/articles", SafeHandler(createArticleHandler(&sqlx.DB{})))

		// Create request with invalid pub_date format
		invalidArticle := `{
			"source": "test-source",
			"pub_date": "2025/04/30",
			"url": "https://example.com/article",
			"title": "Test Article",
			"content": "This is a test article."
		}`
		req, _ := http.NewRequest("POST", "/api/articles", bytes.NewBuffer([]byte(invalidArticle)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Invalid pub_date format")
	})

	t.Run("Duplicate URL", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return article exists
		mockDB.On("ArticleExistsByURL", mock.Anything, "https://example.com/duplicate").Return(true, nil)

		// Create router
		router := gin.New()
		router.POST("/api/articles", SafeHandler(createArticleHandler(&sqlx.DB{})))

		// Create request with duplicate URL
		duplicateArticle := `{
			"source": "test-source",
			"pub_date": "2025-04-30T12:00:00Z",
			"url": "https://example.com/duplicate",
			"title": "Test Article",
			"content": "This is a test article."
		}`
		req, _ := http.NewRequest("POST", "/api/articles", bytes.NewBuffer([]byte(duplicateArticle)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "already exists")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Extra Fields", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router
		router := gin.New()
		router.POST("/api/articles", SafeHandler(createArticleHandler(&sqlx.DB{})))

		// Create request with extra fields
		invalidArticle := `{
			"source": "test-source",
			"pub_date": "2025-04-30T12:00:00Z",
			"url": "https://example.com/article",
			"title": "Test Article",
			"content": "This is a test article.",
			"extra_field": "This field shouldn't be here"
		}`
		req, _ := http.NewRequest("POST", "/api/articles", bytes.NewBuffer([]byte(invalidArticle)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Database Error - Check Existing", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return error
		mockDB.On("ArticleExistsByURL", mock.Anything, mock.Anything).Return(false, errors.New("database error"))

		// Create router
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

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Database Error - Insert", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock expectations
		mockDB.On("ArticleExistsByURL", mock.Anything, mock.Anything).Return(false, nil)
		mockDB.On("InsertArticle", mock.Anything, mock.AnythingOfType("*db.Article")).Return(int64(0), errors.New("database error"))

		// Create router
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

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})
}

// TestGetArticlesHandler tests the getArticlesHandler function
func TestGetArticlesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success - Default Parameters", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create test articles
		articles := []db.Article{
			{ID: 1, Title: "Article 1", Source: "source1"},
			{ID: 2, Title: "Article 2", Source: "source2"},
		}

		// Set up mock expectations
		mockDB.On("FetchArticles", mock.Anything, "", "", 20, 0).Return(articles, nil)
		mockDB.On("FetchLLMScores", mock.Anything, int64(1)).Return([]db.LLMScore{}, nil)
		mockDB.On("FetchLLMScores", mock.Anything, int64(2)).Return([]db.LLMScore{}, nil)

		// Create router
		router := gin.New()
		router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

		// Create request with default parameters
		req, _ := http.NewRequest("GET", "/api/articles", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.True(t, response["success"].(bool))

		data, ok := response["data"].([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 2, len(data))

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Success - With Filters", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create test articles
		articles := []db.Article{
			{ID: 3, Title: "Article 3", Source: "cnn"},
		}

		// Set up mock expectations
		mockDB.On("FetchArticles", mock.Anything, "cnn", "left", 5, 10).Return(articles, nil)
		mockDB.On("FetchLLMScores", mock.Anything, int64(3)).Return([]db.LLMScore{}, nil)

		// Create router
		router := gin.New()
		router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

		// Create request with filters
		req, _ := http.NewRequest("GET", "/api/articles?source=cnn&leaning=left&limit=5&offset=10", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.True(t, response["success"].(bool))

		data, ok := response["data"].([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 1, len(data))

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Invalid Limit Parameter", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router
		router := gin.New()
		router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

		// Create request with invalid limit
		req, _ := http.NewRequest("GET", "/api/articles?limit=invalid", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Invalid 'limit' parameter")
	})

	t.Run("Limit Out Of Range", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router with different limit values
		router := gin.New()
		router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

		// Test limit = 0
		req, _ := http.NewRequest("GET", "/api/articles?limit=0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Test limit = -5
		req, _ = http.NewRequest("GET", "/api/articles?limit=-5", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Test limit = 101 (exceeds maximum)
		req, _ = http.NewRequest("GET", "/api/articles?limit=101", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid Offset Parameter", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router
		router := gin.New()
		router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

		// Create request with invalid offset
		req, _ := http.NewRequest("GET", "/api/articles?offset=invalid", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Invalid 'offset' parameter")
	})

	t.Run("Negative Offset", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router
		router := gin.New()
		router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

		// Create request with negative offset
		req, _ := http.NewRequest("GET", "/api/articles?offset=-1", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Database Error", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return error
		mockDB.On("FetchArticles", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]db.Article{}, errors.New("database error"))

		// Create router
		router := gin.New()
		router.GET("/api/articles", SafeHandler(getArticlesHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Failed to fetch articles")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})
}

// TestGetArticleByIDHandler tests the getArticleByIDHandler function
func TestGetArticleByIDHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up test data
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

		// Set up mock expectations
		mockDB.On("FetchArticleByID", mock.Anything, int64(1)).Return(article, nil)

		// Create router
		router := gin.New()
		router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/1", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
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

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router
		router := gin.New()
		router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

		// Create request with invalid ID
		req, _ := http.NewRequest("GET", "/api/articles/invalid", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Invalid article ID")
	})

	t.Run("Negative ID", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Create router
		router := gin.New()
		router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

		// Create request with negative ID
		req, _ := http.NewRequest("GET", "/api/articles/-1", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Article Not Found", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return not found
		mockDB.On("FetchArticleByID", mock.Anything, int64(999)).Return(nil, db.ErrArticleNotFound)

		// Create router
		router := gin.New()
		router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/999", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Article not found")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Database Error", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return error
		mockDB.On("FetchArticleByID", mock.Anything, int64(2)).Return(nil, errors.New("database error"))

		// Create router
		router := gin.New()
		router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/2", nil)
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Failed to fetch article")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Cache Test", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up test article
		score := 0.5
		confidence := 0.8
		article := &db.Article{
			ID:             3,
			Title:          "Cached Article",
			Content:        "Test content",
			CompositeScore: &score,
			Confidence:     &confidence,
		}

		// Set up mock to be called only once (for first request)
		mockDB.On("FetchArticleByID", mock.Anything, int64(3)).Return(article, nil).Once()

		// Create router
		router := gin.New()
		router.GET("/api/articles/:id", SafeHandler(getArticleByIDHandler(&sqlx.DB{})))

		// First request
		req1, _ := http.NewRequest("GET", "/api/articles/3", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request should use cache
		req2, _ := http.NewRequest("GET", "/api/articles/3", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)

		// Verify mock was called exactly once
		mockDB.AssertExpectations(t)
	})
}
