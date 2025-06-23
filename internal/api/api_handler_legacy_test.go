// File: api_integration_legacy_test.go
// Purpose: Legacy integration and handler tests for API edge cases and progress/SSE flows.
// Note: This file contains only handler-level and progress/SSE tests.
// True end-to-end rescoring and scoring pipeline tests are performed via Postman collections
// and external E2E tools.
//
// For real E2E and workflow validation, see Postman collections in /postman and scripts in /test_sse_progress.js.

package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	_ "modernc.org/sqlite"
)

// --- Integration Tests for api.go ---

// TestSSEProgressConcurrentClients verifies multiple clients receive progress updates
func TestSSEProgressConcurrentClients(t *testing.T) {
	done := make(chan struct{})
	go func() { // Create a simple progress manager for testing
		progressMgr := llm.NewProgressManager(5 * time.Minute)

		// Create a minimal score manager with the progress manager
		scoreManager := &llm.ScoreManager{}
		// Use reflection to set the progressMgr field since it's private
		// Or alternatively, create it using the constructor with nil for other fields

		// Set the expected progress state directly in the progress manager
		expectedProgress := &models.ProgressState{
			Step:    "done",
			Percent: 100,
			Status:  "Success",
		}
		progressMgr.SetProgress(1, expectedProgress)

		// Create ScoreManager with the progress manager (pass nil for other dependencies we don't need)
		scoreManager = llm.NewScoreManager(nil, nil, nil, progressMgr)

		router := gin.New()
		router.GET("/api/llm/score-progress/:id", scoreProgressSSEHandler(scoreManager))

		var wg sync.WaitGroup
		clients := 5

		for i := 0; i < clients; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequest("GET", "/api/llm/score-progress/1", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Body.String(), `"status":"Success"`)
			}()
		}
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// test completed
	case <-time.After(3 * time.Second):
		t.Fatal("TestSSEProgressConcurrentClients timed out")
	}
}

// --- Helpers for panic recovery test ---

// --- Use the existing MockDBOperations from api_test.go ---
// --- Minimal MockScoreManager for this test ---
type MockScoreManager struct{ mock.Mock }

func (m *MockScoreManager) UpdateArticleScore(articleID int64, scores []db.LLMScore, cfg interface{}) (float64, float64, error) {
	args := m.Called(articleID, scores, cfg)
	return 0, 0, args.Error(0)
}
func (m *MockScoreManager) SetProgress(articleID int64, state *models.ProgressState) {
	m.Called(articleID, state)
}
func (m *MockScoreManager) GetProgress(articleID int64) *models.ProgressState {
	args := m.Called(articleID)
	if args.Get(0) == nil {
		return nil
	}
	val, ok := args.Get(0).(*models.ProgressState)
	if !ok {
		// This might indicate a misconfiguration of the mock's Return arguments
		// or the test is intentionally providing a different type.
		// For a mock, returning nil might be acceptable, or you could panic.
		// log.Printf("WARN: MockScoreManager.GetProgress: type assertion to *models.ProgressState failed for articleID %d", articleID)
		return nil
	}
	return val
}
