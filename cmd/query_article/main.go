package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Article struct {
	ID      int64  `db:"id"`
	Title   string `db:"title"`
	Content string `db:"content"`
}

func main() {
	db, err := sqlx.Open("sqlite", "news.db")
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	var article Article
	err = db.Get(&article, "SELECT id, title, content FROM articles WHERE id = ?", 94)
	if err != nil {
		log.Fatalf("Failed to fetch article: %v", err)
	}

	fmt.Printf("ID: %d\nTitle: %s\nContent length: %d\nContent preview: %.100s\n",
		article.ID, article.Title, len(article.Content), article.Content)
}
