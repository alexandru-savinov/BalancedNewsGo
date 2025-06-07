package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sqlx.Open("sqlite", "news.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM articles")
	if err != nil {
		log.Fatalf("Failed to get article count: %v", err)
	}

	fmt.Printf("Total articles in database: %d\n", count)

	if count > 0 {
		// Get the first article
		var article struct {
			ID    int64  `db:"id"`
			Title string `db:"title"`
			URL   string `db:"url"`
		}
		err = db.Get(&article, "SELECT id, title, url FROM articles LIMIT 1")
		if err != nil {
			log.Fatalf("Failed to get first article: %v", err)
		}
		fmt.Printf("First article: ID=%d, Title=%s, URL=%s\n", article.ID, article.Title, article.URL)
	}
}
