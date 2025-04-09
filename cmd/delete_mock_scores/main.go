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

	res, err := db.Exec(`DELETE FROM llm_scores WHERE metadata LIKE '%"mock": true%'`)
	if err != nil {
		log.Fatalf("Failed to delete mock scores: %v", err)
	}

	count, _ := res.RowsAffected()
	log.Printf("Deleted %d mock scores.", count)
}
