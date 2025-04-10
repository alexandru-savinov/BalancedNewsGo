package main

import (
	"fmt"
	"log"

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
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close DB connection: %v", err)
		}
	}()

	var scores []LLMScore
	err = db.Select(&scores, "SELECT id, article_id, model, score, metadata FROM llm_scores WHERE article_id = ?", 681)
	if err != nil {
		log.Fatalf("Failed to fetch scores: %v", err)
	}

	for _, s := range scores {
		fmt.Printf("Model: %s, Score: %.2f, Metadata: %s\n", s.Model, s.Score, s.Metadata)
	}
}
