package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func setupSourcesTestDB(t *testing.T) *sqlx.DB {
	dbConn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() { dbConn.Close() })
	return dbConn
}

func TestGetSourcesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articlesCache = NewSimpleCache()
	dbConn := setupSourcesTestDB(t)

	// Insert initial data
	articles := []db.Article{
		{Source: "CNN", PubDate: time.Now(), URL: "http://cnn.com/1", Title: "t1", Content: "c1", CreatedAt: time.Now()},
		{Source: "BBC", PubDate: time.Now(), URL: "http://bbc.com/1", Title: "t2", Content: "c2", CreatedAt: time.Now()},
	}
	for i := range articles {
		_, err := db.InsertArticle(dbConn, &articles[i])
		assert.NoError(t, err)
	}

	router := gin.New()
	router.GET("/api/sources", SafeHandler(getSourcesHandler(dbConn)))

	req, _ := http.NewRequest("GET", "/api/sources", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "CNN")
	assert.Contains(t, w.Body.String(), "BBC")

	// Add new article after first call
	_, err := db.InsertArticle(dbConn, &db.Article{Source: "NYT", PubDate: time.Now(), URL: "http://nyt.com/1", Title: "t3", Content: "c3", CreatedAt: time.Now()})
	assert.NoError(t, err)

	// Second call should use cache and not include NYT
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.NotContains(t, w2.Body.String(), "NYT")
}
