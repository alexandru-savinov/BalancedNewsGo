package main

import (
	"fmt"
	"log"

	appdb "github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/testing"
)

func main() {
	fmt.Println("--- Setup Script S3 Started ---")
	dbPath := "news.db"
	fmt.Printf("Attempting to init DB at: %s\n", dbPath)
	db, err := appdb.InitDB(dbPath)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize DB: %v", err)
	}
	fmt.Println("DB Initialized successfully by setup script.")

	// Defer close with checkpoint
	defer func() {
		fmt.Println("Setup script: Attempting to force WAL checkpoint before closing DB...")
		_, cerr := db.Exec("PRAGMA wal_checkpoint(FULL);")
		if cerr != nil {
			log.Printf("Setup script: Warning: Failed to execute wal_checkpoint: %v", cerr)
		} else {
			fmt.Println("Setup script: WAL checkpoint executed.")
		}
		if err := db.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
		fmt.Println("Setup script: DB closed.")
	}()

	// Reset DB for target IDs
	testIDs := []int64{9003, 9004, 9005}
	fmt.Printf("Resetting DB for article IDs: %v\n", testIDs)
	err = testing.DBReset(db, testIDs)
	if err != nil {
		log.Fatalf("FATAL: Failed to reset DB for IDs %v: %v", testIDs, err)
	}
	fmt.Printf("DB reset for article IDs %v successful.\n", testIDs)

	// Article A (Success Case)
	articleA_ID, err := testing.InsertTestArticle(db, "Test Article S3-A (Success)", "Content S3-A")
	if err != nil {
		log.Fatalf("FATAL: Insert S3-A failed: %v", err)
	}
	err = testing.SeedLLMScoresForSuccessfulScore(db, articleA_ID)
	if err != nil {
		log.Fatalf("FATAL: Seed S3-A failed: %v", err)
	}
	fmt.Printf("Setup S3-A (Success Case) complete for Article ID: %d\n", articleA_ID)

	// Article B (ErrAllPerspectivesInvalid Case - requires handle_invalid: ignore)
	articleB_ID, err := testing.InsertTestArticle(db, "Test Article S3-B (AllInvalid)", "Content S3-B")
	if err != nil {
		log.Fatalf("FATAL: Insert S3-B failed: %v", err)
	}
	err = testing.SeedLLMScoresForErrAllPerspectivesInvalid(db, articleB_ID)
	if err != nil {
		log.Fatalf("FATAL: Seed S3-B failed: %v", err)
	}
	fmt.Printf("Setup S3-B (AllInvalid Case) complete for Article ID: %d\n", articleB_ID)

	// Article C (ErrAllScoresZeroConfidence Case)
	articleC_ID, err := testing.InsertTestArticle(db, "Test Article S3-C (ZeroConf)", "Content S3-C")
	if err != nil {
		log.Fatalf("FATAL: Insert S3-C failed: %v", err)
	}
	err = testing.SeedLLMScoresForErrAllScoresZeroConfidence(db, articleC_ID)
	if err != nil {
		log.Fatalf("FATAL: Seed S3-C failed: %v", err)
	}
	fmt.Printf("Setup S3-C (ZeroConf Case) complete for Article ID: %d\n", articleC_ID)

	fmt.Printf("--- Setup Script S3 Finished --- IDs: A=%d, B=%d, C=%d\n", articleA_ID, articleB_ID, articleC_ID)
}
