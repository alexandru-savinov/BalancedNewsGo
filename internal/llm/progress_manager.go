package llm

import (
	"sync"
	"time"
)

// ProgressState mirrors the structure used in internal/api/api.go
// (Step, Message, Percent, Status, Error, FinalScore, LastUpdated)
type ProgressState struct {
	Step        string   `json:"step"`
	Message     string   `json:"message"`
	Percent     int      `json:"percent"`
	Status      string   `json:"status"`
	Error       string   `json:"error,omitempty"`
	FinalScore  *float64 `json:"final_score,omitempty"`
	LastUpdated int64    `json:"last_updated"`
}

// ProgressManager tracks scoring progress with cleanup
type ProgressManager struct {
	progressMap     map[int64]*ProgressState
	progressMapLock sync.RWMutex
	cleanupInterval time.Duration
}

// NewProgressManager creates a progress manager with cleanup
func NewProgressManager(cleanupInterval time.Duration) *ProgressManager {
	pm := &ProgressManager{
		progressMap:     make(map[int64]*ProgressState),
		cleanupInterval: cleanupInterval,
	}
	go pm.startCleanupRoutine()
	return pm
}

// SetProgress sets the progress state for an article
func (pm *ProgressManager) SetProgress(articleID int64, state *ProgressState) {
	pm.progressMapLock.Lock()
	defer pm.progressMapLock.Unlock()
	pm.progressMap[articleID] = state
}

// GetProgress retrieves the progress state for an article
func (pm *ProgressManager) GetProgress(articleID int64) *ProgressState {
	pm.progressMapLock.RLock()
	defer pm.progressMapLock.RUnlock()
	return pm.progressMap[articleID]
}

// startCleanupRoutine periodically removes stale entries
func (pm *ProgressManager) startCleanupRoutine() {
	ticker := time.NewTicker(pm.cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		pm.cleanup()
	}
}

// cleanup removes completed or stale progress entries
func (pm *ProgressManager) cleanup() {
	pm.progressMapLock.Lock()
	defer pm.progressMapLock.Unlock()
	now := time.Now().Unix()
	for id, progress := range pm.progressMap {
		if (progress.Status == "Success" || progress.Status == "Error") && now-progress.LastUpdated > 300 {
			delete(pm.progressMap, id)
			continue
		}
		if progress.Status == "InProgress" && now-progress.LastUpdated > 1800 {
			delete(pm.progressMap, id)
		}
	}
}
