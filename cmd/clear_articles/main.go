package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sqlx.Open("sqlite", "news.db")
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM llm_scores")
	if err != nil {
		log.Fatalf("Failed to delete llm_scores: %v", err)
	}

	_, err = db.Exec("DELETE FROM articles")
	if err != nil {
		log.Fatalf("Failed to delete articles: %v", err)
	}

	log.Println("All articles and bias scores deleted successfully.")
}
