package tests

import (
	"database/sql"
	"os"
	"runtime"
	"testing"

	internalTesting "github.com/alexandru-savinov/BalancedNewsGo/internal/testing"
)

// TestDatabaseIntegration demonstrates database testing with testcontainers
func TestDatabaseIntegration(t *testing.T) {
	// Skip Docker-based tests on Windows or in CI environment
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Docker-based PostgreSQL test on Windows (Docker Desktop issues)")
	}

	// Also skip if NO_DOCKER environment variable is set
	if os.Getenv("NO_DOCKER") == "true" {
		t.Skip("Skipping Docker-based test (NO_DOCKER=true)")
	}

	// Setup test database with PostgreSQL
	config := internalTesting.DatabaseTestConfig{
		UsePostgres:    true,
		MigrationsPath: "../migrations",
		SeedDataPath:   "../testdata/seed",
	}
	testDB := internalTesting.SetupTestDatabase(t, config)
	defer func() {
		if err := testDB.Cleanup(); err != nil {
			t.Logf("Failed to cleanup test database: %v", err)
		}
	}()

	t.Run("TestArticleStorage", func(t *testing.T) {
		testArticleStorage(t, testDB)
	})

	t.Run("TestScoreStorage", func(t *testing.T) {
		testScoreStorage(t, testDB)
	})

	t.Run("TestFeedbackStorage", func(t *testing.T) {
		testFeedbackStorage(t, testDB)
	})
}

// TestSQLiteIntegration demonstrates SQLite testing
func TestSQLiteIntegration(t *testing.T) { // Setup test database with SQLite
	config := internalTesting.DatabaseTestConfig{
		UseSQLite:      true,
		SQLiteInMemory: true,
		MigrationsPath: "../migrations",
	}
	testDB := internalTesting.SetupTestDatabase(t, config)
	defer func() {
		if err := testDB.Cleanup(); err != nil {
			t.Logf("Failed to cleanup test database: %v", err)
		}
	}()

	t.Run("TestBasicOperations", func(t *testing.T) {
		testBasicDatabaseOperations(t, testDB)
	})
}

func testArticleStorage(t *testing.T, testDB *internalTesting.TestDatabase) {
	// Test article storage functionality
	testDB.Transaction(t, func(tx *sql.Tx) {
		// Insert test article
		_, err := tx.Exec(`
			INSERT INTO articles (id, title, content, url, source, pub_date, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		`, 1, "Test Article", "Test content", "http://test.com", "test-source")

		if err != nil {
			t.Fatalf("Failed to insert test article: %v", err)
		}

		// Query the article
		var title, content string
		err = tx.QueryRow("SELECT title, content FROM articles WHERE id = $1", 1).Scan(&title, &content)
		if err != nil {
			t.Fatalf("Failed to query test article: %v", err)
		}

		// Verify data
		if title != "Test Article" {
			t.Errorf("Expected title 'Test Article', got '%s'", title)
		}
		if content != "Test content" {
			t.Errorf("Expected content 'Test content', got '%s'", content)
		}
	})
}

func testScoreStorage(t *testing.T, testDB *internalTesting.TestDatabase) {
	// Test score storage functionality
	testDB.Transaction(t, func(tx *sql.Tx) {
		// First insert an article
		_, err := tx.Exec(`
			INSERT INTO articles (id, title, content, url, source, pub_date, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		`, 2, "Test Article 2", "Test content 2", "http://test2.com", "test-source")

		if err != nil {
			t.Fatalf("Failed to insert test article: %v", err)
		}

		// Insert score
		_, err = tx.Exec(`
			INSERT INTO llm_scores (article_id, model, score, metadata, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, 2, "test-model", 0.80, "{\"bias_score\": 0.75, \"credibility_score\": 0.85}")

		if err != nil {
			t.Fatalf("Failed to insert test score: %v", err)
		}

		// Query the score
		var model string
		var score float64
		var metadata string
		err = tx.QueryRow(`
			SELECT model, score, metadata
			FROM llm_scores WHERE article_id = $1
		`, 2).Scan(&model, &score, &metadata)

		if err != nil {
			t.Fatalf("Failed to query test score: %v", err)
		}

		// Verify scores
		if model != "test-model" {
			t.Errorf("Expected model 'test-model', got '%s'", model)
		}
		if score != 0.80 {
			t.Errorf("Expected score 0.80, got %f", score)
		}
		if metadata != "{\"bias_score\": 0.75, \"credibility_score\": 0.85}" {
			t.Errorf("Expected metadata with bias and credibility scores, got '%s'", metadata)
		}
	})
}

func testFeedbackStorage(t *testing.T, testDB *internalTesting.TestDatabase) {
	// Test feedback storage functionality
	testDB.Transaction(t, func(tx *sql.Tx) {
		// First insert an article to reference
		_, err := tx.Exec(`
			INSERT INTO articles (id, title, content, url, source, pub_date, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		`, 3, "Test Article for Feedback", "Test content for feedback", "http://test-feedback.com", "test-source")

		if err != nil {
			t.Fatalf("Failed to insert test article for feedback: %v", err)
		}

		// Insert feedback
		_, err = tx.Exec(`
			INSERT INTO feedback (article_id, user_id, feedback_text, category, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, 3, "test-user", "Good article", "positive")

		if err != nil {
			t.Fatalf("Failed to insert test feedback: %v", err)
		}

		// Query the feedback
		var category string
		var feedbackText string
		err = tx.QueryRow(`
			SELECT category, feedback_text
			FROM feedback WHERE article_id = $1
		`, 3).Scan(&category, &feedbackText)

		if err != nil {
			t.Fatalf("Failed to query test feedback: %v", err)
		}

		// Verify feedback
		if category != "positive" {
			t.Errorf("Expected category 'positive', got '%s'", category)
		}
		if feedbackText != "Good article" {
			t.Errorf("Expected feedback 'Good article', got '%s'", feedbackText)
		}
	})
}

func testBasicDatabaseOperations(t *testing.T, testDB *internalTesting.TestDatabase) {
	// Test basic database operations
	queries := []string{
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table'",
	}

	results := testDB.ExecuteTestQueries(t, queries)

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	t.Logf("Database contains %v tables", results[0])
}
