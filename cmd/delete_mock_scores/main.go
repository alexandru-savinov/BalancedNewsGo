package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func run() error {
	db, err := sqlx.Open("sqlite", "news.db")
	if err != nil {
		return fmt.Errorf("failed to open DB: %w", err)
	}
	defer func() { _ = db.Close() }()

	res, err := db.Exec(`DELETE FROM llm_scores WHERE article_id=133`)
	if err != nil {
		return fmt.Errorf("failed to delete mock scores: %w", err)
	}

	count, _ := res.RowsAffected()
	log.Printf("Deleted %d mock scores.", count)
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Printf("ERROR: %v", err)
		os.Exit(1)
	}
}
