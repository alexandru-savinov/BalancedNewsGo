package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

func main() {
	dbPath := flag.String("db", "news.db", "Path to SQLite database")
	articleID := flag.Int64("article_id", 0, "Article ID")
	userID := flag.String("user_id", "", "User ID or source identifier")
	feedbackText := flag.String("feedback_text", "", "Feedback text")
	category := flag.String("category", "", "Feedback category (agree, disagree, unclear, other)")
	ensembleOutputID := flag.Int64("ensemble_output_id", 0, "Ensemble output ID (optional)")
	source := flag.String("source", "api", "Source of feedback (api, email, form)")

	flag.Parse()

	if *articleID == 0 || *feedbackText == "" || *category == "" {
		log.Printf("ERROR: Missing required fields: article_id, feedback_text, category")
		os.Exit(1)
	}

	dbConn, err := sqlx.Open("sqlite", *dbPath)
	if err != nil {
		log.Printf("ERROR: Failed to open database: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			log.Printf("Failed to close DB connection: %v", err)
		}
	}()

	var ensemblePtr *int64
	if *ensembleOutputID != 0 {
		ensemblePtr = ensembleOutputID
	}

	feedback := db.Feedback{
		ArticleID:        *articleID,
		UserID:           *userID,
		FeedbackText:     *feedbackText,
		Category:         *category,
		EnsembleOutputID: ensemblePtr,
		Source:           *source,
		CreatedAt:        time.Now(),
	}

	// Store the feedback
	err = db.InsertFeedback(dbConn, &feedback)
	if err != nil {
		log.Printf("ERROR: Failed to insert feedback: %v", err)
		os.Exit(1)
	}

	fmt.Println("Feedback ingested successfully")
}
