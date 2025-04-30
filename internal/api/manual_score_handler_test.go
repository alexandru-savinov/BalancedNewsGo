package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestManualScoreHandler(t *testing.T) {
	// Setup test router
	router := gin.New()
	router.POST("/api/scores", func(c *gin.Context) {
		var req struct {
			Score float64 `json:"score"`
		}
		if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// Test cases
	t.Run("successful score update", func(t *testing.T) {
		reqBody := bytes.NewBufferString(`{"score": 0.5}`)
		req, _ := http.NewRequest("POST", "/api/scores", reqBody)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var response map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["status"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		reqBody := bytes.NewBufferString(`{invalid json}`)
		req, _ := http.NewRequest("POST", "/api/scores", reqBody)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}
