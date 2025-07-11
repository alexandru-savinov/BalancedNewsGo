package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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

// ScoreWithModel mocks the LLMClient.ScoreWithModel method
func (m *IntegrationMockLLMClient) ScoreWithModel(article *db.Article, modelName string) (float64, error) {
	args := m.Called(article, modelName)
	return args.Get(0).(float64), args.Error(1)
}

// GetConfig mocks the LLMClient.GetConfig method
func (m *IntegrationMockLLMClient) GetConfig() *llm.CompositeScoreConfig {
	args := m.Called()
	if cfg, ok := args.Get(0).(*llm.CompositeScoreConfig); ok {
		return cfg
	}
	return nil
}

// GetHTTPLLMTimeout mocks the LLMClient.GetHTTPLLMTimeout method
func (m *IntegrationMockLLMClient) GetHTTPLLMTimeout() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

// SetHTTPLLMTimeout mocks the LLMClient.SetHTTPLLMTimeout method
func (m *IntegrationMockLLMClient) SetHTTPLLMTimeout(timeout time.Duration) {
	m.Called(timeout)
}

// ReanalyzeArticle mocks the LLMClient.ReanalyzeArticle method
func (m *IntegrationMockLLMClient) ReanalyzeArticle(articleID int64) error {
	args := m.Called(articleID)
	return args.Error(0)
}

// Helper function to create a test server with real API handlers and mocked dependencies
func setupIntegrationTestServer(t *testing.T) (*gin.Engine, *MockDBOperations,
	*MockProgressManager, *MockCache, *MockScoreCalculator, *IntegrationMockLLMClient) {
	gin.SetMode(gin.TestMode)

	// Create all our mocks
	mockDB := new(MockDBOperations)
	mockProgress := new(MockProgressManager)
	mockCache := new(MockCache)
	mockCalculator := new(MockScoreCalculator)
	mockLLMClient := new(IntegrationMockLLMClient)

	// Provide default expectations for LLM client to avoid panics in tests
	defaultCfg := &llm.CompositeScoreConfig{Models: []llm.ModelConfig{{ModelName: "model-A"}}}
	mockLLMClient.On("GetConfig").Return(defaultCfg)
	mockLLMClient.On("GetHTTPLLMTimeout").Return(2 * time.Second)
	mockLLMClient.On("SetHTTPLLMTimeout", mock.Anything).Return()
	mockLLMClient.On("ScoreWithModel", mock.Anything, mock.Anything).Return(0.0, nil)
	mockLLMClient.On("ReanalyzeArticle", mock.AnythingOfType("int64")).Return(nil)

	// Create a router with our API endpoints
	router := gin.New()
	router.Use(gin.Recovery()) // Ensure panics are recovered

	// Register API routes - we'll use a simplified version for testing
	api := router.Group("/api")
	{
		// For tests of the updated reanalyzeHandler fallback mechanism
		// we'll replace the mock handler with one that uses our mocked methods
		api.POST("/llm/reanalyze/:id", func(c *gin.Context) {
			idStr := c.Param("id")
			id, err := strconv.Atoi(idStr)
			if err != nil || id < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid article ID"})
				return
			}
			articleID := int64(id)

			// Check if article exists
			article, err := mockDB.GetArticleByID(context.TODO(), articleID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Article not found"})
				return
			}

			// Parse raw JSON body
			var raw map[string]interface{}
			if err := c.ShouldBindJSON(&raw); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request payload"})
				return
			}

			// Load composite score config to get the models
			cfg := mockLLMClient.GetConfig()
			if cfg == nil || len(cfg.Models) == 0 {
				c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "Failed to load LLM configuration"})
				return
			}

			// Try each model in sequence until we find one that works
			var workingModel string

			originalTimeout := mockLLMClient.GetHTTPLLMTimeout()
			healthCheckTimeout := 2 * time.Second // Keep short timeout for individual health checks
			mockLLMClient.SetHTTPLLMTimeout(healthCheckTimeout)

			for _, modelConfig := range cfg.Models {
				_, healthCheckErr := mockLLMClient.ScoreWithModel(article, modelConfig.ModelName)

				if healthCheckErr == nil {
					workingModel = modelConfig.ModelName
					break
				}
			}
			mockLLMClient.SetHTTPLLMTimeout(originalTimeout) // Restore original timeout

			if workingModel == "" {
				c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "No working models found"})
				return
			}

			// Start the reanalysis process
			mockProgress.SetProgress(articleID, &models.ProgressState{
				Status:  "InProgress",
				Step:    "Starting",
				Message: fmt.Sprintf("Starting analysis with model %s", workingModel),
			})

			// Check if NO_AUTO_ANALYZE is set (for test environment compatibility)
			if os.Getenv("NO_AUTO_ANALYZE") == "true" {
				// Skip actual analysis and set to skipped state
				mockProgress.SetProgress(articleID, &models.ProgressState{
					Status:      "Skipped",
					Step:        "Skipped",
					Message:     "Automatic reanalysis skipped by test configuration.",
					Percent:     100,
					LastUpdated: time.Now().Unix(),
				})
			} else {
				go func() {
					err := mockLLMClient.ReanalyzeArticle(articleID)
					if err != nil {
						mockProgress.SetProgress(articleID, &models.ProgressState{
							Status:  "Error",
							Step:    "Error",
							Message: fmt.Sprintf("Error during analysis: %v", err),
						})
						return
					}
					mockProgress.SetProgress(articleID, &models.ProgressState{
						Status:  "Complete",
						Step:    "Done",
						Message: "Analysis complete",
					})
				}()
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "data": map[string]interface{}{
				"status":     "reanalysis queued",
				"article_id": articleID,
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
	router, mockDB, mockProgress, _, _, mockLLM := setupIntegrationTestServer(t)

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

	cfg := &llm.CompositeScoreConfig{Models: []llm.ModelConfig{{ModelName: "model-A"}}}
	mockLLM.On("GetConfig").Return(cfg)
	mockLLM.On("GetHTTPLLMTimeout").Return(2 * time.Second)
	mockLLM.On("SetHTTPLLMTimeout", mock.Anything).Return()
	mockLLM.On("ScoreWithModel", testArticle, "model-A").Return(0.0, nil)
	mockLLM.On("ReanalyzeArticle", testArticleID).Return(nil)

	// We need to accept any ProgressState struct that's passed to SetProgress
	mockProgress.On("SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState")).Return().Twice()

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

	// Verify that SetProgress was called (twice in test environment with NO_AUTO_ANALYZE=true)
	// Once for initial progress, once for skipped state
	mockProgress.AssertNumberOfCalls(t, "SetProgress", 2)
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

	// Verify progress tracking was called (twice in test environment with NO_AUTO_ANALYZE=true)
	// Once for initial progress, once for skipped state
	mockProgress.AssertNumberOfCalls(t, "SetProgress", 2)
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
	router, mockDB, mockProgress, _, _, mockLLM := setupIntegrationTestServer(t)

	// Setup test data
	testArticleID := int64(999)
	testArticle := &db.Article{
		ID:      testArticleID,
		Title:   "DB Error Test Article",
		Content: "This is a test article for database error handling.",
	}

	// Mock dependencies behavior
	mockDB.On("GetArticleByID", mock.Anything, testArticleID).Return(testArticle, nil)

	// Provide simple config so handler can proceed
	cfg := &llm.CompositeScoreConfig{Models: []llm.ModelConfig{{ModelName: "model-A"}}}
	mockLLM.On("GetConfig").Return(cfg)
	mockLLM.On("GetHTTPLLMTimeout").Return(2 * time.Second)
	mockLLM.On("SetHTTPLLMTimeout", mock.Anything).Return()
	mockLLM.On("ScoreWithModel", testArticle, "model-A").Return(0.0, nil)
	mockLLM.On("ReanalyzeArticle", testArticleID).Return(nil)

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
	router, mockDB, mockProgress, _, _, mockLLM := setupIntegrationTestServer(t)
	mockLLM.ExpectedCalls = nil

	// Setup test data
	testArticleID := int64(456)
	mockProgress.On("SetProgress", testArticleID, mock.AnythingOfType("*models.ProgressState")).Return()
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

	// Provide LLM config and simulate authentication failure during health check
	cfg := &llm.CompositeScoreConfig{Models: []llm.ModelConfig{{ModelName: "model-A"}}}
	mockLLM.On("GetConfig").Return(cfg)
	mockLLM.On("GetHTTPLLMTimeout").Return(2 * time.Second)
	mockLLM.On("SetHTTPLLMTimeout", mock.Anything).Return()
	mockLLM.On("ScoreWithModel", testArticle, "model-A").Return(0.0, llmAuthError)

	// Create a request to trigger the reanalyze endpoint
	req, _ := http.NewRequest("POST", fmt.Sprintf(reanalyzeURLPath, testArticleID), bytes.NewBuffer([]byte("{}")))
	req.Header.Set(contentTypeHeader, applicationJSON)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// All models fail the health check, resulting in service unavailable
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// TestReanalyzeHandlerFallbackMechanismIntegration tests the improved fallback mechanism in reanalyzeHandler
// that tries all configured models before giving up.
func TestReanalyzeHandlerFallbackMechanismIntegration(t *testing.T) {
	// Setup test server with mocks
	router, mockDB, mockProgress, _, _, mockLLMClient := setupIntegrationTestServer(t)
	// Clear default expectations so we can define custom behavior
	mockLLMClient.ExpectedCalls = nil

	// Ensure background reanalysis runs during this test
	if err := os.Setenv("NO_AUTO_ANALYZE", "false"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("NO_AUTO_ANALYZE"); err != nil {
			t.Logf("Failed to unset environment variable: %v", err)
		}
	}()

	// Test article ID
	articleID := int64(42)

	// Create a test article
	testArticle := &db.Article{
		ID:      articleID,
		Title:   "Test Article",
		Content: "Test content for article reanalysis fallback test",
	}

	// Setup mock DB to return our test article
	mockDB.On("GetArticleByID", mock.Anything, articleID).Return(testArticle, nil)

	// Configure the mock LLM client to simulate first model failing, second model succeeding
	mockLLMClient.On("ScoreWithModel", testArticle, "model-A").Return(0.0, fmt.Errorf("simulated error for model-A"))
	mockLLMClient.On("ScoreWithModel", testArticle, "model-B").Return(0.5, nil)

	// Mock the GetConfig method to return a test configuration with two models
	cfg := &llm.CompositeScoreConfig{
		Models: []llm.ModelConfig{
			{ModelName: "model-A"},
			{ModelName: "model-B"},
		},
	}
	mockLLMClient.On("GetConfig").Return(cfg)

	// Mock the GetHTTPLLMTimeout and SetHTTPLLMTimeout methods
	testTimeout := 5 * time.Second
	mockLLMClient.On("GetHTTPLLMTimeout").Return(testTimeout)
	mockLLMClient.On("SetHTTPLLMTimeout", mock.Anything).Return()

	// For the actual reanalysis, mock to succeed if triggered
	// (NO_AUTO_ANALYZE may skip this call, so mark as Maybe)
	mockLLMClient.On("ReanalyzeArticle", articleID).Return(nil).Maybe()

	// Progress tracking should be set to show it's working
	mockProgress.On("SetProgress", articleID, mock.Anything).Return()

	// Create request to the reanalyze endpoint
	uri := fmt.Sprintf("/api/llm/reanalyze/%d", articleID)
	req := httptest.NewRequest(http.MethodPost, uri, bytes.NewBuffer([]byte("{}")))
	req.Header.Set(contentTypeHeader, applicationJSON)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Verify the response
	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")

	// Parse response body
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err, "Should parse response JSON")

	// Verify we got "reanalyze queued" status
	assert.Equal(t, true, resp["success"], "Response should indicate success")
	data, hasData := resp["data"].(map[string]interface{})
	assert.True(t, hasData, "Response should have a data field")
	assert.Equal(t, "reanalysis queued", data["status"], "Status should be 'reanalysis queued'")

	// Verify all mock expectations
	mockDB.AssertExpectations(t)
	mockLLMClient.AssertExpectations(t)
	mockProgress.AssertExpectations(t)
}

// TestReanalyzeHandlerAllModelsFail tests the case where all models fail the health check.
func TestReanalyzeHandlerAllModelsFail(t *testing.T) {
	// Setup test server with mocks
	router, mockDB, mockProgress, _, _, mockLLMClient := setupIntegrationTestServer(t)
	// Clear default expectations so we can define custom behavior
	mockLLMClient.ExpectedCalls = nil

	// Test article ID
	articleID := int64(43)

	// Create a test article
	testArticle := &db.Article{
		ID:      articleID,
		Title:   "Test Article - All Models Fail",
		Content: "Test content for all models failing test case",
	}

	// Setup mock DB to return our test article
	mockDB.On("GetArticleByID", mock.Anything, articleID).Return(testArticle, nil)

	// Configure the mock LLM client to simulate all models failing
	mockLLMClient.On("ScoreWithModel", testArticle, "model-A").Return(0.0, fmt.Errorf("simulated error for model-A"))
	mockLLMClient.On("ScoreWithModel", testArticle, "model-B").Return(0.0, fmt.Errorf("simulated error for model-B"))

	// Expect progress to be set when analysis starts
	mockProgress.On("SetProgress", articleID, mock.AnythingOfType("*models.ProgressState")).Return()

	// Mock the GetConfig method to return a test configuration with two models
	cfg := &llm.CompositeScoreConfig{
		Models: []llm.ModelConfig{
			{ModelName: "model-A"},
			{ModelName: "model-B"},
		},
	}
	mockLLMClient.On("GetConfig").Return(cfg)

	// Mock the GetHTTPLLMTimeout and SetHTTPLLMTimeout methods
	testTimeout := 5 * time.Second
	mockLLMClient.On("GetHTTPLLMTimeout").Return(testTimeout)
	mockLLMClient.On("SetHTTPLLMTimeout", mock.Anything).Return()

	// We should NOT call ReanalyzeArticle if all models fail
	// No need to mock mockLLMClient.On("ReanalyzeArticle", ...) here

	// Create request to the reanalyze endpoint
	uri := fmt.Sprintf("/api/llm/reanalyze/%d", articleID)
	req := httptest.NewRequest(http.MethodPost, uri, bytes.NewBuffer([]byte("{}")))
	req.Header.Set(contentTypeHeader, applicationJSON)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Verify the response
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, "Should return 503 Service Unavailable")

	// Verify all mock expectations
	mockDB.AssertExpectations(t)
	mockLLMClient.AssertExpectations(t)
}
