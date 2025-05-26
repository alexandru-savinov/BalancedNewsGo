package testing

import (
	"fmt"
	"math"
	"time"

	"database/sql"

	"log"

	appdb "github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/jmoiron/sqlx"
)

// DBReset clears test-specific data from the database for a given set of article IDs.
func DBReset(db *sqlx.DB, testArticleIDs []int64) error {
	if len(testArticleIDs) == 0 {
		return nil // Nothing to do
	}

	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("DBReset: failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone && err != sql.ErrConnDone {
			log.Printf("Error rolling back transaction in DBResetForTestIDs: %v", err)
		}
	}()

	// Prepare queries - using IN clause for efficiency if supported and not too many IDs.
	// For simplicity with variable number of IDs and to ensure order, loop and delete.

	queries := []string{
		"DELETE FROM llm_scores WHERE article_id = ?",
		"DELETE FROM feedback WHERE article_id = ?",
		"DELETE FROM articles WHERE id = ?",
	}

	for _, queryTemplate := range queries {
		for _, articleID := range testArticleIDs {
			_, err := tx.Exec(queryTemplate, articleID)
			if err != nil {
				return fmt.Errorf("DBReset: failed to execute query '%s' for article ID %d: %w", queryTemplate, articleID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("DBReset: failed to commit transaction: %w", err)
	}
	return nil
}

// InsertTestArticle inserts a basic article for testing and returns its ID.
func InsertTestArticle(db *sqlx.DB, title, content string) (int64, error) {
	dummyURL := fmt.Sprintf("http://test.example.com/article/%d", time.Now().UnixNano())
	dummySource := "http://test.example.com/feed"
	pubDate := time.Now()

	testArticle := &appdb.Article{
		URL:     dummyURL,
		Source:  dummySource,
		Title:   title,
		Content: content,
		PubDate: pubDate,
	}

	articleID, err := appdb.InsertArticle(db, testArticle)
	if err != nil {
		return 0, fmt.Errorf("InsertTestArticle: failed to insert article: %w", err)
	}
	return articleID, nil
}

// SeedLLMScoresForErrAllPerspectivesInvalid seeds llm_scores to trigger ErrAllPerspectivesInvalid.
// Assumes config.HandleInvalid is "ignore" or similar for Inf scores.
func SeedLLMScoresForErrAllPerspectivesInvalid(db *sqlx.DB, articleID int64) error {
	perspectives := []struct {
		modelName string
	}{
		{"meta-llama/llama-4-maverick"}, // Left
		{"google/gemini-2.0-flash-001"}, // Center
		{"openai/gpt-4.1-nano"},         // Right
	}

	for _, p := range perspectives {
		scoreRecord := &appdb.LLMScore{
			ArticleID: articleID,
			Model:     p.modelName,
			Score:     math.Inf(1), // Use +Infinity instead of NaN
			Metadata:  `{"confidence": 0.1}`,
			Version:   1,
			CreatedAt: time.Now(),
		}
		_, err := appdb.InsertLLMScore(db, scoreRecord)
		if err != nil {
			return fmt.Errorf("SeedLLMScoresForErrAllPerspectivesInvalid: failed to insert Inf score for model %s: %w", p.modelName, err)
		}
	}
	return nil
}

// SeedLLMScoresForErrAllScoresZeroConfidence seeds llm_scores to trigger ErrAllScoresZeroConfidence.
// This means all non-ensemble scores have metadata indicating zero confidence.
func SeedLLMScoresForErrAllScoresZeroConfidence(db *sqlx.DB, articleID int64) error {
	modelsToSeed := []struct {
		modelName string
		scoreVal  float64
	}{
		{"meta-llama/llama-4-maverick", 0.1}, // Left
		{"google/gemini-2.0-flash-001", 0.2}, // Center
		{"openai/gpt-4.1-nano", -0.1},        // Right
	}

	zeroConfidenceMetadata := `{"confidence": 0.0}`

	for _, m := range modelsToSeed {
		scoreRecord := &appdb.LLMScore{
			ArticleID: articleID,
			Model:     m.modelName,
			Score:     m.scoreVal,
			Metadata:  zeroConfidenceMetadata,
			Version:   1,
			CreatedAt: time.Now(),
		}
		_, err := appdb.InsertLLMScore(db, scoreRecord)
		if err != nil {
			return fmt.Errorf("SeedLLMScoresForErrAllScoresZeroConfidence: failed to insert zero-confidence score "+
				"for model %s: %w", m.modelName, err)
		}
	}
	return nil
}

// SeedLLMScoresForSuccessfulScore seeds llm_scores that should lead to a successful score calculation.
func SeedLLMScoresForSuccessfulScore(db *sqlx.DB, articleID int64) error {
	successfulScores := []struct {
		modelName string
		scoreVal  float64
		conf      float64
	}{
		{"meta-llama/llama-4-maverick", -0.6, 0.9}, // Left
		{"google/gemini-2.0-flash-001", 0.1, 0.85}, // Center
		{"openai/gpt-4.1-nano", 0.7, 0.92},         // Right
	}

	for _, s := range successfulScores {
		metadata := fmt.Sprintf(`{"confidence": %.2f}`, s.conf)
		scoreRecord := &appdb.LLMScore{
			ArticleID: articleID,
			Model:     s.modelName,
			Score:     s.scoreVal,
			Metadata:  metadata,
			Version:   1,
			CreatedAt: time.Now(),
		}
		_, err := appdb.InsertLLMScore(db, scoreRecord)
		if err != nil {
			return fmt.Errorf("SeedLLMScoresForSuccessfulScore: failed to insert successful score for model %s: %w", s.modelName, err)
		}
	}
	return nil
}
