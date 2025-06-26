package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/testing"
)

func main() {
	// Get database path from environment or use default
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "newsbalancer.db"
	}

	fmt.Printf("Seeding test data into database: %s\n", dbPath)

	// Connect to the database
	database, err := sqlx.Connect("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Verify database connection
	if err := database.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Database connection successful")

	// Seed test articles for accessibility tests
	testArticles := []struct {
		title   string
		content string
	}{
		{
			title:   "Test Article for Accessibility Testing",
			content: "This is test content for accessibility testing. It ensures that H1 elements have proper content and are visible to screen readers.",
		},
		{
			title:   "Test Article 1 for Accessibility",
			content: "Test content for article 1 to ensure proper page rendering.",
		},
		{
			title:   "Test Article 2 for Accessibility",
			content: "Test content for article 2 to ensure proper page rendering.",
		},
		{
			title:   "Test Article 3 for Accessibility",
			content: "Test content for article 3 to ensure proper page rendering.",
		},
	}

	fmt.Printf("Inserting %d test articles...\n", len(testArticles))

	var insertedIDs []int64
	for i, article := range testArticles {
		articleID, err := testing.InsertTestArticle(database, article.title, article.content)
		if err != nil {
			log.Fatalf("Failed to insert test article %d: %v", i+1, err)
		}
		insertedIDs = append(insertedIDs, articleID)
		fmt.Printf("✓ Inserted article %d with ID: %d, Title: %s\n", i+1, articleID, article.title)
	}

	// Verify articles were inserted by counting total articles
	var count int
	err = database.Get(&count, "SELECT COUNT(*) FROM articles")
	if err != nil {
		log.Fatalf("Failed to count articles: %v", err)
	}

	fmt.Printf("✓ Database now contains %d total articles\n", count)

	// Verify specific articles exist and have content
	for _, id := range insertedIDs {
		var article db.Article
		err = database.Get(&article, "SELECT id, title, content FROM articles WHERE id = ?", id)
		if err != nil {
			log.Fatalf("Failed to verify article %d: %v", id, err)
		}
		if article.Title == "" {
			log.Fatalf("Article %d has empty title", id)
		}
		if article.Content == "" {
			log.Fatalf("Article %d has empty content", id)
		}
		fmt.Printf("✓ Verified article %d: title='%s', content_length=%d\n", id, article.Title, len(article.Content))
	}

	fmt.Println("✅ Test data seeding completed successfully!")
	fmt.Printf("Inserted article IDs: %v\n", insertedIDs)
}
