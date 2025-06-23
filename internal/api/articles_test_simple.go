package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testingutils "github.com/alexandru-savinov/BalancedNewsGo/internal/testing"
)

// Simple test using the new database testing infrastructure
func TestArticlesAPISimple(t *testing.T) {
	// Create test database with automatic cleanup
	config := testingutils.DatabaseTestConfig{
		UseSQLite:      true,
		SQLiteInMemory: true,
	}
	testDB := testingutils.SetupTestDatabase(t, config)
	defer func() { _ = testDB.Cleanup() }()

	t.Run("Database Connection", func(t *testing.T) {
		// Test that database connection works
		err := testDB.DB.Ping()
		require.NoError(t, err, "Database should be accessible")
	})

	t.Run("Basic Test", func(t *testing.T) {
		// Simple placeholder test
		assert.True(t, true, "This should pass")
	})
}
