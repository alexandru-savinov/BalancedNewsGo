package testing

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestCleanupTestData tests the CleanupTestData function to ensure it works correctly
// and validates the SQL injection fix
func TestCleanupTestData(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Apply test schema
	if err := applyTestSchema(db); err != nil {
		t.Fatalf("Failed to apply test schema: %v", err)
	}

	// Insert test data
	testData := []struct {
		table string
		query string
		args  []interface{}
	}{
		{
			"articles",
			"INSERT INTO articles (title, content, score, created_at) VALUES (?, ?, ?, ?)",
			[]interface{}{"Test Article", "Test content", 0.5, time.Now()},
		},
		{
			"scores",
			"INSERT INTO scores (article_id, score_type, score_value, created_at) VALUES (?, ?, ?, ?)",
			[]interface{}{1, "test", 0.8, time.Now()},
		},
	}

	for _, data := range testData {
		_, err := db.Exec(data.query, data.args...)
		if err != nil {
			t.Fatalf("Failed to insert test data into %s: %v", data.table, err)
		}
	}

	// Verify data exists before cleanup
	var articleCount, scoreCount int
	err = db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&articleCount)
	if err != nil {
		t.Fatalf("Failed to count articles: %v", err)
	}
	err = db.QueryRow("SELECT COUNT(*) FROM scores").Scan(&scoreCount)
	if err != nil {
		t.Fatalf("Failed to count scores: %v", err)
	}

	if articleCount == 0 || scoreCount == 0 {
		t.Fatal("Test data was not inserted correctly")
	}

	// Run cleanup
	err = CleanupTestData(db)
	if err != nil {
		t.Fatalf("CleanupTestData failed: %v", err)
	}

	// Verify data is cleaned up
	err = db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&articleCount)
	if err != nil {
		t.Fatalf("Failed to count articles after cleanup: %v", err)
	}
	err = db.QueryRow("SELECT COUNT(*) FROM scores").Scan(&scoreCount)
	if err != nil {
		t.Fatalf("Failed to count scores after cleanup: %v", err)
	}

	if articleCount != 0 || scoreCount != 0 {
		t.Errorf("Data was not cleaned up correctly. Articles: %d, Scores: %d", articleCount, scoreCount)
	}
}

// TestDeleteFromTableValidation tests the table name validation in deleteFromTable
func TestDeleteFromTableValidation(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Apply test schema
	if err := applyTestSchema(db); err != nil {
		t.Fatalf("Failed to apply test schema: %v", err)
	}

	// Test valid table names
	validTables := []string{"articles", "scores", "test_table", "Table123", "TABLE_NAME"}
	for _, table := range validTables {
		if !isValidTableName(table) {
			t.Errorf("Valid table name '%s' was rejected", table)
		}
	}

	// Test invalid table names
	invalidTables := []string{
		"",                   // empty
		"123table",           // starts with number
		"table-name",         // contains dash
		"table name",         // contains space
		"table;DROP TABLE",   // SQL injection attempt
		"table'OR'1'='1",     // SQL injection attempt
		"table\"OR\"1\"=\"1", // SQL injection attempt
		"table\nname",        // contains newline
		"table\tname",        // contains tab
	}
	for _, table := range invalidTables {
		if isValidTableName(table) {
			t.Errorf("Invalid table name '%s' was accepted", table)
		}
	}

	// Test deleteFromTable with invalid table name should fail
	err = deleteFromTable(db, "invalid;table")
	if err == nil {
		t.Error("deleteFromTable should have failed with invalid table name")
	}
}
