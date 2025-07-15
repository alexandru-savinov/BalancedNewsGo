// Package testing provides test utilities for database operations
package testing

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// DatabaseContainer wraps database functionality for testing
type DatabaseContainer struct {
	DB      *sql.DB
	ConnStr string
}

// NewSQLiteTestDatabase creates an in-memory SQLite database for testing
func NewSQLiteTestDatabase(t *testing.T) *DatabaseContainer {
	t.Helper()

	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite test database: %v", err)
	}

	// Apply migrations/schema
	if err := applyTestSchema(db); err != nil {
		_ = db.Close()
		t.Fatalf("Failed to apply test schema: %v", err)
	}

	return &DatabaseContainer{
		DB:      db,
		ConnStr: ":memory:",
	}
}

// NewPostgresTestDatabase is deprecated - use SQLite instead
// This function is kept for backward compatibility but should not be used
func NewPostgresTestDatabase(t *testing.T, ctx context.Context) *DatabaseContainer {
	t.Helper()
	t.Skip("PostgreSQL testing is deprecated - use NewSQLiteTestDatabase instead")
	return nil
}

// Close cleans up the database resources
func (dc *DatabaseContainer) Close(ctx context.Context) error {
	if dc.DB != nil {
		return dc.DB.Close()
	}
	return nil
}

// SeedTestData loads test data into the database
func SeedTestData(db *sql.DB, seedFilePath string) error {
	if seedFilePath == "" {
		return nil // No seed file specified
	}

	content, err := os.ReadFile(seedFilePath) // #nosec G304 - seedFilePath is from test configuration, controlled input
	if err != nil {
		return fmt.Errorf("failed to read seed file %s: %w", seedFilePath, err)
	}

	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute seed data: %w", err)
	}

	return nil
}

// CleanupTestData removes test data from the database
func CleanupTestData(db *sql.DB) error {
	// Get all table names
	rows, err := db.Query(`
        SELECT name FROM sqlite_master 
        WHERE type='table' AND name NOT LIKE 'sqlite_%'
    `)
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Delete data from all tables (using whitelist for security)
	allowedTables := map[string]bool{
		"articles":        true,
		"llm_scores":      true,
		"feedback":        true,
		"sources":         true,
		"source_errors":   true,
		"scores":          true, // Test table for score data
		"users":           true, // Test table for user data
		"sqlite_sequence": true,
	}

	for _, table := range tables {
		// Skip system tables and validate against whitelist
		if table == "sqlite_master" || table == "sqlite_temp_master" {
			continue
		}
		if !allowedTables[table] {
			log.Printf("Warning: Skipping unknown table '%s' for security", table)
			continue
		}

		// Execute DELETE with proper table name validation and quoting
		// Table names cannot be parameterized in SQL, but we validate against whitelist
		// and use proper SQL identifier quoting to prevent injection
		if err := deleteFromTable(db, table); err != nil {
			return fmt.Errorf("failed to delete from table %s: %w", table, err)
		}
	}

	return nil
}

// deleteFromTable safely deletes all data from a specific table
// This function isolates the SQL construction to address SonarQube security concerns
// while maintaining proper validation and quoting
func deleteFromTable(db *sql.DB, tableName string) error {
	// Additional validation: ensure table name contains only valid characters
	// This prevents any potential injection even though we already validate against whitelist
	if !isValidTableName(tableName) {
		return fmt.Errorf("invalid table name format: %s", tableName)
	}

	// Use proper SQL identifier quoting for SQLite
	// Double quotes are the standard SQL way to quote identifiers
	quotedTable := `"` + tableName + `"`
	query := "DELETE FROM " + quotedTable

	_, err := db.Exec(query)
	return err
}

// isValidTableName validates that a table name contains only safe characters
// This provides an additional security layer beyond the whitelist
func isValidTableName(name string) bool {
	// Allow only alphanumeric characters, underscores, and reasonable length
	if len(name) == 0 || len(name) > 64 {
		return false
	}

	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}

	// Ensure it doesn't start with a number
	firstChar := rune(name[0])
	return (firstChar >= 'a' && firstChar <= 'z') ||
		(firstChar >= 'A' && firstChar <= 'Z') ||
		firstChar == '_'
}

// SeedTestData inserts test fixtures into the database
func (dc *DatabaseContainer) SeedTestData(t *testing.T) {
	t.Helper()

	fixtures := []struct {
		query string
		args  []interface{}
	}{
		{
			"INSERT INTO articles (title, content, score, created_at) VALUES (?, ?, ?, ?)",
			[]interface{}{"Test Article 1", "Test content for article 1", 0.85, time.Now()},
		},
		{
			"INSERT INTO articles (title, content, score, created_at) VALUES (?, ?, ?, ?)",
			[]interface{}{"Test Article 2", "Test content for article 2", 0.72, time.Now()},
		},
		{
			"INSERT INTO articles (title, content, score, created_at) VALUES (?, ?, ?, ?)",
			[]interface{}{"Test Article 3", "Test content for article 3", 0.91, time.Now()},
		},
	}

	for _, fixture := range fixtures {
		_, err := dc.DB.Exec(fixture.query, fixture.args...)
		if err != nil {
			t.Fatalf("Failed to seed test data: %v", err)
		}
	}
}

// CleanupTestData removes test data from the database
func (dc *DatabaseContainer) CleanupTestData(t *testing.T) {
	t.Helper()

	cleanupQueries := []string{
		"DELETE FROM scores WHERE article_id IN (SELECT id FROM articles WHERE title LIKE 'Test Article%')",
		"DELETE FROM articles WHERE title LIKE 'Test Article%'",
		"DELETE FROM users WHERE email LIKE 'test%@example.com'",
	}

	for _, query := range cleanupQueries {
		_, err := dc.DB.Exec(query)
		if err != nil {
			t.Logf("Warning: Failed to cleanup with query '%s': %v", query, err)
		}
	}
}

// applyTestSchema applies the database schema for testing
func applyTestSchema(db *sql.DB) error {
	// Read and apply schema from schema.sql or embedded schema
	schema := `
    CREATE TABLE IF NOT EXISTS articles (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        content TEXT,
        score REAL DEFAULT 0.0,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS scores (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        article_id INTEGER NOT NULL,
        score_type TEXT NOT NULL,
        score_value REAL NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (article_id) REFERENCES articles(id)
    );
    
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        email TEXT UNIQUE NOT NULL,
        name TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    `

	_, err := db.Exec(schema)
	return err
}

// TestDatabaseWrapper provides common test database functionality
type TestDatabaseWrapper struct {
	*testing.T
	DB      *DatabaseContainer
	Cleanup func()
}

// NewTestDatabase creates a test database with automatic cleanup
func NewTestDatabase(t *testing.T, usePostgres bool) *TestDatabaseWrapper {
	ctx := context.Background()

	var db *DatabaseContainer
	if usePostgres {
		db = NewPostgresTestDatabase(t, ctx)
	} else {
		db = NewSQLiteTestDatabase(t)
	}

	// Setup automatic cleanup
	cleanup := func() {
		if err := db.Close(ctx); err != nil {
			t.Logf("Warning: Failed to close test database: %v", err)
		}
	}

	t.Cleanup(cleanup)

	return &TestDatabaseWrapper{
		T:       t,
		DB:      db,
		Cleanup: cleanup,
	}
}
