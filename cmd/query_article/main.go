package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Article struct {
	ID      int64  `db:"id"`
	Title   string `db:"title"`
	Content string `db:"content"`
}

func main() {
	// Parse command line flags
	var articleID int
	flag.IntVar(&articleID, "id", 94, "The ID of the article to query")
	flag.Parse()

	// Use database in root directory
	dbPath := filepath.Join("../..", "news.db")
	log.Printf("Connecting to database at: %s", dbPath)

	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		log.Printf("ERROR: Failed to open DB: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close DB connection: %v", err)
		}
	}()

	var article Article
	err = db.Get(&article, "SELECT id, title, content FROM articles WHERE id = ?", articleID)
	if err != nil {
		log.Printf("ERROR: Failed to fetch article: %v", err)
		os.Exit(1)
	}

	fmt.Printf("ID: %d\nTitle: %s\nContent length: %d\nContent preview: %.100s\n",
		article.ID, article.Title, len(article.Content), article.Content)
}
