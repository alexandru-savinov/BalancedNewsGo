package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDBOperationsArticles implements db.DBOperations interface for testing the articles handler
type MockDBOperationsArticles struct {
	mock.Mock
}

// FetchArticles mocks the db.FetchArticles function
func (m *MockDBOperationsArticles) FetchArticles(ctx any, source, leaning string, limit, offset int) ([]*db.Article, error) {
	args := m.Called(ctx, source, leaning, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*db.Article), args.Error(1)
}

// FetchLLMScores mocks the db.FetchLLMScores function
func (m *MockDBOperationsArticles) FetchLLMScores(ctx any, articleID int64) ([]db.LLMScore, error) {
	args := m.Called(ctx, articleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.LLMScore), args.Error(1)
}

// Setup router with mock DB for get articles handler tests
func setupArticlesRouter(mock db.DBOperations) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/articles", SafeHandler(getArticlesHandler(mock)))
	return router
}

// Helper function to create dummy scores
func createDummyLLMScores(confidence float64) []db.LLMScore {
	metadataJSON := `{"confidence": ` + string(json.RawMessage([]byte(json.Number(confidence).String()))) + `}`

	return []db.LLMScore{
		{
			ID:        1,
			ArticleID: 1,
			Model:     "model1",
			Score:     0.5,
			Metadata:  metadataJSON,
			CreatedAt: time.Now(),
		},
	}
}

// Helper function to create dummy articles
func createDummyArticles(count int) []*db.Article {
	articles := make([]*db.Article, count)
	for i := 0; i < count; i++ {
		score := 0.1 * float64(i)
		confidence := 0.8
		articles[i] = &db.Article{
			ID:             int64(i + 1),
			Source:         "source" + string(rune(i%3+'A')),
			PubDate:        time.Now().Add(-time.Duration(i) * time.Hour),
			URL:            "http://example.com/article" + string(rune(i+'0')),
			Title:          "Title " + string(rune(i+'0')),
			Content:        "Content " + string(rune(i+'0')),
			CreatedAt:      time.Now().Add(-time.Duration(i) * time.Hour),
			CompositeScore: &score,
			Confidence:     &confidence,
		}
	}
	return articles
}

func TestGetArticlesHandler_Success(t *testing.T) {
	// Setup mock DB and router
	mock := &MockDBOperationsArticles{}

	// Create test articles
	articles := createDummyArticles(3)

	// Setup mock expectations
	mock.On("FetchArticles", mock.Anything, "", "", 20, 0).Return(articles, nil)

	// Setup scores for each article
	for _, article := range articles {
		mock.On("FetchLLMScores", mock.Anything, article.ID).Return(createDummyLLMScores(0.8), nil)
	}

	router := setupArticlesRouter(mock)

	// Test with default parameters
	req, _ := http.NewRequest("GET", "/api/articles", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Verify response
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	data, ok := response["data"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 3, len(data), "Expected 3 articles in the response")

	// Verify first article fields
	firstArticle := data[0].(map[string]interface{})
	assert.Equal(t, float64(1), firstArticle["article_id"])
	assert.Equal(t, "Title 0", firstArticle["Title"])

	// Verify expectations
	mock.AssertExpectations(t)
}

func TestGetArticlesHandler_WithParameters(t *testing.T) {
	testCases := []struct {
		name       string
		source     string
		leaning    string
		limit      int
		offset     int
		queryParam string
	}{
		{
			name:       "Filter by source",
			source:     "CNN",
			leaning:    "",
			limit:      20,
			offset:     0,
			queryParam: "?source=CNN",
		},
		{
			name:       "Filter by leaning",
			source:     "",
			leaning:    "left",
			limit:      20,
			offset:     0,
			queryParam: "?leaning=left",
		},
		{
			name:       "Custom limit",
			source:     "",
			leaning:    "",
			limit:      5,
			offset:     0,
			queryParam: "?limit=5",
		},
		{
			name:       "Pagination with offset",
			source:     "",
			leaning:    "",
			limit:      20,
			offset:     10,
			queryParam: "?offset=10",
		},
		{
			name:       "Combined parameters",
			source:     "FOX",
			leaning:    "right",
			limit:      10,
			offset:     5,
			queryParam: "?source=FOX&leaning=right&limit=10&offset=5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB and router
			mock := &MockDBOperationsArticles{}

			// Create test articles
			articles := createDummyArticles(2)

			// Setup mock expectations
			mock.On("FetchArticles", mock.Anything, tc.source, tc.leaning, tc.limit, tc.offset).Return(articles, nil)

			// Setup scores for each article
			for _, article := range articles {
				mock.On("FetchLLMScores", mock.Anything, article.ID).Return(createDummyLLMScores(0.8), nil)
			}

			router := setupArticlesRouter(mock)

			// Test with parameters
			req, _ := http.NewRequest("GET", "/api/articles"+tc.queryParam, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			// Verify response
			assert.Equal(t, http.StatusOK, recorder.Code)

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.True(t, response["success"].(bool))

			data := response["data"].([]interface{})
			assert.Len(t, data, 2)

			// Verify expectations
			mock.AssertExpectations(t)
		})
	}
}

func TestGetArticlesHandler_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name        string
		queryParam  string
		expectedMsg string
	}{
		{
			name:        "Invalid limit - negative",
			queryParam:  "?limit=-5",
			expectedMsg: "Invalid 'limit' parameter",
		},
		{
			name:        "Invalid limit - zero",
			queryParam:  "?limit=0",
			expectedMsg: "Invalid 'limit' parameter",
		},
		{
			name:        "Invalid limit - too large",
			queryParam:  "?limit=101",
			expectedMsg: "Invalid 'limit' parameter",
		},
		{
			name:        "Invalid limit - non-numeric",
			queryParam:  "?limit=abc",
			expectedMsg: "Invalid 'limit' parameter",
		},
		{
			name:        "Invalid offset - negative",
			queryParam:  "?offset=-1",
			expectedMsg: "Invalid 'offset' parameter",
		},
		{
			name:        "Invalid offset - non-numeric",
			queryParam:  "?offset=abc",
			expectedMsg: "Invalid 'offset' parameter",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB and router
			mock := &MockDBOperationsArticles{}
			router := setupArticlesRouter(mock)

			// Test with invalid parameters
			req, _ := http.NewRequest("GET", "/api/articles"+tc.queryParam, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			// Verify response
			assert.Equal(t, http.StatusBadRequest, recorder.Code)

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
			assert.Contains(t, response["error_message"].(string), tc.expectedMsg)
		})
	}
}

func TestGetArticlesHandler_DatabaseError(t *testing.T) {
	// Setup mock DB and router
	mock := &MockDBOperationsArticles{}

	// Setup mock expectations - database error
	mock.On("FetchArticles", mock.Anything, "", "", 20, 0).Return(nil, errors.New("database connection error"))

	router := setupArticlesRouter(mock)

	// Test with default parameters
	req, _ := http.NewRequest("GET", "/api/articles", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Verify response
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error_message"].(string), "Failed to fetch articles")

	// Verify expectations
	mock.AssertExpectations(t)
}

func TestGetArticlesHandler_EmptyResults(t *testing.T) {
	// Setup mock DB and router
	mock := &MockDBOperationsArticles{}

	// Setup mock expectations - empty result
	var emptyArticles []*db.Article
	mock.On("FetchArticles", mock.Anything, "nonexistent", "", 20, 0).Return(emptyArticles, nil)

	router := setupArticlesRouter(mock)

	// Test with source that returns no results
	req, _ := http.NewRequest("GET", "/api/articles?source=nonexistent", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Verify response
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	data := response["data"].([]interface{})
	assert.Empty(t, data, "Expected empty article list")

	// Verify expectations
	mock.AssertExpectations(t)
}

func TestGetArticlesHandler_PanicRecovery(t *testing.T) {
	// Create a mock that will panic
	mock := &MockDBOperationsArticles{}
	mock.On("FetchArticles", mock.Anything, "", "", 20, 0).Run(func(args mock.Arguments) {
		panic("simulated panic in handler")
	}).Return(nil, nil)

	router := setupArticlesRouter(mock)

	// Test with default parameters
	req, _ := http.NewRequest("GET", "/api/articles", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// The SafeHandler should recover and return a 500 error
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error_message"].(string), "Internal server error")
}

func TestGetArticlesHandler_LLMScoreErrors(t *testing.T) {
	// Setup mock DB and router
	mock := &MockDBOperationsArticles{}

	// Create test articles
	articles := createDummyArticles(2)

	// Setup mock expectations
	mock.On("FetchArticles", mock.Anything, "", "", 20, 0).Return(articles, nil)

	// Setup score fetch error for the first article
	mock.On("FetchLLMScores", mock.Anything, int64(1)).Return(nil, errors.New("score fetch error"))

	// Setup successful score fetch for the second article
	mock.On("FetchLLMScores", mock.Anything, int64(2)).Return(createDummyLLMScores(0.8), nil)

	router := setupArticlesRouter(mock)

	// Test with default parameters
	req, _ := http.NewRequest("GET", "/api/articles", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Should still return 200 OK even if scores can't be fetched
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	// Both articles should still be returned
	data := response["data"].([]interface{})
	assert.Len(t, data, 2)

	// Verify expectations
	mock.AssertExpectations(t)
}

func TestGetArticlesHandler_CacheReuse(t *testing.T) {
	// Setup mock DB and router
	mock := &MockDBOperationsArticles{}

	// Create test articles
	articles := createDummyArticles(1)

	// Setup mock expectations - only called once
	mock.On("FetchArticles", mock.Anything, "", "", 20, 0).Return(articles, nil).Once()
	mock.On("FetchLLMScores", mock.Anything, int64(1)).Return(createDummyLLMScores(0.8), nil).Once()

	router := setupArticlesRouter(mock)

	// First request
	req1, _ := http.NewRequest("GET", "/api/articles", nil)
	recorder1 := httptest.NewRecorder()
	router.ServeHTTP(recorder1, req1)

	// Verify first response
	assert.Equal(t, http.StatusOK, recorder1.Code)

	// Second request should use cache
	req2, _ := http.NewRequest("GET", "/api/articles", nil)
	recorder2 := httptest.NewRecorder()
	router.ServeHTTP(recorder2, req2)

	// Verify second response
	assert.Equal(t, http.StatusOK, recorder2.Code)

	// Both responses should be identical
	assert.Equal(t, recorder1.Body.String(), recorder2.Body.String())

	// Verify expectations - DB was only called once
	mock.AssertExpectations(t)
}

func TestGetArticlesHandler_EdgeCases(t *testing.T) {
	testCases := []struct {
		name   string
		limit  int
		offset int
	}{
		{
			name:   "Maximum valid limit",
			limit:  100,
			offset: 0,
		},
		{
			name:   "Minimum valid limit",
			limit:  1,
			offset: 0,
		},
		{
			name:   "High offset",
			limit:  20,
			offset: 1000000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB and router
			mock := &MockDBOperationsArticles{}

			// Setup mock expectations
			var emptyArticles []*db.Article
			mock.On("FetchArticles", mock.Anything, "", "", tc.limit, tc.offset).Return(emptyArticles, nil)

			router := setupArticlesRouter(mock)

			// Test with edge case parameters
			req, _ := http.NewRequest("GET", fmt.Sprintf("/api/articles?limit=%d&offset=%d", tc.limit, tc.offset), nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			// Verify response
			assert.Equal(t, http.StatusOK, recorder.Code)

			// Verify expectations
			mock.AssertExpectations(t)
		})
	}
}
