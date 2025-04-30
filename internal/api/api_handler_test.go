package api

import (
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

// Tests for the handler functions with 0% coverage, focusing on:
// - summaryHandler
// - biasHandler
// - ensembleDetailsHandler

// TestSummaryHandler tests the summary handler functionality
func TestSummaryHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success - Summary exists", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock DB to return a valid article and summary
		mockDB.On("FetchArticleByID", mock.Anything, int64(1)).Return(&db.Article{ID: 1}, nil)
		mockDB.On("FetchLLMScores", mock.Anything, int64(1)).Return([]db.LLMScore{
			{Model: "summarizer", Metadata: `{"summary": "This is a test summary"}`, CreatedAt: time.Now()},
		}, nil)

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/summary", summaryHandler(&sqlx.DB{})) // Passing empty DB as it's mocked

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/1/summary", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "This is a test summary", data["summary"])
		assert.Contains(t, data, "created_at")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Not Found - No summary", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock DB to return a valid article but no summary
		mockDB.On("FetchArticleByID", mock.Anything, int64(2)).Return(&db.Article{ID: 2}, nil)
		mockDB.On("FetchLLMScores", mock.Anything, int64(2)).Return([]db.LLMScore{
			{Model: "gpt", Metadata: `{}`, CreatedAt: time.Now()},
		}, nil)

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/summary", SafeHandler(summaryHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/2/summary", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "summary not available")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Invalid Article ID", func(t *testing.T) {
		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/summary", SafeHandler(summaryHandler(&sqlx.DB{})))

		// Create request with invalid ID
		req, _ := http.NewRequest("GET", "/api/articles/invalid/summary", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Verify response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Invalid article ID")
	})

	t.Run("Article Not Found", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock DB to return article not found error
		mockDB.On("FetchArticleByID", mock.Anything, int64(999)).Return(nil, db.ErrArticleNotFound)

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/summary", SafeHandler(summaryHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/999/summary", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestBiasHandler tests the bias handler functionality
func TestBiasHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success - With Ensemble Score", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return scores with ensemble
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

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/bias", SafeHandler(biasHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/1/bias", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, 0.75, data["composite_score"])
		assert.IsType(t, []interface{}{}, data["results"])
		results := data["results"].([]interface{})
		assert.Equal(t, 2, len(results)) // Should include both gpt and claude scores

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Success - Without Ensemble Score", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return scores without ensemble
		now := time.Now()
		mockDB.On("FetchLLMScores", mock.Anything, int64(2)).Return([]db.LLMScore{
			{
				Model:     "gpt",
				Score:     0.7,
				Metadata:  `{"Confidence":0.8,"Explanation":"Liberal leaning"}`,
				CreatedAt: now,
			},
		}, nil)

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/bias", SafeHandler(biasHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/2/bias", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Nil(t, data["composite_score"]) // Should be nil when no ensemble score exists
		assert.Equal(t, "scoring_unavailable", data["status"])

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Query Parameters", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock
		now := time.Now()
		mockDB.On("FetchLLMScores", mock.Anything, int64(3)).Return([]db.LLMScore{
			{
				Model:     "ensemble",
				Score:     0.75,
				Metadata:  `{"aggregation":{"weighted_mean":0.75,"confidence":0.8}}`,
				CreatedAt: now,
			},
			{
				Model:     "gpt",
				Score:     0.7, // Within filter range
				Metadata:  `{"Confidence":0.8,"Explanation":"Liberal leaning"}`,
				CreatedAt: now,
			},
			{
				Model:     "claude",
				Score:     -0.1, // Outside filter range
				Metadata:  `{"Confidence":0.9,"Explanation":"Somewhat liberal"}`,
				CreatedAt: now,
			},
		}, nil)

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/bias", SafeHandler(biasHandler(&sqlx.DB{})))

		// Create request with min_score
		req, _ := http.NewRequest("GET", "/api/articles/3/bias?min_score=0.0&max_score=1.0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		data := response["data"].(map[string]interface{})
		results := data["results"].([]interface{})
		assert.Equal(t, 1, len(results)) // Should only include gpt score which is within range

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Invalid Parameters", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/articles/:id/bias", SafeHandler(biasHandler(&sqlx.DB{})))

		// Test invalid min_score
		req, _ := http.NewRequest("GET", "/api/articles/1/bias?min_score=invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Test invalid max_score
		req, _ = http.NewRequest("GET", "/api/articles/1/bias?max_score=invalid", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Test invalid sort order
		req, _ = http.NewRequest("GET", "/api/articles/1/bias?sort=invalid", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestEnsembleDetailsHandler tests the ensemble details handler
func TestEnsembleDetailsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return ensemble scores
		mockDB.On("FetchLLMScores", mock.Anything, int64(1)).Return([]db.LLMScore{
			{
				ID:        1,
				Model:     "ensemble",
				Score:     0.75,
				Metadata:  `{"sub_results":[{"model":"gpt","score":0.7},{"model":"claude","score":0.8}],"aggregation":{"weighted_mean":0.75}}`,
				CreatedAt: time.Now(),
			},
		}, nil)

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/1/ensemble", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.True(t, response["success"].(bool))
		scores := response["scores"].([]interface{})
		assert.Equal(t, 1, len(scores))

		scoreData := scores[0].(map[string]interface{})
		assert.Equal(t, 0.75, scoreData["score"])
		assert.Contains(t, scoreData, "sub_results")
		assert.Contains(t, scoreData, "aggregation")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Invalid Metadata", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return ensemble scores with invalid metadata
		mockDB.On("FetchLLMScores", mock.Anything, int64(2)).Return([]db.LLMScore{
			{
				ID:        2,
				Model:     "ensemble",
				Score:     0.75,
				Metadata:  `{invalid-json`, // Invalid JSON
				CreatedAt: time.Now(),
			},
		}, nil)

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/2/ensemble", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response - should still return 200 but with error in data
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.True(t, response["success"].(bool))
		scores := response["scores"].([]interface{})
		assert.Equal(t, 1, len(scores))

		scoreData := scores[0].(map[string]interface{})
		assert.Equal(t, 0.75, scoreData["score"])
		assert.Contains(t, scoreData, "error") // Should contain error message

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("No Ensemble Data", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return non-ensemble scores
		mockDB.On("FetchLLMScores", mock.Anything, int64(3)).Return([]db.LLMScore{
			{
				Model:     "gpt",
				Score:     0.7,
				Metadata:  `{"Confidence":0.8}`,
				CreatedAt: time.Now(),
			},
		}, nil)

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/3/ensemble", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Ensemble data not found")

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Database Error", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return error
		mockDB.On("FetchLLMScores", mock.Anything, int64(4)).Return(nil, errors.New("database error"))

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(&sqlx.DB{})))

		// Create request
		req, _ := http.NewRequest("GET", "/api/articles/4/ensemble", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	t.Run("Cache Busting", func(t *testing.T) {
		mockDB := &MockDBOperations{}

		// Set up mock to return ensemble scores
		mockDB.On("FetchLLMScores", mock.Anything, int64(5)).Return([]db.LLMScore{
			{
				Model:     "ensemble",
				Score:     0.75,
				Metadata:  `{"sub_results":[],"aggregation":{}}`,
				CreatedAt: time.Now(),
			},
		}, nil)

		// Create router and register handler
		router := gin.New()
		router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(&sqlx.DB{})))

		// Create request with cache busting parameter
		req, _ := http.NewRequest("GET", "/api/articles/5/ensemble?_t=123456", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})
}
