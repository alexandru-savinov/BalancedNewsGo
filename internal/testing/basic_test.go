package testing

import (
	"testing"
)

// Test that the testing utilities compile and basic functions work
func TestBasicFunctionality(t *testing.T) {
	t.Run("SQLite Database Setup", func(t *testing.T) {
		config := DatabaseTestConfig{
			UseSQLite:      true,
			SQLiteInMemory: true,
		}

		testDB := SetupTestDatabase(t, config)
		if testDB == nil {
			t.Fatal("SetupTestDatabase returned nil")
		}
		defer testDB.Cleanup()

		// Test basic connection
		if testDB.DB == nil {
			t.Fatal("Database connection is nil")
		}

		// Test ping
		if err := testDB.DB.Ping(); err != nil {
			t.Fatalf("Database ping failed: %v", err)
		}

		// Test connection string
		connStr := testDB.GetConnectionString()
		if connStr == "" {
			t.Error("Connection string is empty")
		}

		t.Log("SQLite database test passed")
	})

	t.Run("Server Config", func(t *testing.T) {
		config := DefaultTestServerConfig()

		if config.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", config.Port)
		}

		if config.HealthEndpoint != "/healthz" {
			t.Errorf("Expected health endpoint '/healthz', got '%s'", config.HealthEndpoint)
		}

		t.Log("Server configuration test passed")
	})

	t.Run("API Test Suite", func(t *testing.T) {
		suite := NewAPITestSuite("http://localhost:8080")

		if suite == nil {
			t.Fatal("NewAPITestSuite returned nil")
		}

		if suite.BaseURL != "http://localhost:8080" {
			t.Errorf("Expected base URL 'http://localhost:8080', got '%s'", suite.BaseURL)
		}

		t.Log("API test suite creation passed")
	})
}
