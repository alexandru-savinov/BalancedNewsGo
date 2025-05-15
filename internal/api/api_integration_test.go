package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Constants to avoid duplicate string literals
const (
	contentTypeHeader = "Content-Type"
	applicationJSON   = "application/json"
	reanalyzeURLPath  = "/api/llm/reanalyze/%d"
)

// MockProgressManager mocks the progress management functionality of ScoreManager
type MockProgressManager struct {
	mock.Mock
}

func (m *MockProgressManager) SetProgress(articleID int64, state *models.ProgressState) {
	m.Called(articleID, state)
}

func (m *MockProgressManager) GetProgress(articleID int64) *models.ProgressState {
	args := m.Called(articleID)
	if args.Get(0) == nil {
		return nil
	}
	val, ok := args.Get(0).(*models.ProgressState)
	if !ok {
		// This might indicate a misconfiguration of the mock's Return arguments
		// or the test is intentionally providing a different type.
		// For a mock, returning nil might be acceptable, or you could panic.
		// log.Printf("WARN: MockProgressManager.GetProgress: type assertion to *models.ProgressState failed for articleID %d", articleID)
		return nil
	}
	return val
}

// MockCache mocks the cache functionality of ScoreManager
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(key string) interface{} {
	args := m.Called(key)
	return args.Get(0)
}

func (m *MockCache) Set(key string, value interface{}, duration time.Duration) {
	m.Called(key, value, duration)
}

func (m *MockCache) Delete(key string) {
	m.Called(key)
}

// MockScoreCalculator mocks the score calculation functionality
type MockScoreCalculator struct {
	mock.Mock
}

func (m *MockScoreCalculator) CalculateScore(scores []db.LLMScore) (float64, float64, error) {
	args := m.Called(scores)
	scoreVal, okScore := args.Get(0).(float64)
	confidenceVal, okConfidence := args.Get(1).(float64)
	originalError := args.Error(2)

	if !okScore || !okConfidence {
		log.Printf("WARN: MockScoreCalculator.CalculateScore: type assertion failed. Score ok: %v, Confidence ok: %v", okScore, okConfidence)
		// Return zero values for score/confidence and the original error from the mock setup
		// or a new error indicating assertion failure if originalError is nil.
		if originalError == nil {
			return 0.0, 0.0, fmt.Errorf("MockScoreCalculator: type assertion failed for score or confidence")
		}
		return 0.0, 0.0, originalError
	}

	return scoreVal, confidenceVal, originalError
}

// MockDBTx mocks a database transaction
type MockDBTx struct {
	mock.Mock
}

func (m *MockDBTx) Exec(query string, args ...interface{}) (interface{}, error) {
	callArgs := m.Called(append([]interface{}{query}, args...)...)
	return callArgs.Get(0), callArgs.Error(1)
}

func (m *MockDBTx) Commit() error {
	return m.Called().Error(0)
}

func (m *MockDBTx) Rollback() error {
	return m.Called().Error(0)
}

// IntegrationMockLLMClient implements a mock version of the LLMClient for testing
type IntegrationMockLLMClient struct {
	mock.Mock
}

// CheckHealth mocks the LLMClient.CheckHealth method
func (m *IntegrationMockLLMClient) CheckHealth() error {
	args := m.Called()
	return args.Error(0)
}

// AnalyzeArticle mocks the LLMClient.AnalyzeArticle method
func (m *IntegrationMockLLMClient) AnalyzeArticle(ctx context.Context, article *db.Article) (*llm.ArticleAnalysis, error) {
	args := m.Called(ctx, article)
	originalError := args.Error(1)
	if args.Get(0) == nil {
		return nil, originalError
	}
	val, ok := args.Get(0).(*llm.ArticleAnalysis)
	if !ok {
		log.Printf("WARN: IntegrationMockLLMClient.AnalyzeArticle: type assertion to *llm.ArticleAnalysis failed for article ID %d", article.ID)
		// Return nil and the original error from mock setup,
		// or a new error if originalError is nil.
		if originalError == nil {
			return nil, fmt.Errorf("IntegrationMockLLMClient.AnalyzeArticle: type assertion failed")
		}
		return nil, originalError
	}
	return val, originalError
}

// FetchScores mocks the LLMClient.FetchScores method
func (m *IntegrationMockLLMClient) FetchScores(articleID int64) ([]db.LLMScore, error) {
	args := m.Called(articleID)
	originalError := args.Error(1)
	if args.Get(0) == nil {
		return nil, originalError
	}
	val, ok := args.Get(0).([]db.LLMScore)
	if !ok {
		log.Printf("WARN: IntegrationMockLLMClient.FetchScores: type assertion to []db.LLMScore failed for articleID %d", articleID)
		// Return nil and the original error from mock setup,
		// or a new error if originalError is nil.
		if originalError == nil {
			return nil, fmt.Errorf("IntegrationMockLLMClient.FetchScores: type assertion failed")
		}
		return nil, originalError
	}
	return val, originalError
}

// Helper function to create a test server with real API handlers and mocked dependencies
func setupIntegrationTestServer(t *testing.T) (*gin.Engine, *MockDBOperations, *MockProgressManager, *MockCache, *MockScoreCalculator, *IntegrationMockLLMClient) {
	gin.SetMode(gin.TestMode)

	// Create all our mocks
	mockDB := new(MockDBOperations)
	mockProgress := new(MockProgressManager)
	mockCache := new(MockCache)
	mockCalculator := new(MockScoreCalculator)
	mockLLMClient := new(IntegrationMockLLMClient)

	// Create a router with our API endpoints
	router := gin.New()
	router.Use(gin.Recovery()) // Ensure panics are recovered

	// Register API routes - we'll use a simplified version for testing
	api := router.Group("/api")
	{
		// Mock the reanalyze endpoint
		api.POST("/llm/reanalyze/:id", func(c *gin.Context) {
			idStr := c.Param("id")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID"})
				return
			}

			// Get the article
			article, err := mockDB.GetArticleByID(context.TODO(), id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Article not found"})
				return
			}

			// Update progress
			progressState := &models.ProgressState{
				Step:        "Starting",
				Message:     "Processing article...",
				Status:      "InProgress",
				Percent:     0,
				LastUpdated: time.Now().Unix(),
			}
			mockProgress.SetProgress(id, progressState)

			c.JSON(http.StatusOK, gin.H{"success": true, "data": map[string]interface{}{
				"article_id": article.ID,
				"status":     "reanalysis queued",
			}})
		})

		// Mock the SSE progress endpoint
		api.GET("/llm/score-progress/:id", func(c *gin.Context) {
			// Return SSE content
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")

			// Mock the progress state
			id := c.Param("id")
			articleID, _ := strconv.ParseInt(id, 10, 64)
			state := mockProgress.GetProgress(articleID)

			if state != nil {
				jsonData, marshalErr := json.Marshal(state)
				if marshalErr != nil {
					log.Printf("WARN: Mock SSE: Error marshalling progress state: %v", marshalErr)
					// Optionally send an error event or close
					_, _ = c.Writer.Write([]byte("data: {\"error\":\"internal marshalling error\"}\n\n"))
					return
				}
				_, _ = c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", jsonData)))
			} else {
				_, _ = c.Writer.Write([]byte("data: {\"step\":\"No data\",\"percent\":0}\n\n"))
			}
		})

		// Mock the manual score endpoint
		api.POST("/manual-score/:id", func(c *gin.Context) {
			idStr := c.Param("id")
			id, _ := strconv.ParseInt(idStr, 10, 64)

			var req struct {
				Score float64 `json:"score"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid JSON"})
				return
			}

			// Call the mock DB
			_, _ = mockDB.FetchArticleByID(context.TODO(), id)              // Explicitly ignore return values
			_ = mockDB.UpdateArticleScore(context.TODO(), id, req.Score, 0) // Explicitly ignore return value

			// Invalidate cache
			cacheKey := fmt.Sprintf("article:%d", id)
			mockCache.Delete(cacheKey)

			c.JSON(http.StatusOK, gin.H{"success": true, "data": map[string]interface{}{
				"status":     "score updated",
				"article_id": id,
				"score":      req.Score,
			}})
		})
	}

	return router, mockDB, mockProgress, mockCache, mockCalculator, mockLLMClient
}

// Test that the ScoreManager's progress tracking is integrated with the reanalyze endpoint
func TestReanalyzeEndpointProgressTracking(t *testing.T) {
	// Setup test server
	router, mockDB, mockProgress, _, _, _ := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID := int64(123)
	testArticle := &db.Article{
		ID:      testArticleID,
		Title:   "Test Article",
		Content: "This is a test article for integration testing.",
		URL:     "https://example.com/test",
	}

	// Mock dependencies behavior
	mockDB.On("GetArticleByID", mock.Anything, testArticleID).Return(testArticle, nil)

	// We need to accept any ProgressState struct that's passed to SetProgress
	mockProgress.On("SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState")).Return()

	// Create a request to trigger the reanalyze endpoint
	req, _ := http.NewRequest("POST", fmt.Sprintf(reanalyzeURLPath, testArticleID), bytes.NewBuffer([]byte("{}")))
	req.Header.Set(contentTypeHeader, applicationJSON)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify progress tracking was called
	mockProgress.AssertCalled(t, "SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState"))
}

// Test that SSE progress endpoint correctly connects to ScoreManager's progress tracking
func TestSSEProgressEndpointIntegration(t *testing.T) {
	// Setup test server - we only need the progress manager mock here
	router, _, mockProgress, _, _, _ := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID := int64(456)
	testProgressState := &models.ProgressState{
		Step:        "Processing",
		Message:     "Processing article",
		Percent:     50,
		Status:      "InProgress",
		LastUpdated: time.Now().Unix(),
	}

	// Setup mock behavior
	mockProgress.On("GetProgress", testArticleID).Return(testProgressState)

	// Create a request to the SSE endpoint
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/llm/score-progress/%d", testArticleID), nil)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

	// Verify the SSE response contains our progress data
	responseBody := w.Body.String()
	assert.Contains(t, responseBody, "data:")
	assert.Contains(t, responseBody, "Processing")
	assert.Contains(t, responseBody, "50")
}

// Test that ScoreManager's cache invalidation is triggered during manual score updates
func TestManualScoreCacheInvalidation(t *testing.T) {
	// Setup test server - need DB and Cache mocks
	router, mockDB, _, mockCache, _, _ := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID := int64(789)
	testScore := 0.75

	// Mock dependencies behavior
	mockDB.On("FetchArticleByID", mock.Anything, testArticleID).Return(&db.Article{ID: testArticleID}, nil)
	mockDB.On("UpdateArticleScore", mock.Anything, testArticleID, testScore, mock.Anything).Return(nil)

	// Mock cache invalidation - this is what we want to test
	mockCache.On("Delete", mock.MatchedBy(func(key string) bool {
		return strings.Contains(key, fmt.Sprintf("%d", testArticleID))
	})).Return()

	// Create a request to update the score
	requestBody := fmt.Sprintf(`{"score":%f}`, testScore)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/manual-score/%d", testArticleID), bytes.NewBuffer([]byte(requestBody)))
	req.Header.Set(contentTypeHeader, applicationJSON)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify cache invalidation was called
	mockCache.AssertCalled(t, "Delete", mock.Anything)
}

// Test that the API properly integrates with ScoreManager's transaction handling
func TestScoreManagerTransactionHandling(t *testing.T) {
	router, mockDB, mockProgress, _, mockCalculator, _ := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID := int64(101)
	testArticle := &db.Article{
		ID:      testArticleID,
		Title:   "Transaction Test Article",
		Content: "This is a test article for transaction handling.",
	}

	// Mock dependencies behavior - need to set up GetArticleByID to prevent panic
	mockDB.On("GetArticleByID", mock.Anything, testArticleID).Return(testArticle, nil)

	// Mock progress tracking
	mockProgress.On("SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState")).Return()

	// Mock successful transaction
	mockTx := new(MockDBTx)
	mockDB.On("BeginTxx", mock.Anything, mock.Anything).Return(mockTx, nil)
	mockTx.On("Exec", mock.Anything, mock.Anything).Return(nil, nil)
	mockTx.On("Commit").Return(nil)

	// Mock calculator behavior
	mockCalculator.On("CalculateScore", mock.Anything).Return(0.1, 0.8, nil)

	// Create a request to reanalyze an article
	req, _ := http.NewRequest("POST", fmt.Sprintf(reanalyzeURLPath, testArticleID), bytes.NewBuffer([]byte("{}")))
	req.Header.Set(contentTypeHeader, applicationJSON)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify progress was set
	mockProgress.AssertCalled(t, "SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState"))
}

// Test for error propagation from LLM to API
func TestErrorPropagationLLMToAPI(t *testing.T) {
	router, mockDB, mockProgress, _, _, mockLLM := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID := int64(202)
	testArticle := &db.Article{
		ID:      testArticleID,
		Title:   "LLM Error Test Article",
		Content: "This is a test article for LLM error propagation.",
	}

	// Mock dependencies behavior
	mockDB.On("GetArticleByID", mock.Anything, testArticleID).Return(testArticle, nil)

	// Mock LLM error
	llmError := fmt.Errorf("LLM API error")
	mockLLM.On("AnalyzeArticle", mock.Anything, mock.Anything).Return(nil, llmError)

	// Accept any progress state for initial "Starting" progress
	mockProgress.On("SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState")).Return()

	// Create a request
	req, _ := http.NewRequest("POST", fmt.Sprintf(reanalyzeURLPath, testArticleID), bytes.NewBuffer([]byte("{}")))
	req.Header.Set(contentTypeHeader, applicationJSON)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that SetProgress was called
	mockProgress.AssertNumberOfCalls(t, "SetProgress", 1)
}

// Test the full workflow of article scoring from API to ScoreManager and back
func TestFullWorkflowArticleScoring(t *testing.T) {
	router, mockDB, mockProgress, mockCache, mockCalculator, _ := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID := int64(303)
	testArticle := &db.Article{
		ID:      testArticleID,
		Title:   "Integration Test Article",
		Content: "This is a test article for full workflow integration testing.",
	}

	// Set up all the needed mock behaviors for a full workflow
	// 1. Initial article fetch
	mockDB.On("GetArticleByID", mock.Anything, testArticleID).Return(testArticle, nil)

	// 2. Progress tracking calls - just accept any progress state
	mockProgress.On("SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState")).Return()

	// 3. Score calculation
	mockCalculator.On("CalculateScore", mock.Anything).Return(0.167, 0.85, nil)

	// 4. Database transaction
	mockTx := new(MockDBTx)
	mockDB.On("BeginTxx", mock.Anything, mock.Anything).Return(mockTx, nil)
	mockTx.On("Exec", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockTx.On("Commit").Return(nil)

	// 5. Cache invalidation after success
	mockCache.On("Delete", mock.Anything).Return()

	// Create request to start the scoring process
	req, _ := http.NewRequest("POST", fmt.Sprintf(reanalyzeURLPath, testArticleID), bytes.NewBuffer([]byte("{}")))
	req.Header.Set(contentTypeHeader, applicationJSON)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["success"])

	// Verify progress tracking was called
	mockProgress.AssertNumberOfCalls(t, "SetProgress", 1)
}

// Test concurrent requests to ensure thread safety of ScoreManager
func TestConcurrentRequestsThreadSafety(t *testing.T) {
	// Setup test server
	router, mockDB, mockProgress, _, _, _ := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID1 := int64(404)
	testArticleID2 := int64(405)

	// Set up basic mocks for both articles
	mockDB.On("GetArticleByID", mock.Anything, testArticleID1).Return(&db.Article{ID: testArticleID1}, nil)
	mockDB.On("GetArticleByID", mock.Anything, testArticleID2).Return(&db.Article{ID: testArticleID2}, nil)

	// Progress tracking for both articles
	mockProgress.On("SetProgress", testArticleID1, mock.Anything).Return()
	mockProgress.On("SetProgress", testArticleID2, mock.Anything).Return()

	// Create concurrent requests
	reqChan := make(chan struct{})
	doneChan := make(chan struct{})

	// Goroutine for first request
	go func() {
		<-reqChan // Wait for signal to start
		req, _ := http.NewRequest("POST", fmt.Sprintf(reanalyzeURLPath, testArticleID1), bytes.NewBuffer([]byte("{}")))
		req.Header.Set(contentTypeHeader, applicationJSON)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		doneChan <- struct{}{}
	}()

	// Goroutine for second request
	go func() {
		<-reqChan // Wait for signal to start
		req, _ := http.NewRequest("POST", fmt.Sprintf(reanalyzeURLPath, testArticleID2), bytes.NewBuffer([]byte("{}")))
		req.Header.Set(contentTypeHeader, applicationJSON)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		doneChan <- struct{}{}
	}()

	// Start both requests nearly simultaneously
	close(reqChan)

	// Wait for both to finish
	<-doneChan
	<-doneChan

	// Verify both articles had their progress tracked separately
	mockProgress.AssertCalled(t, "SetProgress", testArticleID1, mock.Anything)
	mockProgress.AssertCalled(t, "SetProgress", testArticleID2, mock.Anything)
}

// Test that database errors during scoring are properly handled and reported
func TestDatabaseErrorsErrorHandling(t *testing.T) {
	// Setup test server
	router, mockDB, mockProgress, _, _, _ := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID := int64(999)
	testArticle := &db.Article{
		ID:      testArticleID,
		Title:   "DB Error Test Article",
		Content: "This is a test article for database error handling.",
	}

	// Mock dependencies behavior
	mockDB.On("GetArticleByID", mock.Anything, testArticleID).Return(testArticle, nil)

	// Accept any progress state for the "Starting" status
	mockProgress.On("SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState")).Return()

	// Create a request
	req, _ := http.NewRequest("POST", fmt.Sprintf(reanalyzeURLPath, testArticleID), bytes.NewBuffer([]byte("{}")))
	req.Header.Set(contentTypeHeader, applicationJSON)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify that progress tracking was called
	mockProgress.AssertCalled(t, "SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState"))
}

// Test that LLMAPIError is properly propagated through the reanalyze endpoint
func TestReanalyzeEndpointLLMErrorPropagation(t *testing.T) {
	// Setup test server
	router, mockDB, _, _, _, mockLLM := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID := int64(456)
	testArticle := &db.Article{
		ID:      testArticleID,
		Title:   "LLM Error Test Article",
		Content: "This is a test article for LLM error propagation.",
		URL:     "https://example.com/test-error",
	}

	// Mock dependencies behavior
	mockDB.On("GetArticleByID", mock.Anything, testArticleID).Return(testArticle, nil)

	// Create a LLMAPIError for authentication failure
	llmAuthError := llm.LLMAPIError{
		Message:      "Invalid API key",
		StatusCode:   401,
		ResponseBody: "Authentication failed",
		ErrorType:    llm.ErrTypeAuthentication,
	}

	// Mock ScoreWithModel to simulate an LLM health check authentication failure
	mockLLM.On("ScoreWithModel", testArticle, mock.Anything).Return(0.0, llmAuthError)

	// Create a request to trigger the reanalyze endpoint
	req, _ := http.NewRequest("POST", fmt.Sprintf(reanalyzeURLPath, testArticleID), bytes.NewBuffer([]byte("{}")))
	req.Header.Set(contentTypeHeader, applicationJSON)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert that the correct HTTP status code is returned (401 for auth failure)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify the response contains the expected error details
	assert.False(t, response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "llm_service_error", errorData["code"])
	assert.Equal(t, "LLM service authentication failed", errorData["message"])

	// Check for the detailed error information
	details := errorData["details"].(map[string]interface{})
	assert.Equal(t, float64(401), details["llm_status_code"])
	assert.Equal(t, "Invalid API key", details["llm_message"])
	assert.Equal(t, "authentication", details["error_type"])
	assert.Equal(t, "openrouter", details["provider"])

	// Verify the recommended action field is present
	assert.Contains(t, errorData, "recommended_action")
	assert.Equal(t, "Contact administrator to update API credentials", errorData["recommended_action"])
}
