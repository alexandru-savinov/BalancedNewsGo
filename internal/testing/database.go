// Package testing provides test utilities for database operations
package testing

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/mattn/go-sqlite3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DatabaseContainer wraps testcontainers functionality for database testing
type DatabaseContainer struct {
	Container testcontainers.Container
	DB        *sql.DB
	ConnStr   string
}

// NewSQLiteTestDatabase creates an in-memory SQLite database for testing
func NewSQLiteTestDatabase(t *testing.T) *DatabaseContainer {
	t.Helper()

	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite test database: %v", err)
	}

	// Apply migrations/schema
	if err := applyTestSchema(db); err != nil {
		db.Close()
		t.Fatalf("Failed to apply test schema: %v", err)
	}

	return &DatabaseContainer{
		Container: nil, // No container for SQLite
		DB:        db,
		ConnStr:   ":memory:",
	}
}

// NewPostgresTestDatabase creates a containerized PostgreSQL database for testing
func NewPostgresTestDatabase(t *testing.T, ctx context.Context) *DatabaseContainer {
	t.Helper()

	// Create PostgreSQL container with testcontainers
	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"), testcontainers.WithWaitStrategy(
			wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
				return fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())
			}).WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL test container: %v", err)
	}

	// Get connection string
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		container.Terminate(ctx)
		t.Fatalf("Failed to ping PostgreSQL: %v", err)
	}

	// Apply test schema
	if err := applyTestSchema(db); err != nil {
		db.Close()
		container.Terminate(ctx)
		t.Fatalf("Failed to apply test schema: %v", err)
	}

	return &DatabaseContainer{
		Container: container,
		DB:        db,
		ConnStr:   connStr,
	}
}

// Close cleans up the database resources
func (dc *DatabaseContainer) Close(ctx context.Context) error {
	if dc.DB != nil {
		dc.DB.Close()
	}

	if dc.Container != nil {
		return dc.Container.Terminate(ctx)
	}

	return nil
}

// SeedTestData loads test data into the database
func SeedTestData(db *sql.DB, seedFilePath string) error {
	if seedFilePath == "" {
		return nil // No seed file specified
	}

	content, err := os.ReadFile(seedFilePath)
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
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Delete data from all tables
	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return fmt.Errorf("failed to delete from table %s: %w", table, err)
		}
	}

	return nil
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
