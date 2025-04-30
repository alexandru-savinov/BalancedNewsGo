package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/rss"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock RSS Collector
type MockRSSCollector struct {
	mock.Mock
}

func (m *MockRSSCollector) ManualRefresh() {
	m.Called()
}

func (m *MockRSSCollector) CheckFeedHealth() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// TestRegisterRoutes tests that all routes are registered correctly
func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Create necessary mocks
	dbConn := &sqlx.DB{} // Empty DB connection for test
	mockRSS := new(rss.Collector)
	mockLLM := new(llm.LLMClient)
	mockScoreManager := new(llm.ScoreManager)
	
	// Register routes
	RegisterRoutes(router, dbConn, mockRSS, mockLLM, mockScoreManager)
	
	// Test that key routes exist
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/articles"},
		{"GET", "/api/articles/:id"},
		{"POST", "/api/articles"},
		{"POST", "/api/refresh"},
		{"POST", "/api/llm/reanalyze/:id"},
		{"POST", "/api/manual-score/:id"},
		{"GET", "/api/articles/:id/summary"},
		{"GET", "/api/articles/:id/bias"},
		{"GET", "/api/articles/:id/ensemble"},
		{"POST", "/api/feedback"},
		{"GET", "/api/feeds/healthz"},
		{"GET", "/api/llm/score-progress/:id"},
	}
	
	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			// Check that route exists
			found := false
			for _, r := range router.Routes() {
				if r.Method == route.method && r.Path == route.path {
					found = true
					break
				}
			}
			assert.True(t, found, "Route not found")
		})
	}
}

// TestSafeHandler tests that the SafeHandler correctly recovers from panics
func TestSafeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Create a handler that will panic
	router.GET("/panic", SafeHandler(func(c *gin.Context) {
		panic("test panic")
	}))
	
	// Test panic recovery
	req, _ := http.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should return 500 status code
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// Response should contain error
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"].(string), "Internal server error")
}

// TestRefreshHandlerFunc tests the refresh handler
func TestRefreshHandlerFunc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	mockRSS := &MockRSSCollector{}
	mockRSS.On("ManualRefresh").Return()
	
	router.POST("/api/refresh", SafeHandler(refreshHandler(mockRSS)))
	
	// Test refresh handler
	req, _ := http.NewRequest("POST", "/api/refresh", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should return 200 status code
	assert.Equal(t, http.StatusOK, w.Code)
	// Response should contain success
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "refresh started", response["data"].(map[string]interface{})["status"])
	
	// Verify that ManualRefresh was called
	mockRSS.AssertCalled(t, "ManualRefresh")
}

// TestFeedHealthHandlerFunc tests the feed health handler
func TestFeedHealthHandlerFunc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	mockRSS := &MockRSSCollector{}
	healthData := map[string]interface{}{
		"status": "healthy",
		"feeds": map[string]bool{
			"feed1": true,
			"feed2": false,
		},
	}
	mockRSS.On("CheckFeedHealth").Return(healthData)
	
	router.GET("/api/feeds/healthz", SafeHandler(feedHealthHandler(mockRSS)))
	
	// Test feed health handler
	req, _ := http.NewRequest("GET", "/api/feeds/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should return 200 status code
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Response should contain health data
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	// Verify response structure
	assert.Equal(t, "healthy", response["status"])
	feeds, ok := response["feeds"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, feeds["feed1"])
	assert.Equal(t, false, feeds["feed2"])
	
	// Verify that CheckFeedHealth was called
	mockRSS.AssertCalled(t, "CheckFeedHealth")
}

// TestSetProgressAndGetProgress tests the setProgress and getProgress functions
func TestSetProgressAndGetProgress(t *testing.T) {
	// Setup test data
	articleID := int64(123)
	state := &models.ProgressState{
		Status: "InProgress",
		Step: "Testing",
		Message: "Running tests",
		Percent: 50,
	}
	
	// Test setProgress
	setProgress(articleID, state)
	
	// Test getProgress
	result := getProgress(articleID)
	
	// Verify results
	assert.Equal(t, state.Status, result.Status)
	assert.Equal(t, state.Step, result.Step)
	assert.Equal(t, state.Message, result.Message)
	assert.Equal(t, state.Percent, result.Percent)
	
	// Test with non-existent article
	nonExistentID := int64(456)
	nullResult := getProgress(nonExistentID)
	assert.Nil(t, nullResult)
}

// TestScoreProgressSSEHandler tests the SSE progress handler
func TestScoreProgressSSEHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	router.GET("/api/llm/score-progress/:id", SafeHandler(scoreProgressSSEHandler()))
	
	// Setup test data
	articleID := int64(789)
	state := &models.ProgressState{
		Status: "Success",
		Step: "Complete",
		Message: "Scoring complete",
		Percent: 100,
	}
	setProgress(articleID, state)
	
	// Test SSE handler
	req, _ := http.NewRequest("GET", "/api/llm/score-progress/789", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Verify headers
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
	
	// Should contain SSE data format
	assert.Contains(t, w.Body.String(), "data: ")
	
	// Test with invalid ID
	req, _ = http.NewRequest("GET", "/api/llm/score-progress/invalid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should return 400 status code
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestPercent tests the percent function
func TestPercent(t *testing.T) {
	tests := []struct {
		step     int
		total    int
		expected int
	}{
		{1, 4, 25},
		{2, 4, 50},
		{3, 4, 75},
		{4, 4, 100},
		{5, 4, 100}, // Should be capped at 100
		{1, 0, 0},   // Avoid division by zero
	}
	
	for _, test := range tests {
		result := percent(test.step, test.total)
		assert.Equal(t, test.expected, result)
	}
}

// TestArticleToPostmanSchema tests the articleToPostmanSchema function
func TestArticleToPostmanSchema(t *testing.T) {
	// Create test article
	score := 0.5
	confidence := 0.8
	article := &db.Article{
		ID:             123,
		Title:          "Test Article",
		Content:        "Test Content",
		URL:            "http://test.com",
		Source:         "Test Source",
		CompositeScore: &score,
		Confidence:     &confidence,
	}
	
	// Convert to schema
	result := articleToPostmanSchema(article)
	
	// Verify result
	assert.Equal(t, int64(123), result["article_id"])
	assert.Equal(t, "Test Article", result["Title"])
	assert.Equal(t, "Test Content", result["Content"])
	assert.Equal(t, "http://test.com", result["URL"])
	assert.Equal(t, "Test Source", result["Source"])
	assert.Equal(t, score, result["CompositeScore"])
	assert.Equal(t, confidence, result["Confidence"])
}

// TestStrPtr tests the strPtr helper function
func TestStrPtr(t *testing.T) {
	s := "test"
	ptr := strPtr(s)
	
	assert.NotNil(t, ptr)
	assert.Equal(t, s, *ptr)
}