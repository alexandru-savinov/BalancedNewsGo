// EMERGENCY STUBS - Generated for emergency build recovery
// Created: June 14, 2025
// Owner: API Team Lead
// TODO: Replace with proper implementations within 48 hours
// WARNING: These are minimal stubs for emergency build recovery only

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/gin-gonic/gin"
)

// Emergency stub: Missing template handlers with proper structure
type APITemplateHandlers struct {
	initialized  bool
	version      string
	createdAt    time.Time
	stubMode     bool
	handlerCount int
	baseURL      string
}

func NewAPITemplateHandlers(baseURL string) *APITemplateHandlers {
	return &APITemplateHandlers{
		initialized:  true,
		version:      "emergency-stub-2.0",
		createdAt:    time.Now(),
		stubMode:     true,
		handlerCount: 0,
		baseURL:      baseURL,
	}
}

// Method stubs for APITemplateHandlers
func (h *APITemplateHandlers) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"status":     "emergency_stub",
		"version":    h.version,
		"created_at": h.createdAt,
		"stub_mode":  h.stubMode,
		"base_url":   h.baseURL,
	}
}

// Emergency method stubs for APITemplateHandlers - these match the expected interface
func (h *APITemplateHandlers) GetArticleStats(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_articles": 0,
		"status":         "emergency_stub",
		"message":        "API template handler temporarily unavailable during emergency recovery",
	}, nil
}

func (h *APITemplateHandlers) GetSourceStats(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_sources": 0,
		"status":        "emergency_stub",
		"message":       "API template handler temporarily unavailable during emergency recovery",
	}, nil
}

// Emergency stub: Missing progress tracking with thread safety
var progressMapLock sync.RWMutex
var progressMap = make(map[int64]*models.ProgressState)

// Emergency stub: Progress management utilities
func GetProgress(articleID int64) (*models.ProgressState, bool) {
	progressMapLock.RLock()
	defer progressMapLock.RUnlock()
	state, exists := progressMap[articleID]
	return state, exists
}

func SetProgress(articleID int64, state *models.ProgressState) {
	progressMapLock.Lock()
	defer progressMapLock.Unlock()
	progressMap[articleID] = state
}

// Emergency stub: Missing handlers with comprehensive responses
func scoreProgressHandler(pm *llm.ProgressManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get article ID from URL parameter
		articleIDStr := c.Param("id")
		articleID, err := strconv.ParseInt(articleIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid article ID",
			})
			return
		}
		// Set SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Cache-Control")

		// Get flusher for immediate sending
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Streaming unsupported",
			})
			return
		}

		// Send initial event
		initialState := pm.GetProgress(articleID)
		if initialState == nil {
			initialState = &models.ProgressState{
				Status:      "Initializing",
				Message:     "SSE connection established, awaiting progress.",
				Percent:     0,
				LastUpdated: 0,
			}
		}

		// Send initial progress event
		fmt.Fprintf(c.Writer, "event: progress\n")
		fmt.Fprintf(c.Writer, "data: %s\n\n", formatProgressData(*initialState))
		flusher.Flush()

		// Set up ticker for periodic updates
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		// Context for cancellation
		ctx := c.Request.Context()

		for {
			select {
			case <-ctx.Done():
				// Client disconnected
				return
			case <-ticker.C:
				// Check for progress updates
				if pm != nil {
					if currentState := pm.GetProgress(articleID); currentState != nil {
						fmt.Fprintf(c.Writer, "event: progress\n")
						fmt.Fprintf(c.Writer, "data: %s\n\n", formatProgressData(*currentState))
						flusher.Flush()

						// Stop if completed
						if currentState.Status == "Completed" || currentState.Status == "Failed" {
							return
						}
					}
				}
			}
		}
	}
}

// Helper function to format progress data as JSON
func formatProgressData(state models.ProgressState) string {
	data, _ := json.Marshal(map[string]interface{}{
		"step":         state.Status,
		"message":      state.Message,
		"percent":      state.Percent,
		"status":       "Connected",
		"last_updated": state.LastUpdated,
	})
	return string(data)
}

// Emergency stub: Health check endpoint for monitoring
func EmergencyHealthHandler(c *gin.Context) {
	c.Header("X-Emergency-Mode", "active")

	health := gin.H{
		"status":                  "emergency_recovery",
		"build_status":            "functional",
		"api_status":              "stubs_active",
		"estimated_full_recovery": "48-72 hours",
		"emergency_contact":       "dev-team-lead@company.com",
		"last_updated":            time.Now().UTC(), "stub_handlers": []string{
			"scoreProgressHandler",
			"APITemplateHandlers",
		},
	}

	c.JSON(http.StatusOK, health)
}
