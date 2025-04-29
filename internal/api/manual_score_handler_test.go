package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Utility functions and constants for testing
const manualScoreEndpointTest = "/api/manual-score/:id"

// MockDBOperations implements the db.DBOperations interface for testing
type MockDBOperationsScore struct {
	mock.Mock
}

// Mock all required methods from db.DBOperations
// FetchArticleByID mocks the db.FetchArticleByID function
func (m *MockDBOperationsScore) FetchArticleByID(ctx any, articleID int64) (*db.Article, error) {
	args := m.Called(ctx, articleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Article), args.Error(1)
}

// GetArticleByID is an alias for FetchArticleByID
func (m *MockDBOperationsScore) GetArticleByID(ctx context.Context, id int64) (*db.Article, error) {
	return m.FetchArticleByID(ctx, id)
}

// UpdateArticleScore mocks the db.UpdateArticleScore function
func (m *MockDBOperationsScore) UpdateArticleScore(ctx any, articleID int64, score float64, confidence float64) error {
	args := m.Called(ctx, articleID, score, confidence)
	return args.Error(0)
}

// Implement remaining required methods from DBOperations interface with stub implementations
func (m *MockDBOperationsScore) ArticleExistsByURL(ctx context.Context, url string) (bool, error) {
	args := m.Called(ctx, url)
	return args.Bool(0), args.Error(1)
}

func (m *MockDBOperationsScore) GetArticles(ctx context.Context, filter db.ArticleFilter) ([]*db.Article, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*db.Article), args.Error(1)
}

func (m *MockDBOperationsScore) FetchArticles(ctx context.Context, source, leaning string, limit, offset int) ([]*db.Article, error) {
	args := m.Called(ctx, source, leaning, limit, offset)
	return args.Get(0).([]*db.Article), args.Error(1)
}

func (m *MockDBOperationsScore) InsertArticle(ctx context.Context, article *db.Article) (int64, error) {
	args := m.Called(ctx, article)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDBOperationsScore) UpdateArticleScoreObj(ctx context.Context, articleID int64, score *db.ArticleScore, confidence float64) error {
	args := m.Called(ctx, articleID, score, confidence)
	return args.Error(0)
}

func (m *MockDBOperationsScore) SaveArticleFeedback(ctx context.Context, feedback *db.ArticleFeedback) error {
	args := m.Called(ctx, feedback)
	return args.Error(0)
}

func (m *MockDBOperationsScore) InsertFeedback(ctx context.Context, feedback *db.Feedback) error {
	args := m.Called(ctx, feedback)
	return args.Error(0)
}

func (m *MockDBOperationsScore) FetchLLMScores(ctx context.Context, articleID int64) ([]db.LLMScore, error) {
	args := m.Called(ctx, articleID)
	return args.Get(0).([]db.LLMScore), args.Error(1)
}

// Setup router with mock DB for manual score handler tests
func setupManualScoreRouter(mock db.DBOperations) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST(manualScoreEndpointTest, SafeHandler(manualScoreHandler(mock)))
	return router
}

func TestManualScoreHandler_Success(t *testing.T) {
	// Setup mock DB and router
	mock := &MockDBOperationsScore{}
	// Expect article to exist
	mock.On("FetchArticleByID", mock.Anything, int64(1)).Return(&db.Article{ID: 1}, nil)
	// Expect score update to succeed
	mock.On("UpdateArticleScore", mock.Anything, int64(1), 0.5, float64(0)).Return(nil)

	router := setupManualScoreRouter(mock)

	// Test valid score update
	body := `{"score": 0.5}`
	req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Verify response
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "score updated", data["status"])
	assert.Equal(t, float64(1), data["article_id"])
	assert.Equal(t, 0.5, data["score"])

	// Verify expectations
	mock.AssertExpectations(t)
}

func TestManualScoreHandler_IntegerScore(t *testing.T) {
	// Setup mock DB and router
	mock := &MockDBOperationsScore{}
	mock.On("FetchArticleByID", mock.Anything, int64(1)).Return(&db.Article{ID: 1}, nil)
	mock.On("UpdateArticleScore", mock.Anything, int64(1), -1.0, float64(0)).Return(nil)

	router := setupManualScoreRouter(mock)

	// Test with integer score that should be converted to float
	body := `{"score": -1}`
	req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	// Verify response
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, -1.0, data["score"])

	// Verify expectations
	mock.AssertExpectations(t)
}

func TestManualScoreHandler_InvalidID(t *testing.T) {
	mock := &MockDBOperationsScore{}
	router := setupManualScoreRouter(mock)

	testCases := []struct {
		name        string
		articleID   string
		expectedMsg string
	}{
		{
			name:        "Non-numeric ID",
			articleID:   "abc",
			expectedMsg: "Invalid article ID",
		},
		{
			name:        "Negative ID",
			articleID:   "-1",
			expectedMsg: "Invalid article ID",
		},
		{
			name:        "Zero ID",
			articleID:   "0",
			expectedMsg: "Invalid article ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"score": 0.5}`
			req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", tc.articleID, 1), bytes.NewBuffer([]byte(body)))
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
			assert.Contains(t, response["error_message"].(string), tc.expectedMsg)
		})
	}
}

func TestManualScoreHandler_InvalidPayload(t *testing.T) {
	mock := &MockDBOperationsScore{}
	router := setupManualScoreRouter(mock)

	testCases := []struct {
		name        string
		payload     string
		expectedMsg string
	}{
		{
			name:        "Empty payload",
			payload:     `{}`,
			expectedMsg: "Payload must contain only 'score' field",
		},
		{
			name:        "Missing score field",
			payload:     `{"other": 123}`,
			expectedMsg: "Payload must contain only 'score' field",
		},
		{
			name:        "Extra fields",
			payload:     `{"score": 0.5, "extra": "field"}`,
			expectedMsg: "Invalid JSON body",
		},
		{
			name:        "Invalid JSON",
			payload:     `{score: 0.5`,
			expectedMsg: "Invalid JSON body",
		},
		{
			name:        "Non-numeric score",
			payload:     `{"score": "not-a-number"}`,
			expectedMsg: "'score' must be a number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", "1", 1), bytes.NewBuffer([]byte(tc.payload)))
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
			assert.Contains(t, response["error_message"].(string), tc.expectedMsg)
		})
	}
}

func TestManualScoreHandler_ScoreOutOfRange(t *testing.T) {
	mock := &MockDBOperationsScore{}
	router := setupManualScoreRouter(mock)

	testCases := []struct {
		name  string
		score float64
	}{
		{
			name:  "Score too low",
			score: -1.1,
		},
		{
			name:  "Score too high",
			score: 1.1,
		},
		{
			name:  "Score much too low",
			score: -100,
		},
		{
			name:  "Score much too high",
			score: 100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(map[string]float64{"score": tc.score})
			req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", "1", 1), bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
			assert.Contains(t, response["error_message"].(string), "Score must be between -1.0 and 1.0")
		})
	}
}

func TestManualScoreHandler_ArticleNotFound(t *testing.T) {
	mock := &MockDBOperationsScore{}
	mock.On("FetchArticleByID", mock.Anything, int64(999)).Return(nil, db.ErrArticleNotFound)

	router := setupManualScoreRouter(mock)

	body := `{"score": 0.5}`
	req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", "999", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error_message"].(string), "Article not found")

	mock.AssertExpectations(t)
}

func TestManualScoreHandler_DatabaseError(t *testing.T) {
	testCases := []struct {
		name         string
		fetchError   error
		updateError  error
		expectedCode int
		expectedMsg  string
	}{
		{
			name:         "Fetch article error",
			fetchError:   fmt.Errorf("database connection lost"),
			updateError:  nil,
			expectedCode: http.StatusInternalServerError,
			expectedMsg:  "Failed to fetch article",
		},
		{
			name:         "Update score error - constraint violation",
			fetchError:   nil,
			updateError:  fmt.Errorf("UNIQUE constraint failed"),
			expectedCode: http.StatusBadRequest,
			expectedMsg:  "Failed to update score due to invalid data or constraint violation",
		},
		{
			name:         "Update score error - database error",
			fetchError:   nil,
			updateError:  fmt.Errorf("database timeout"),
			expectedCode: http.StatusInternalServerError,
			expectedMsg:  "Failed to update article score",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockDBOperationsScore{}

			if tc.fetchError != nil {
				mock.On("FetchArticleByID", mock.Anything, int64(1)).Return(nil, tc.fetchError)
			} else {
				mock.On("FetchArticleByID", mock.Anything, int64(1)).Return(&db.Article{ID: 1}, nil)
				mock.On("UpdateArticleScore", mock.Anything, int64(1), 0.5, float64(0)).Return(tc.updateError)
			}

			router := setupManualScoreRouter(mock)

			body := `{"score": 0.5}`
			req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			assert.Equal(t, tc.expectedCode, recorder.Code)

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
			assert.Contains(t, response["error_message"].(string), tc.expectedMsg)

			mock.AssertExpectations(t)
		})
	}
}

func TestManualScoreHandler_BoundaryScores(t *testing.T) {
	testCases := []struct {
		name  string
		score float64
	}{
		{
			name:  "Minimum valid score",
			score: -1.0,
		},
		{
			name:  "Maximum valid score",
			score: 1.0,
		},
		{
			name:  "Zero score",
			score: 0.0,
		},
		{
			name:  "Near minimum score",
			score: -0.999,
		},
		{
			name:  "Near maximum score",
			score: 0.999,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockDBOperationsScore{}
			mock.On("FetchArticleByID", mock.Anything, int64(1)).Return(&db.Article{ID: 1}, nil)
			mock.On("UpdateArticleScore", mock.Anything, int64(1), tc.score, float64(0)).Return(nil)

			router := setupManualScoreRouter(mock)

			payload, _ := json.Marshal(map[string]float64{"score": tc.score})
			req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", "1", 1), bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.True(t, response["success"].(bool))
			data := response["data"].(map[string]interface{})
			assert.Equal(t, tc.score, data["score"])

			mock.AssertExpectations(t)
		})
	}
}

func TestManualScoreHandler_PanicRecovery(t *testing.T) {
	// Create a mock that will panic
	mock := &MockDBOperationsScore{}
	mock.On("FetchArticleByID", mock.Anything, int64(1)).Run(func(args mock.Arguments) {
		panic("simulated panic in handler")
	}).Return(nil, nil)

	router := setupManualScoreRouter(mock)

	body := `{"score": 0.5}`
	req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", "1", 1), bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

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

func TestGetArticleInputValidation(t *testing.T) {
	// This test is focused on validation of article ID input
	mock := &MockDBOperationsScore{}
	router := setupManualScoreRouter(mock)

	// Test cases with invalid article IDs
	invalidIDs := []string{"0", "-1", "abc", "1.23", "9999999999999999999"}

	for _, id := range invalidIDs {
		t.Run("Invalid ID: "+id, func(t *testing.T) {
			body := `{"score": 0.5}`
			req, _ := http.NewRequest("POST", strings.Replace(manualScoreEndpointTest, ":id", id, 1), bytes.NewBuffer([]byte(body)))
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			// IDs that are valid integers but <= 0 or too large should return a validation error
			expectedCode := http.StatusBadRequest
			assert.Equal(t, expectedCode, recorder.Code, "Expected status code %d for ID '%s'", expectedCode, id)

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool))
		})
	}
}
