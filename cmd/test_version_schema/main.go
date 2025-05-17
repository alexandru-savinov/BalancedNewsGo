package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Initialize DB
	dbConn, err := sqlx.Open("sqlite", "news.db")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer dbConn.Close()

	// Get article ID from command line or use a default
	articleID := int64(4202)
	if len(os.Args) > 1 {
		id, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err != nil {
			log.Fatalf("Invalid article ID: %v", err)
		}
		articleID = id
	}

	// Create a test score with explicit version as integer
	testScore := &db.LLMScore{
		ArticleID: articleID,
		Model:     "test-model",
		Score:     0.5,
		Metadata:  `{"confidence": 0.9, "explanation": "Test score"}`,
		CreatedAt: time.Now(),
		Version:   1, // Explicitly set to integer 1
	}

	// Insert score using NamedExec to test fields - matching the existing constraint
	result, err := dbConn.NamedExec(`
		INSERT INTO llm_scores (article_id, model, score, metadata, version, created_at)
		VALUES (:article_id, :model, :score, :metadata, :version, :created_at)`,
		testScore)

	if err != nil {
		log.Fatalf("Failed to insert test score: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()
	fmt.Printf("Insert successful: ID=%d, Rows affected=%d\n", lastID, rowsAffected)

	// Query to verify what was inserted
	var scores []db.LLMScore
	err = dbConn.Select(&scores, "SELECT * FROM llm_scores WHERE article_id = ? AND model = ?",
		articleID, "test-model")

	if err != nil {
		log.Fatalf("Failed to fetch scores: %v", err)
	}

	fmt.Printf("Found %d scores:\n", len(scores))
	for i, s := range scores {
		fmt.Printf("  Score[%d]: Model=%s, Score=%.2f, Version=%d, CreatedAt=%v\n",
			i, s.Model, s.Score, s.Version, s.CreatedAt)
	}

	// Also test ensemble score insertion
	ensembleScore := &db.LLMScore{
		ArticleID: articleID,
		Model:     "ensemble",
		Score:     0.25,
		Metadata:  `{"confidence": 0.85, "final_aggregation": {"weighted_mean": 0.25}}`,
		CreatedAt: time.Now(),
		Version:   1, // Explicitly set to integer 1
	}

	result, err = dbConn.NamedExec(`
		INSERT INTO llm_scores (article_id, model, score, metadata, version, created_at)
		VALUES (:article_id, :model, :score, :metadata, :version, :created_at)`,
		ensembleScore)

	if err != nil {
		log.Fatalf("Failed to insert ensemble score: %v", err)
	}

	rowsAffected, _ = result.RowsAffected()
	lastID, _ = result.LastInsertId()
	fmt.Printf("Ensemble insert successful: ID=%d, Rows affected=%d\n", lastID, rowsAffected)
}
