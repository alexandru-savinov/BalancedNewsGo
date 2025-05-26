package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

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
	var articleIDStr string
	flag.StringVar(&articleIDStr, "id", "94", "The ID of the article to query")
	flag.Parse()

	articleID, err := strconv.ParseInt(articleIDStr, 10, 64)
	if err != nil {
		log.Printf("Error: Invalid article ID '%s': %v", articleIDStr, err)
		os.Exit(1)
	}

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
		log.Printf("Invalid article ID: %s\n", os.Args[1])
		func() {
			err := db.Close()
			if err != nil {
				log.Printf("Error closing db: %v", err)
			}
		}()
		os.Exit(1)
	}

	fmt.Printf("ID: %d\nTitle: %s\nContent length: %d\nContent preview: %.100s\n",
		article.ID, article.Title, len(article.Content), article.Content)
}
