package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func run(articleID int64) error {

	db, err := sqlx.Open("sqlite", "news.db")
	if err != nil {
		return fmt.Errorf("failed to open DB: %w", err)
	}
	defer func() { _ = db.Close() }()

	res, err := db.Exec(`DELETE FROM llm_scores WHERE article_id=?`, articleID)
	if err != nil {
		return fmt.Errorf("failed to delete mock scores: %w", err)
	}

	count, _ := res.RowsAffected()
	log.Printf("Deleted %d mock scores for article ID %d.", count, articleID)
	return nil
}

func main() {
	articleID := flag.Int64("id", 0, "Article ID (required)")
	flag.Parse()

	if *articleID == 0 {
		log.Printf("ERROR: article ID is required")
		flag.Usage()
		os.Exit(1)
	}

	if err := run(*articleID); err != nil {
		log.Printf("ERROR: %v", err)
		os.Exit(1)
	}
}
