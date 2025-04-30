// File: api_integration_legacy_test.go
// Purpose: Legacy integration and handler tests for API edge cases and progress/SSE flows.
// Note: This file contains only handler-level and progress/SSE tests. True end-to-end rescoring and scoring pipeline tests are performed via Postman collections and external E2E tools.
//
// For real E2E and workflow validation, see Postman collections in /postman and scripts in /test_sse_progress.js.

package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	_ "modernc.org/sqlite"
)

// --- Integration Tests for api.go ---

// TestSSEProgressConcurrentClients verifies multiple clients receive progress updates
func TestSSEProgressConcurrentClients(t *testing.T) {
	done := make(chan struct{})
	go func() {
		// Set the final state before starting clients
		progressMapLock.Lock()
		progressMap[1] = &models.ProgressState{
			Step:    "done",
			Percent: 100,
			Status:  "Success",
		}
		progressMapLock.Unlock()

		router := gin.New()
		router.GET("/api/llm/score-progress/:id", scoreProgressSSEHandler())

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
type panicCalculator struct{}

func (p *panicCalculator) CalculateScore(scores []db.LLMScore) (float64, float64, error) {
	panic("simulated panic")
}

func TestReanalyzeHandlerPanicRecovery(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		progressMgr := llm.NewProgressManager(1 * time.Minute)
		manager := llm.NewScoreManager(nil, nil, &panicCalculator{}, progressMgr)
		dbConn, err := sqlx.Open("sqlite", ":memory:")
		if err != nil {
			done <- err
			return
		}
		defer dbConn.Close()

		router := gin.New()
		router.POST("/api/llm/reanalyze/:id", reanalyzeHandler(nil, dbConn, manager))

		req, _ := http.NewRequest("POST", "/api/llm/reanalyze/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Wait for goroutine to finish and check progress state
		time.Sleep(100 * time.Millisecond)
		progress := progressMgr.GetProgress(1)
		if progress != nil {
			if progress.Status != "Error" {
				done <- fmt.Errorf("expected status Error, got %s", progress.Status)
				return
			}
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("TestReanalyzeHandlerPanicRecovery timed out")
	}
}

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
	return args.Get(0).(*models.ProgressState)
}
