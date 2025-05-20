package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSafeHandlerExtended(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	t.Run("should handle normal execution without panic", func(t *testing.T) {
		router := gin.New()

		// Create a handler that simply sets a 200 status
		normalHandler := func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "success"})
		}

		// Wrap with SafeHandler
		router.GET("/normal", SafeHandler(normalHandler))

		// Create request and response recorder
		req := httptest.NewRequest("GET", "/normal", nil)
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "success")
	})

	t.Run("should recover from panic with custom error message", func(t *testing.T) {
		router := gin.New()

		// Create a handler that will panic with a custom error message
		panicHandler := func(c *gin.Context) {
			err := &ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "TEST_ERROR",
					Message: "Custom test error",
				},
			}
			panic(err)
		}

		// Wrap with SafeHandler
		router.GET("/panic-custom", SafeHandler(panicHandler))

		// Create request and response recorder
		req := httptest.NewRequest("GET", "/panic-custom", nil)
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal Server Error")
	})

	t.Run("should recover from nil panic", func(t *testing.T) {
		router := gin.New()

		// Create a handler that will panic with nil
		panicHandler := func(c *gin.Context) {
			panic("nil panic")
		}

		// Wrap with SafeHandler
		router.GET("/panic-nil", SafeHandler(panicHandler))

		// Create request and response recorder
		req := httptest.NewRequest("GET", "/panic-nil", nil)
		w := httptest.NewRecorder()

		// Serve the request
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal Server Error")
	})
}
