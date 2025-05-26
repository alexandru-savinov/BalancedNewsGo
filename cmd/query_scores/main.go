package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type LLMScore struct {
	ID        int64   `db:"id"`
	ArticleID int64   `db:"article_id"`
	Model     string  `db:"model"`
	Score     float64 `db:"score"`
	Metadata  string  `db:"metadata"`
}

func main() {
	db, err := sqlx.Open("sqlite", "news.db")
	if err != nil {
		log.Printf("ERROR: Failed to open DB: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close DB connection: %v", err)
		}
	}()

	var scores []LLMScore
	articleID := int64(681)
	err = db.Select(&scores, "SELECT id, article_id, model, score, metadata FROM llm_scores WHERE article_id = ?", articleID)
	if err != nil {
		log.Printf("Error fetching scores for article ID %d: %v", articleID, err)
		if db != nil {
			if closeErr := db.Close(); closeErr != nil {
				log.Printf("Error closing database connection: %v", closeErr)
			}
		}
		os.Exit(1)
	}

	for _, s := range scores {
		fmt.Printf("Model: %s, Score: %.2f, Metadata: %s\n", s.Model, s.Score, s.Metadata)
	}
}
