package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	articlesPath = "/api/articles"
)

type MockArticlesDB struct {
	mock.Mock
}

func (m *MockArticlesDB) FetchArticles(ctx context.Context, source, leaning string, limit, offset int) ([]*db.Article, error) {
	args := m.Called(ctx, source, leaning, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	val, ok := args.Get(0).([]*db.Article)
	originalError := args.Error(1)
	if !ok {
		// log.Printf("WARN: MockArticlesDB.FetchArticles: type assertion to []*db.Article failed")
		if originalError == nil {
			return nil, fmt.Errorf("MockArticlesDB.FetchArticles: type assertion failed")
		}
		return nil, originalError
	}
	return val, originalError
}

func (m *MockArticlesDB) ArticleExistsByURL(ctx context.Context, url string) (bool, error) {
	args := m.Called(ctx, url)
	return args.Bool(0), args.Error(1)
}

func setupArticlesTestRouter(mockDB *MockArticlesDB) *gin.Engine {
	router := gin.Default()
	router.GET(articlesPath, func(c *gin.Context) {
		// Implement proper handler behavior that calls the mock
		source := c.Query("source")
		leaning := c.Query("leaning")
		limit := 20 // Default value
		offset := 0 // Default value

		// Get the articles using the mock
		articles, err := mockDB.FetchArticles(c.Request.Context(), source, leaning, limit, offset)

		if err != nil {
			// Return error response with 500 status code
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to fetch articles",
			})
			return
		}

		// Return success response
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    articles,
		})
	})
	return router
}

func TestGetArticlesHandlerSuccessWithMock(t *testing.T) {
	ctx := context.Background()
	mockDB := &MockArticlesDB{}
	mockDB.On("FetchArticles", ctx, "", "", 20, 0).Return([]*db.Article{{ID: 1}}, nil)

	router := setupArticlesTestRouter(mockDB)
	req, _ := http.NewRequest("GET", articlesPath, nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	mockDB.AssertExpectations(t)
}

func TestGetArticlesHandlerError(t *testing.T) {
	ctx := context.Background()
	mockDB := &MockArticlesDB{}
	mockDB.On("FetchArticles", ctx, "", "", 20, 0).Return(nil, assert.AnError)

	router := setupArticlesTestRouter(mockDB)
	req, _ := http.NewRequest("GET", articlesPath, nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	mockDB.AssertExpectations(t)
}
