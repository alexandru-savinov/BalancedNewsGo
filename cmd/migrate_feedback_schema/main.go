package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sqlx.Open("sqlite3", "news.db")
	if err != nil {
		log.Printf("ERROR: Failed to open database: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close DB connection: %v", err)
		}
	}()

	// Add 'category' column
	if !columnExists(db, "feedback", "category") {
		_, err = db.Exec(`ALTER TABLE feedback ADD COLUMN category TEXT`)
		if err != nil {
			log.Printf("ERROR: Failed to add 'category' column: %v", err)
			func() {
				err := db.Close()
				if err != nil {
					log.Printf("Error closing db: %v", err)
				}
			}()
			os.Exit(1)
		}
		fmt.Println("Added 'category' column")
	} else {
		fmt.Println("'category' column already exists")
	}

	// Add 'ensemble_output_id' column
	if !columnExists(db, "feedback", "ensemble_output_id") {
		_, err = db.Exec(`ALTER TABLE feedback ADD COLUMN ensemble_output_id INTEGER`)
		if err != nil {
			log.Printf("ERROR: Failed to add 'ensemble_output_id' column: %v", err)
			func() {
				err := db.Close()
				if err != nil {
					log.Printf("Error closing db: %v", err)
				}
			}()
			os.Exit(1)
		}
		fmt.Println("Added 'ensemble_output_id' column")
	} else {
		fmt.Println("'ensemble_output_id' column already exists")
	}

	// Add 'source' column
	if !columnExists(db, "feedback", "source") {
		_, err = db.Exec(`ALTER TABLE feedback ADD COLUMN source TEXT`)
		if err != nil {
			log.Printf("ERROR: Failed to add 'source' column: %v", err)
			func() {
				err := db.Close()
				if err != nil {
					log.Printf("Error closing db: %v", err)
				}
			}()
			os.Exit(1)
		}
		fmt.Println("Added 'source' column")
	} else {
		fmt.Println("'source' column already exists")
	}
}

func columnExists(db *sqlx.DB, tableName, columnName string) bool {
	var count int
	query := `
		SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?;
	`
	err := db.Get(&count, query, tableName, columnName)
	if err != nil {
		log.Printf("ERROR: Failed to check column existence: %v", err)
		os.Exit(1)
	}
	return count > 0
}
