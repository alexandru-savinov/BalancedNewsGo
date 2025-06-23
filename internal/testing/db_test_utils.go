package testing

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TestDatabase represents a test database instance
type TestDatabase struct {
	DB     *sql.DB
	DBName string
	Driver string
}

// DatabaseTestConfig holds configuration for database tests
type DatabaseTestConfig struct {
	UsePostgres    bool
	UseSQLite      bool
	SQLiteInMemory bool
	MigrationsPath string
	SeedDataPath   string
}

// SetupTestDatabase creates a test database based on configuration
func SetupTestDatabase(t *testing.T, config DatabaseTestConfig) *TestDatabase {
	t.Helper()

	if config.UsePostgres {
		return setupPostgresContainer(t, config)
	}

	return setupSQLiteDatabase(t, config)
}

// setupPostgresContainer is deprecated - use SQLite instead
func setupPostgresContainer(t *testing.T, config DatabaseTestConfig) *TestDatabase {
	t.Helper()
	t.Skip("PostgreSQL testing is deprecated - use setupSQLiteDatabase instead")
	return nil
}

// setupSQLiteDatabase creates an SQLite test database
func setupSQLiteDatabase(t *testing.T, config DatabaseTestConfig) *TestDatabase {
	t.Helper()

	var dsn string
	if config.SQLiteInMemory {
		dsn = ":memory:"
	} else {
		// Create temporary file for SQLite database
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		dsn = dbPath
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("Failed to open SQLite database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping SQLite database: %v", err)
	}

	testDB := &TestDatabase{
		DB:     db,
		DBName: dsn,
		Driver: "sqlite",
	}

	// Run migrations if specified
	if config.MigrationsPath != "" {
		if err := runMigrations(testDB, config.MigrationsPath); err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}
	}

	// Load seed data if specified
	if config.SeedDataPath != "" {
		if err := loadSeedData(testDB, config.SeedDataPath); err != nil {
			t.Fatalf("Failed to load seed data: %v", err)
		}
	}

	return testDB
}

// runMigrations applies database migrations from the specified path
func runMigrations(testDB *TestDatabase, migrationsPath string) error {
	migrationFiles, err := filepath.Glob(filepath.Join(migrationsPath, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("failed to find migration files: %w", err)
	}

	for _, file := range migrationFiles {
		content, err := os.ReadFile(file) // #nosec G304 - file is from test configuration, controlled input
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		if _, err := testDB.DB.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return nil
}

// loadSeedData loads test data from the specified path
func loadSeedData(testDB *TestDatabase, seedDataPath string) error {
	seedFiles, err := filepath.Glob(filepath.Join(seedDataPath, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to find seed files: %w", err)
	}

	for _, file := range seedFiles {
		content, err := os.ReadFile(file) // #nosec G304 - file is from test configuration, controlled input
		if err != nil {
			return fmt.Errorf("failed to read seed file %s: %w", file, err)
		}

		if _, err := testDB.DB.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute seed data %s: %w", file, err)
		}
	}

	return nil
}

// Cleanup closes the database connection and cleans up resources
func (td *TestDatabase) Cleanup() error {
	if td.DB != nil {
		if err := td.DB.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	return nil
}

// GetConnectionString returns the database connection string
func (td *TestDatabase) GetConnectionString() string {
	if td.Driver == "sqlite" {
		return td.DBName
	}
	return ""
}

// Transaction runs a function within a database transaction for testing
func (td *TestDatabase) Transaction(t *testing.T, fn func(*sql.Tx)) {
	t.Helper()

	tx, err := td.DB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			t.Logf("Failed to rollback transaction: %v", err)
		}
	}()

	fn(tx)
}

// ExecuteTestQueries runs a set of test queries and returns results
func (td *TestDatabase) ExecuteTestQueries(t *testing.T, queries []string) []map[string]interface{} {
	t.Helper()

	var results []map[string]interface{}

	for _, query := range queries {
		rows, err := td.DB.Query(query)
		if err != nil {
			t.Fatalf("Failed to execute query '%s': %v", query, err)
		}
		defer func() { _ = rows.Close() }()

		columns, err := rows.Columns()
		if err != nil {
			t.Fatalf("Failed to get columns: %v", err)
		}

		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				row[col] = values[i]
			}
			results = append(results, row)
		}

		if err := rows.Err(); err != nil {
			t.Fatalf("Row iteration error: %v", err)
		}
	}

	return results
}
