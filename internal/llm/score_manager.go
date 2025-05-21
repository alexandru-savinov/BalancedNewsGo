package llm

import (
	"errors"
	"fmt"
	"log"
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

// UpdateArticleScore computes and stores a composite score for an article based on LLM scores
func (sm *ScoreManager) UpdateArticleScore(articleID int64, scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
	// First, check if all responses have zero confidence
	if allZeros, errZeroConf := checkForAllZeroResponses(scores); allZeros {
		log.Printf("[ERROR] ArticleID %d: All LLMs returned zero confidence - this is a serious error: %v", articleID, errZeroConf)
		// Set an error progress state
		sm.SetProgress(articleID, &models.ProgressState{
			Step:        "Error",
			Message:     "All LLMs returned zero confidence - scoring failed",
			Status:      "Error",
			Error:       fmt.Sprintf("All LLMs returned zero confidence - this indicates a serious issue with the LLM responses: %v", errZeroConf),
			LastUpdated: time.Now().Unix(),
		})
		// Persist final status to DB
		if dbErr := db.UpdateArticleStatus(sm.db, articleID, models.ArticleStatusFailedZeroConf); dbErr != nil {
			log.Printf("[ERROR] ScoreManager: ArticleID %d: Failed to update article status to %s after zero confidence error: %v", articleID, models.ArticleStatusFailedZeroConf, dbErr)
		}
		// Return the error without modifying the score
		return 0, 0, fmt.Errorf("all LLMs returned zero confidence - this indicates a serious issue with the LLM responses: %w", errZeroConf)
	}

	// Use the score calculator to compute the score and confidence, passing the config
	cfg.ArticleIDForDebug = articleID // Set the ID for logging within calculation
	compositeScore, confidence, errCalc := sm.calculator.CalculateScore(scores, cfg)
	cfg.ArticleIDForDebug = 0 // Reset after use (optional, good practice)
	if errCalc != nil {
		// Check for the specific "all invalid" error
		if errors.Is(errCalc, ErrAllPerspectivesInvalid) {
			log.Printf("[ERROR] ScoreManager: ArticleID %d: %v. Score will not be updated.", articleID, errCalc)
			// Update progress to reflect the error state
			errorState := models.ProgressState{
				Step:        "Error",
				Message:     errCalc.Error(),
				Status:      "Error",
				Error:       errCalc.Error(),
				Percent:     100,
				LastUpdated: time.Now().Unix(),
			}
			if sm.progressMgr != nil {
				sm.progressMgr.SetProgress(articleID, &errorState)
			}
			// Persist final status to DB
			if dbErr := db.UpdateArticleStatus(sm.db, articleID, models.ArticleStatusFailedAllInvalid); dbErr != nil {
				log.Printf("[ERROR] ScoreManager: ArticleID %d: Failed to update article status to %s after all invalid error: %v", articleID, models.ArticleStatusFailedAllInvalid, dbErr)
			} else {
				log.Printf("[DEBUG] ScoreManager: ArticleID %d: Successfully updated status to %s.", articleID, models.ArticleStatusFailedAllInvalid)
			}
			// IMPORTANT: Do NOT proceed to update the DB score. Return the error.
			log.Printf("[DEBUG] ScoreManager: ArticleID %d: Returning ErrAllPerspectivesInvalid error now.", articleID)
			return 0, 0, errCalc
		} else {
			// Handle other, unexpected errors from CalculateScore
			log.Printf("[ERROR] ScoreManager: ArticleID %d: Unexpected error calculating score: %v. Score will not be updated.", articleID, errCalc)
			// Update progress similarly
			errorState := models.ProgressState{
				Step:        "Error",
				Message:     fmt.Sprintf("Internal error calculating score: %v", errCalc),
				Status:      "Error",
				Percent:     100,
				LastUpdated: time.Now().Unix(),
			}
			if sm.progressMgr != nil {
				sm.progressMgr.SetProgress(articleID, &errorState)
			}
			// Persist final status to DB
			if dbErr := db.UpdateArticleStatus(sm.db, articleID, models.ArticleStatusFailedError); dbErr != nil {
				log.Printf("[ERROR] ScoreManager: ArticleID %d: Failed to update article status to %s after calculation error: %v", articleID, models.ArticleStatusFailedError, dbErr)
			}
			return 0, 0, errCalc
		}
	}

	// Update the article score in the database
	errDbUpdate := db.UpdateArticleScoreLLM(sm.db, articleID, compositeScore, confidence)
	if errDbUpdate != nil {
		log.Printf("[ERROR] Failed to update article score: %v", errDbUpdate)
		sm.SetProgress(articleID, &models.ProgressState{
			Step:        "Error",
			Message:     "Failed to update article score in database",
			Status:      "Error",
			Error:       errDbUpdate.Error(),
			LastUpdated: time.Now().Unix(),
		})
		// Persist final status to DB even if score update failed
		if dbStatusErr := db.UpdateArticleStatus(sm.db, articleID, models.ArticleStatusFailedError); dbStatusErr != nil {
			log.Printf("[ERROR] ScoreManager: ArticleID %d: Failed to update article status to %s after DB score update error: %v", articleID, models.ArticleStatusFailedError, dbStatusErr)
		}
		return 0, 0, fmt.Errorf("failed to store score: %w", errDbUpdate)
	}

	// If score update was successful, also update status to Scored
	if dbErr := db.UpdateArticleStatus(sm.db, articleID, models.ArticleStatusScored); dbErr != nil {
		log.Printf("[ERROR] ScoreManager: ArticleID %d: Failed to update article status to %s after successful score update: %v", articleID, models.ArticleStatusScored, dbErr)
		// Note: The main operation (score update) succeeded, so we don't return this error, just log it.
	}

	// Invalidate cache
	sm.InvalidateScoreCache(articleID)

	// Update progress
	successState := models.ProgressState{
		Step:        "Complete",
		Message:     "Analysis complete.",
		Status:      "Success",
		Percent:     100,
		FinalScore:  &compositeScore,
		LastUpdated: time.Now().Unix(),
	}
	if sm.progressMgr != nil {
		sm.progressMgr.SetProgress(articleID, &successState)
	}
	log.Printf("[INFO] ScoreManager: ArticleID %d: Score updated successfully, status set to %s.", articleID, models.ArticleStatusScored)
	return compositeScore, confidence, nil
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
		fmt.Sprintf("summary:%d", articleID),
	}
	for _, key := range keys {
		sm.cache.Delete(key)
	}
}

// TrackProgress registers progress tracking for an article
func (sm *ScoreManager) TrackProgress(articleID int64, step, status string) {
	if sm.progressMgr != nil {
		// Create an initial progress state with parameters
		initialState := &models.ProgressState{
			Step:        step,
			Message:     fmt.Sprintf("Progress update: %s", step),
			Percent:     0,
			Status:      status,
			LastUpdated: time.Now().Unix(),
		}
		sm.progressMgr.SetProgress(articleID, initialState)
	}
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
