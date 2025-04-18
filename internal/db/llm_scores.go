package db

import (
	"time"

	"github.com/jmoiron/sqlx"
)

// LLMScore represents a score from an LLM model
type LLMScore struct {
	ID        int64     `db:"id" json:"id"`
	ArticleID int64     `db:"article_id" json:"article_id"`
	Model     string    `db:"model" json:"model"`
	Score     float64   `db:"score" json:"score"`
	Metadata  string    `db:"metadata" json:"metadata"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// FetchLLMScores retrieves all LLM scores for an article
func FetchLLMScores(db *sqlx.DB, articleID int64) ([]LLMScore, error) {
	var scores []LLMScore
	err := db.Select(&scores, "SELECT * FROM llm_scores WHERE article_id = ? ORDER BY created_at DESC", articleID)
	if err != nil {
		return nil, handleError(err, "failed to fetch LLM scores")
	}
	if len(scores) == 0 {
		return nil, ErrScoreNotFound
	}
	return scores, nil
}

// InsertLLMScore creates a new LLM score entry
func InsertLLMScore(db *sqlx.DB, score *LLMScore) (int64, error) {
	// Verify article exists first
	exists := false
	err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM articles WHERE id = ?)", score.ArticleID)
	if err != nil {
		return 0, handleError(err, "failed to check article existence")
	}
	if !exists {
		return 0, ErrArticleNotFound
	}

	result, err := db.NamedExec(`
		INSERT INTO llm_scores (article_id, model, score, metadata, created_at)
		VALUES (:article_id, :model, :score, :metadata, :created_at)`,
		score)
	if err != nil {
		return 0, handleError(err, "failed to insert LLM score")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, handleError(err, "failed to get inserted score ID")
	}

	return id, nil
}

// DeleteLLMScores removes all scores for an article
func DeleteLLMScores(db *sqlx.DB, articleID int64) error {
	result, err := db.Exec("DELETE FROM llm_scores WHERE article_id = ?", articleID)
	if err != nil {
		return handleError(err, "failed to delete LLM scores")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return ErrScoreNotFound
	}

	return nil
}

// UpdateLLMScore updates an existing score entry
func UpdateLLMScore(db *sqlx.DB, score *LLMScore) error {
	result, err := db.NamedExec(`
		UPDATE llm_scores 
		SET score = :score, metadata = :metadata 
		WHERE id = :id`,
		score)
	if err != nil {
		return handleError(err, "failed to update LLM score")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleError(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return ErrScoreNotFound
	}

	return nil
}

// FetchLatestScoreByModel gets the most recent score for a specific model
func FetchLatestScoreByModel(db *sqlx.DB, articleID int64, model string) (*LLMScore, error) {
	var score LLMScore
	err := db.Get(&score, `
		SELECT * FROM llm_scores 
		WHERE article_id = ? AND model = ? 
		ORDER BY created_at DESC LIMIT 1`,
		articleID, model)
	if err != nil {
		return nil, handleError(err, "failed to fetch latest model score")
	}
	return &score, nil
}
