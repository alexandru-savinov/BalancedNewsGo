package llm

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
)

// Define constants for progress states
const (
	ProgressStatusInProgress = "InProgress"
	ProgressStatusSuccess    = "Success"
	ProgressStatusError      = "Error"

	ProgressStepStart       = "Start"
	ProgressStepCalculating = "Calculating"
	ProgressStepStoring     = "Storing"
	ProgressStepUpdating    = "Updating"
	ProgressStepComplete    = "Complete"
	ProgressStepError       = "Error" // Also a step
)

// ProgressManager tracks scoring progress with cleanup
type ProgressManager struct {
	progressMap     map[int64]*models.ProgressState
	progressMapLock sync.RWMutex
	cleanupInterval time.Duration
	stopChan        chan struct{}
	stopped         bool
}

// NewProgressManager creates a progress manager with cleanup
func NewProgressManager(cleanupInterval time.Duration) *ProgressManager {
	pm := &ProgressManager{
		progressMap:     make(map[int64]*models.ProgressState),
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
		stopped:         false,
	}

	// Only start cleanup routine in non-test environments to prevent I/O timeout
	if os.Getenv("TEST_MODE") != "true" && os.Getenv("NO_AUTO_ANALYZE") != "true" && os.Getenv("CI") != "true" {
		go pm.startCleanupRoutine()
	}

	return pm
}

// SetProgress sets the progress state for an article
func (pm *ProgressManager) SetProgress(articleID int64, state *models.ProgressState) {
	pm.progressMapLock.Lock()
	defer pm.progressMapLock.Unlock()
	pm.progressMap[articleID] = state
}

// UpdateProgress updates progress state with error handling
func (pm *ProgressManager) UpdateProgress(articleID int64, step string, percent int, status string, err error) {
	pm.progressMapLock.Lock()
	defer pm.progressMapLock.Unlock()

	state, exists := pm.progressMap[articleID]
	if !exists {
		state = &models.ProgressState{
			LastUpdated: time.Now().Unix(),
		}
		pm.progressMap[articleID] = state
	}

	state.Step = step
	state.Percent = percent
	state.Status = status
	state.LastUpdated = time.Now().Unix()

	// Enhanced error handling for LLM errors
	if err != nil {
		state.Error = err.Error()

		// Add specific error details for LLM errors
		var llmErr LLMAPIError
		if errors.As(err, &llmErr) {
			errorDetails := map[string]interface{}{
				"type":        string(llmErr.ErrorType),
				"status_code": llmErr.StatusCode,
				"message":     llmErr.Message,
			}

			// Only include retry_after if present
			if llmErr.RetryAfter > 0 {
				errorDetails["retry_after"] = llmErr.RetryAfter
			}

			// Convert to JSON string
			if detailsJSON, jsonErr := json.Marshal(errorDetails); jsonErr == nil {
				state.ErrorDetails = string(detailsJSON)
			}
		}
	} else {
		state.Error = ""
		state.ErrorDetails = ""
	}
}

// GetProgress retrieves the progress state for an article
func (pm *ProgressManager) GetProgress(articleID int64) *models.ProgressState {
	pm.progressMapLock.RLock()
	defer pm.progressMapLock.RUnlock()
	return pm.progressMap[articleID]
}

// Stop gracefully shuts down the progress manager
func (pm *ProgressManager) Stop() {
	pm.progressMapLock.Lock()
	defer pm.progressMapLock.Unlock()

	if !pm.stopped {
		pm.stopped = true
		close(pm.stopChan)
	}
}

// startCleanupRoutine periodically removes stale entries
func (pm *ProgressManager) startCleanupRoutine() {
	ticker := time.NewTicker(pm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.cleanup()
		case <-pm.stopChan:
			return // Exit the goroutine when stop is signaled
		}
	}
}

// cleanup removes completed or stale progress entries
func (pm *ProgressManager) cleanup() {
	pm.progressMapLock.Lock()
	defer pm.progressMapLock.Unlock()
	now := time.Now().Unix()
	for id, progress := range pm.progressMap {
		if (progress.Status == ProgressStatusSuccess || progress.Status == ProgressStatusError) && now-progress.LastUpdated > 300 {
			delete(pm.progressMap, id)
			continue
		}
		if progress.Status == ProgressStatusInProgress && now-progress.LastUpdated > 1800 {
			delete(pm.progressMap, id)
		}
	}
}
