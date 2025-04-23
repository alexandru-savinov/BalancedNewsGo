package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
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

// UpdateArticleScore handles atomic update of score and confidence
func (sm *ScoreManager) UpdateArticleScore(articleID int64, scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
	if sm.calculator == nil {
		return 0, 0, fmt.Errorf("ScoreManager: calculator is nil")
	}
	if sm.db == nil {
		return 0, 0, fmt.Errorf("ScoreManager: db is nil")
	}
	if cfg == nil {
		return 0, 0, fmt.Errorf("ScoreManager: config is nil")
	}

	// Progress: Start
	if sm.progressMgr != nil {
		ps := &models.ProgressState{Step: "Start", Message: "Starting scoring", Percent: 0, Status: "InProgress", LastUpdated: time.Now().Unix()}
		sm.progressMgr.SetProgress(articleID, ps)
	}

	tx, err := sm.db.BeginTxx(context.Background(), nil)
	if err != nil {
		if sm.progressMgr != nil {
			ps := &models.ProgressState{Step: "DB Transaction", Message: "Failed to start DB transaction", Percent: 0, Status: "Error", Error: err.Error(), LastUpdated: time.Now().Unix()}
			sm.progressMgr.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		}
	}()

	// Progress: Calculating
	if sm.progressMgr != nil {
		ps := &models.ProgressState{Step: "Calculating", Message: "Calculating score", Percent: 20, Status: "InProgress", LastUpdated: time.Now().Unix()}
		sm.progressMgr.SetProgress(articleID, ps)
	}

	score, confidence, calcErr := sm.calculator.CalculateScore(scores)
	if calcErr != nil {
		tx.Rollback()
		if sm.progressMgr != nil {
			ps := &models.ProgressState{Step: "Calculation", Message: "Score calculation failed", Percent: 20, Status: "Error", Error: calcErr.Error(), LastUpdated: time.Now().Unix()}
			sm.progressMgr.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("score calculation failed: %w", calcErr)
	}

	// Progress: Storing ensemble score
	if sm.progressMgr != nil {
		ps := &models.ProgressState{Step: "Storing", Message: "Storing ensemble score", Percent: 60, Status: "InProgress", LastUpdated: time.Now().Unix()}
		sm.progressMgr.SetProgress(articleID, ps)
	}

	meta := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"aggregation": "ensemble",
		"confidence":  confidence,
	}
	metaBytes, _ := json.Marshal(meta)
	ensembleScore := &db.LLMScore{
		ArticleID: articleID,
		Model:     "ensemble",
		Score:     score,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
	}
	_, err = db.InsertLLMScore(tx, ensembleScore)
	if err != nil {
		tx.Rollback()
		if sm.progressMgr != nil {
			ps := &models.ProgressState{Step: "DB Insert", Message: "Failed to insert ensemble score", Percent: 70, Status: "Error", Error: err.Error(), LastUpdated: time.Now().Unix()}
			sm.progressMgr.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("failed to insert ensemble score: %w", err)
	}

	// Progress: Updating article
	if sm.progressMgr != nil {
		ps := &models.ProgressState{Step: "Updating", Message: "Updating article score", Percent: 80, Status: "InProgress", LastUpdated: time.Now().Unix()}
		sm.progressMgr.SetProgress(articleID, ps)
	}

	err = db.UpdateArticleScoreLLM(tx, articleID, score, confidence)
	if err != nil {
		tx.Rollback()
		if sm.progressMgr != nil {
			ps := &models.ProgressState{Step: "DB Update", Message: "Failed to update article", Percent: 90, Status: "Error", Error: err.Error(), LastUpdated: time.Now().Unix()}
			sm.progressMgr.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("failed to update article: %w", err)
	}

	if err := tx.Commit(); err != nil {
		if sm.progressMgr != nil {
			ps := &models.ProgressState{Step: "DB Commit", Message: "Failed to commit transaction", Percent: 95, Status: "Error", Error: err.Error(), LastUpdated: time.Now().Unix()}
			sm.progressMgr.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Progress: Invalidate cache
	if sm.cache != nil {
		sm.cache.Delete(fmt.Sprintf("article:%d", articleID))
		sm.cache.Delete(fmt.Sprintf("ensemble:%d", articleID))
		sm.cache.Delete(fmt.Sprintf("bias:%d", articleID))
	}

	// Progress: Success
	if sm.progressMgr != nil {
		ps := &models.ProgressState{Step: "Complete", Message: "Scoring complete", Percent: 100, Status: "Success", FinalScore: &score, LastUpdated: time.Now().Unix()}
		sm.progressMgr.SetProgress(articleID, ps)
	}

	return score, confidence, nil
}

// InvalidateScoreCache invalidates all score-related caches for an article
func (sm *ScoreManager) InvalidateScoreCache(articleID int64) {
	if sm.cache == nil {
		return
	}
	// Invalidate all relevant cache keys (matching API cache usage)
	keys := []string{
		fmt.Sprintf("article:%d", articleID),
		fmt.Sprintf("ensemble:%d", articleID),
		fmt.Sprintf("bias:%d", articleID),
	}
	for _, key := range keys {
		sm.cache.Delete(key)
	}
}

// TrackProgress is a stub for progress tracking
func (sm *ScoreManager) TrackProgress(articleID int64, step, status string) {
	// TODO: Implement progress tracking
}

// SetProgress proxies to ProgressManager
func (sm *ScoreManager) SetProgress(articleID int64, state *models.ProgressState) {
	if sm.progressMgr != nil {
		sm.progressMgr.SetProgress(articleID, state)
	}
}

// GetProgress proxies to ProgressManager
func (sm *ScoreManager) GetProgress(articleID int64) *models.ProgressState {
	if sm.progressMgr != nil {
		return sm.progressMgr.GetProgress(articleID)
	}
	return nil
}
