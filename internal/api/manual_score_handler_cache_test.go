package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockInvalidator struct{ mock.Mock }

func (m *mockInvalidator) InvalidateScoreCache(id int64) { m.Called(id) }

func TestManualScoreHandlerInvalidatesCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDB := &MockDBOperations{}
	invalidator := &mockInvalidator{}

	articleID := int64(42)
	mockDB.On("FetchArticleByID", mock.Anything, articleID).Return(&db.Article{ID: articleID}, nil)
	mockDB.On("UpdateArticleScore", mock.Anything, articleID, 0.5, 1.0).Return(nil)
	invalidator.On("InvalidateScoreCache", articleID).Return()

	router := gin.New()
	router.POST("/manual-score/:id", SafeHandler(manualScoreHandler(mockDB, invalidator)))

	body := bytes.NewBufferString(`{"score":0.5}`)
	req, _ := http.NewRequest("POST", fmt.Sprintf("/manual-score/%d", articleID), body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	invalidator.AssertCalled(t, "InvalidateScoreCache", articleID)
}
