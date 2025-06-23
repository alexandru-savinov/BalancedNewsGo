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
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close database: %v", closeErr)
		}
	}()

	_, err = db.Exec("DELETE FROM llm_scores")
	if err != nil {
		return fmt.Errorf("failed to delete llm_scores: %w", err)
	}

	_, err = db.Exec("DELETE FROM articles")
	if err != nil {
		return fmt.Errorf("failed to delete articles: %w", err)
	}

	log.Println("All articles and bias scores deleted successfully.")
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Printf("ERROR: %v", err)
		os.Exit(1)
	}
}
