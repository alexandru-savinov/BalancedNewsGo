package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	ginTestModeOnceMissing sync.Once
)

// MockRSSCollector for testing RSS-related handlers
type MockRSSCollectorForMissing struct {
	mock.Mock
}

func (m *MockRSSCollectorForMissing) ManualRefresh() {
	m.Called()
}

func (m *MockRSSCollectorForMissing) CheckFeedHealth() map[string]bool {
	args := m.Called()
	return args.Get(0).(map[string]bool)
}

// MockLLMClientForMissing for testing LLM-related handlers
type MockLLMClientForMissing struct {
	mock.Mock
}

func (m *MockLLMClientForMissing) ValidateAPIKey() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockLLMClientForMissing) AnalyzeArticle(title, content string) (float64, float64, error) {
	args := m.Called(title, content)
	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

// TestRefreshHandler tests the RSS refresh handler
func TestRefreshHandler(t *testing.T) {
	ginTestModeOnceMissing.Do(func() {
		gin.SetMode(gin.TestMode)
	})

	tests := []struct {
		name           string
		setupMock      func(*MockRSSCollectorForMissing)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful_refresh_trigger",
			setupMock: func(mockRSS *MockRSSCollectorForMissing) {
				// The refreshHandler calls ManualRefresh in a goroutine, so we need to handle async calls
				mockRSS.On("ManualRefresh").Return().Maybe()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool), "Success should be true")
				data := response["data"].(map[string]interface{})
				assert.Equal(t, "refresh started", data["status"], "Status should indicate refresh started")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock RSS collector
			mockRSS := &MockRSSCollectorForMissing{}
			tt.setupMock(mockRSS)

			// Create handler
			handler := refreshHandler(mockRSS)

			// Setup router
			router := gin.New()
			router.POST("/api/refresh", handler)

			// Create request
			req := httptest.NewRequest("POST", "/api/refresh", nil)
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Verify response status
			assert.Equal(t, tt.expectedStatus, w.Code, "Unexpected status code")

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err, "Failed to parse response JSON")

			// Run custom response checks
			tt.checkResponse(t, response)

			// Verify mock expectations
			mockRSS.AssertExpectations(t)
		})
	}
}

// TestFeedHealthHandler tests the RSS feed health check handler
func TestFeedHealthHandler(t *testing.T) {
	ginTestModeOnceMissing.Do(func() {
		gin.SetMode(gin.TestMode)
	})

	tests := []struct {
		name           string
		setupMock      func(*MockRSSCollectorForMissing)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "all_feeds_healthy",
			setupMock: func(mockRSS *MockRSSCollectorForMissing) {
				mockRSS.On("CheckFeedHealth").Return(map[string]bool{
					"https://feeds.cnn.com/rss/edition.rss":     true,
					"https://feeds.bbc.co.uk/news/rss.xml":      true,
					"https://feeds.reuters.com/reuters/topNews": true,
				})
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				// feedHealthHandler returns the health map directly, not wrapped in success/data
				assert.Equal(t, true, response["https://feeds.cnn.com/rss/edition.rss"], "CNN feed should be healthy")
				assert.Equal(t, true, response["https://feeds.bbc.co.uk/news/rss.xml"], "BBC feed should be healthy")
				assert.Equal(t, true, response["https://feeds.reuters.com/reuters/topNews"], "Reuters feed should be healthy")
			},
		},
		{
			name: "some_feeds_unhealthy",
			setupMock: func(mockRSS *MockRSSCollectorForMissing) {
				mockRSS.On("CheckFeedHealth").Return(map[string]bool{
					"https://feeds.cnn.com/rss/edition.rss": true,
					"https://broken-feed.com/rss":           false,
					"https://feeds.bbc.co.uk/news/rss.xml":  true,
				})
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["https://feeds.cnn.com/rss/edition.rss"], "CNN feed should be healthy")
				assert.Equal(t, false, response["https://broken-feed.com/rss"], "Broken feed should be unhealthy")
				assert.Equal(t, true, response["https://feeds.bbc.co.uk/news/rss.xml"], "BBC feed should be healthy")
			},
		},
		{
			name: "no_feeds_configured",
			setupMock: func(mockRSS *MockRSSCollectorForMissing) {
				mockRSS.On("CheckFeedHealth").Return(map[string]bool{})
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, 0, len(response), "Should return empty map when no feeds configured")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock RSS collector
			mockRSS := &MockRSSCollectorForMissing{}
			tt.setupMock(mockRSS)

			// Create handler
			handler := feedHealthHandler(mockRSS)

			// Setup router
			router := gin.New()
			router.GET("/api/feeds/healthz", handler)

			// Create request
			req := httptest.NewRequest("GET", "/api/feeds/healthz", nil)
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Verify response status
			assert.Equal(t, tt.expectedStatus, w.Code, "Unexpected status code")

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err, "Failed to parse response JSON")

			// Run custom response checks
			tt.checkResponse(t, response)

			// Verify mock expectations
			mockRSS.AssertExpectations(t)
		})
	}
}

// TestLLMHealthHandler tests the LLM health check handler
func TestLLMHealthHandler(t *testing.T) {
	ginTestModeOnceMissing.Do(func() {
		gin.SetMode(gin.TestMode)
	})

	tests := []struct {
		name           string
		setupMock      func(*MockLLMClientForMissing)
		llmClient      *llm.LLMClient
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "llm_client_nil",
			setupMock: func(mockLLM *MockLLMClientForMissing) {
				// No setup needed for nil client test
			},
			llmClient:      nil,
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.False(t, response["success"].(bool), "Success should be false")
				assert.Contains(t, response, "error", "Should contain error field")
				errorObj := response["error"].(map[string]interface{})
				assert.Contains(t, errorObj["message"], "LLM client not initialized")
			},
		},
		// Note: Testing with actual LLM client would require complex setup
		// The main coverage is testing the nil client case and error handling
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLM client
			mockLLM := &MockLLMClientForMissing{}
			tt.setupMock(mockLLM)

			// Create handler
			handler := llmHealthHandler(tt.llmClient)

			// Setup router
			router := gin.New()
			router.GET("/api/llm/health", handler)

			// Create request
			req := httptest.NewRequest("GET", "/api/llm/health", nil)
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Verify response status
			assert.Equal(t, tt.expectedStatus, w.Code, "Unexpected status code")

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err, "Failed to parse response JSON")

			// Run custom response checks
			tt.checkResponse(t, response)

			// Verify mock expectations if LLM client was used
			if tt.llmClient != nil {
				mockLLM.AssertExpectations(t)
			}
		})
	}
}
