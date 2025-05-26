package main

import (
	"fmt"
	"log"

	appdb "github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/testing"
)

func main() {
	dbPath := "news.db"
	db, err := appdb.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}()

	// Reset specific test IDs (e.g., 1 and any IDs used in failed attempts)
	testIDsToReset := []int64{1}
	fmt.Printf("Resetting DB for article IDs: %v\n", testIDsToReset)
	err = testing.DBReset(db, testIDsToReset)
	if err != nil {
		log.Fatalf("Failed to reset DB: %v", err)
	}
	fmt.Println("DB Reset complete.")
}
