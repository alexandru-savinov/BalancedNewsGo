package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockArticleDB struct {
	mock.Mock
}

func (m *MockArticleDB) GetArticleByID(ctx context.Context, id int64) (*db.Article, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	val, ok := args.Get(0).(*db.Article)
	originalError := args.Error(1)
	if !ok {
		// log.Printf("WARN: MockArticleDB.GetArticleByID: type assertion to *db.Article failed for ID %d", id)
		if originalError == nil {
			return nil, fmt.Errorf("MockArticleDB.GetArticleByID: type assertion failed")
		}
		return nil, originalError
	}
	return val, originalError
}

func TestGetArticleHandler(t *testing.T) {
	// Setup mock
	mockDB := &MockArticleDB{}
	mockDB.On("GetArticleByID", mock.Anything, int64(1)).Return(&db.Article{
		ID:    1,
		Title: "Test Article",
	}, nil)

	// Setup router
	router := gin.Default()
	router.GET("/api/article/:id", func(c *gin.Context) {
		// Implement proper handler behavior
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid article ID",
			})
			return
		}

		article, err := mockDB.GetArticleByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to retrieve article",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    article,
		})
	})

	// Test request
	req, _ := http.NewRequest("GET", "/api/article/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rec.Code)
	mockDB.AssertExpectations(t)
}
