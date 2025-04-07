package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestArticlesEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/articles", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
	if len(w.Body.String()) == 0 {
		t.Errorf("Expected non-empty response body")
	}
}

func TestHealthzEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/articles", func(c *gin.Context) {
		c.String(200, "<div>Test Article</div>")
	})
	return router
}
