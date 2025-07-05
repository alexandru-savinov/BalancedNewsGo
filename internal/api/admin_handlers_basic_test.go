package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	ginTestModeOnceBasic sync.Once
)

// MockRSSCollectorBasic for basic testing
type MockRSSCollectorBasic struct {
	mock.Mock
}

func (m *MockRSSCollectorBasic) ManualRefresh() {
	m.Called()
}

func (m *MockRSSCollectorBasic) CheckFeedHealth() map[string]bool {
	args := m.Called()
	return args.Get(0).(map[string]bool)
}

func setupBasicTestRouter() *gin.Engine {
	ginTestModeOnceBasic.Do(func() {
		gin.SetMode(gin.TestMode)
	})
	return gin.New()
}

func TestAdminRefreshFeedsHandlerBasic(t *testing.T) {
	router := setupBasicTestRouter()
	mockCollector := new(MockRSSCollectorBasic)

	// Setup expectations with a channel to wait for the goroutine
	done := make(chan bool, 1)
	mockCollector.On("ManualRefresh").Run(func(args mock.Arguments) {
		done <- true
	}).Return()

	// Setup route
	router.POST("/api/admin/refresh-feeds", adminRefreshFeedsHandler(mockCollector))

	// Create request
	req := httptest.NewRequest("POST", "/api/admin/refresh-feeds", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	data, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "refresh_initiated", data["status"])
	assert.Contains(t, data["message"], "Feed refresh started successfully")

	// Wait for the goroutine to complete with timeout
	select {
	case <-done:
		// Success - the goroutine called ManualRefresh
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for ManualRefresh to be called")
	}

	mockCollector.AssertExpectations(t)
}

func TestAdminResetFeedErrorsHandlerBasic(t *testing.T) {
	router := setupBasicTestRouter()
	mockCollector := new(MockRSSCollectorBasic)

	// Setup route
	router.POST("/api/admin/reset-feed-errors", adminResetFeedErrorsHandler(mockCollector))

	// Create request
	req := httptest.NewRequest("POST", "/api/admin/reset-feed-errors", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	data, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "errors_reset", data["status"])
}

func TestAdminGetSourcesStatusHandlerBasic(t *testing.T) {
	router := setupBasicTestRouter()
	mockCollector := new(MockRSSCollectorBasic)

	// Setup expectations
	healthStatus := map[string]bool{
		"feed1": true,
		"feed2": false,
		"feed3": true,
	}
	mockCollector.On("CheckFeedHealth").Return(healthStatus)

	// Setup route
	router.GET("/api/admin/sources", adminGetSourcesStatusHandler(mockCollector))

	// Create request
	req := httptest.NewRequest("GET", "/api/admin/sources", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	data, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(2), data["active_sources"]) // JSON numbers are float64
	assert.Equal(t, float64(3), data["total_sources"])

	mockCollector.AssertExpectations(t)
}

func TestAdminGetLogsHandlerBasic(t *testing.T) {
	router := setupBasicTestRouter()

	// Setup route
	router.GET("/api/admin/logs", adminGetLogsHandler())

	// Create request
	req := httptest.NewRequest("GET", "/api/admin/logs", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response StandardResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	data, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	logs, ok := data["logs"].([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(logs), 0) // Should have sample logs
}

// Test for admin logs endpoint with different scenarios
func TestAdminGetLogsHandlerBasicScenarios(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		checkLogs      bool
	}{
		{
			name:           "successful logs retrieval",
			expectedStatus: http.StatusOK,
			checkLogs:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupBasicTestRouter()
			router.GET("/api/admin/logs", adminGetLogsHandler())

			req := httptest.NewRequest("GET", "/api/admin/logs", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkLogs {
				var response StandardResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)

				data, ok := response.Data.(map[string]interface{})
				assert.True(t, ok)
				assert.Contains(t, data, "logs")
				assert.Contains(t, data, "message")
				assert.Contains(t, data, "timestamp")
			}
		})
	}
}
