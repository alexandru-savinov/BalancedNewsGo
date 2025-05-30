package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type feedHealthMock struct{ mock.Mock }

func (m *feedHealthMock) ManualRefresh() { m.Called() }
func (m *feedHealthMock) CheckFeedHealth() map[string]bool {
	args := m.Called()
	if val, ok := args.Get(0).(map[string]bool); ok {
		return val
	}
	return nil
}

func TestFeedHealthHandlerCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articlesCache = NewSimpleCache()

	mockRSS := &feedHealthMock{}
	result := map[string]bool{"feed1": true}
	mockRSS.On("CheckFeedHealth").Return(result).Once()

	router := gin.New()
	router.GET("/healthz", feedHealthHandler(mockRSS))

	req, _ := http.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second call should use cache
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusOK, w2.Code)

	mockRSS.AssertExpectations(t)
}

func TestFeedHealthCacheInvalidatedOnRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articlesCache = NewSimpleCache()

	mockRSS := &feedHealthMock{}
	result := map[string]bool{"feed1": true}
	mockRSS.On("CheckFeedHealth").Return(result).Twice()
	mockRSS.On("ManualRefresh").Return().Once()

	router := gin.New()
	router.GET("/healthz", feedHealthHandler(mockRSS))
	router.POST("/refresh", refreshHandler(mockRSS))

	req, _ := http.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	reqRefresh, _ := http.NewRequest("POST", "/refresh", nil)
	wRef := httptest.NewRecorder()
	router.ServeHTTP(wRef, reqRefresh)
	time.Sleep(10 * time.Millisecond)

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	mockRSS.AssertExpectations(t)
}
