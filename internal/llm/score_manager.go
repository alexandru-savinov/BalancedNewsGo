package llm

import (
	"github.com/jmoiron/sqlx"
)

// ScoreManager orchestrates score operations and dependencies
// This is a skeleton for the refactor, to be filled in with logic in later steps
type ScoreManager struct {
	db          *sqlx.DB
	cache       *Cache
	calculator  ScoreCalculator
	progressMgr *ProgressManager
}

// NewScoreManager creates a new score manager with dependencies
func NewScoreManager(db *sqlx.DB, cache *Cache, calculator ScoreCalculator, progressMgr *ProgressManager) *ScoreManager {
	return &ScoreManager{
		db:          db,
		cache:       cache,
		calculator:  calculator,
		progressMgr: progressMgr,
	}
}

// UpdateArticleScore handles atomic update of score and confidence (stub)
func (sm *ScoreManager) UpdateArticleScore(articleID int64, scores []interface{}) (float64, float64, error) {
	// TODO: Implement transaction, calculation, storage, cache invalidation
	return 0, 0, nil
}

// InvalidateScoreCache invalidates all score-related caches for an article (stub)
func (sm *ScoreManager) InvalidateScoreCache(articleID int64) {
	// TODO: Implement cache invalidation
}

// TrackProgress is a stub for progress tracking
func (sm *ScoreManager) TrackProgress(articleID int64, step, status string) {
	// TODO: Implement progress tracking
}

// SetProgress proxies to ProgressManager
func (sm *ScoreManager) SetProgress(articleID int64, state *ProgressState) {
	if sm.progressMgr != nil {
		sm.progressMgr.SetProgress(articleID, state)
	}
}

// GetProgress proxies to ProgressManager
func (sm *ScoreManager) GetProgress(articleID int64) *ProgressState {
	if sm.progressMgr != nil {
		return sm.progressMgr.GetProgress(articleID)
	}
	return nil
}
