package api

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorHandlingMiddleware catches panics and converts them to 500 errors
func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a request ID for tracking
		requestID := uuid.New().String()
		c.Set("RequestID", requestID)
		c.Header("X-Request-ID", requestID)

		// Start timer for request duration
		startTime := time.Now()

		// Add recovery handling
		defer func() {
			if err := recover(); err != nil {
				// Log the error with stack trace
				stack := string(debug.Stack())
				log.Printf("[PANIC][%s] %s\n%s", requestID, err, stack)

				// Respond with a generic error to avoid exposing sensitive details
				RespondError(c, http.StatusInternalServerError, "An unexpected error occurred")

				// Mark request as completed to avoid further processing
				c.Abort()
			}
		}()

		// Process request
		c.Next()

		// Log request completion
		duration := time.Since(startTime)
		status := c.Writer.Status()

		// Log differently based on status code
		if status >= 400 {
			log.Printf("[ERROR][%s] %s %s - %d (%s)",
				requestID, c.Request.Method, c.Request.URL.Path, status, duration)
		} else {
			log.Printf("[INFO][%s] %s %s - %d (%s)",
				requestID, c.Request.Method, c.Request.URL.Path, status, duration)
		}
	}
}

// RateLimitMiddleware provides basic rate limiting
func RateLimitMiddleware(rps int, windowSec int) gin.HandlerFunc {
	// Simple in-memory store for rate limiting
	// In production, use Redis or similar for distributed rate limiting
	type clientWindow struct {
		count     int
		startTime time.Time
	}

	windows := make(map[string]*clientWindow)

	return func(c *gin.Context) {
		// Get client IP or use API key if available
		clientID := c.ClientIP()
		if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
			clientID = apiKey
		}

		// Get or create window
		window, exists := windows[clientID]
		now := time.Now()

		if !exists || now.Sub(window.startTime) > time.Duration(windowSec)*time.Second {
			// Create new window
			windows[clientID] = &clientWindow{
				count:     1,
				startTime: now,
			}
		} else {
			// Check if rate limit exceeded
			if window.count >= rps {
				// Calculate reset time
				resetTime := window.startTime.Add(time.Duration(windowSec) * time.Second)
				resetMillis := resetTime.UnixNano() / int64(time.Millisecond)

				// Set rate limit headers
				c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rps))
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetMillis))

				// Create headers map for error response
				headers := map[string]string{
					"X-RateLimit-Limit":     fmt.Sprintf("%d", rps),
					"X-RateLimit-Remaining": "0",
					"X-RateLimit-Reset":     fmt.Sprintf("%d", resetMillis),
				}

				// Return rate limit error
				RespondRateLimitError(c, "Rate limit exceeded", headers)
				c.Abort()
				return
			}

			// Increment counter
			window.count++

			// Set rate limit headers for successful requests
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rps))
			c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", rps-window.count))
		}

		c.Next()
	}
}

// RequestLoggingMiddleware logs incoming requests
func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get request ID from error handling middleware
		requestID, exists := c.Get("RequestID")
		if !exists {
			requestID = uuid.New().String()
		}

		// Log request details
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		if query != "" {
			path = path + "?" + query
		}

		// Sanitize authorization headers for logging
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) > 1 {
				// Mask token except first and last few characters
				token := parts[1]
				if len(token) > 10 {
					maskedToken := token[:4] + "..." + token[len(token)-4:]
					authHeader = parts[0] + " " + maskedToken
				}
			}
		}

		log.Printf("[REQUEST][%s] %s %s - Auth: %s",
			requestID, method, path, authHeader)

		c.Next()
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}
