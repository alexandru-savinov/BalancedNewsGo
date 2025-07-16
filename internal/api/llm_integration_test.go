package api

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	_ "modernc.org/sqlite"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// LLMTestDB represents a test database for LLM integration tests
type LLMTestDB struct {
	*sqlx.DB
	cleanup func()
}

// setupLLMTestDB creates a test database with proper schema for LLM tests
func setupLLMTestDB(t *testing.T) *LLMTestDB {
	dbConn, err := sqlx.Open("sqlite", ":memory:")
	assert.NoError(t, err, "Failed to create test database")

	// Apply comprehensive schema
	schema := `
	CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		pub_date TIMESTAMP NOT NULL,
		url TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'pending',
		fail_count INTEGER DEFAULT 0,
		last_attempt TIMESTAMP,
		escalated BOOLEAN DEFAULT FALSE,
		composite_score REAL,
		confidence REAL,
		score_source TEXT
	);

	CREATE TABLE IF NOT EXISTS llm_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id INTEGER NOT NULL,
		model TEXT NOT NULL,
		score REAL NOT NULL,
		metadata TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (article_id) REFERENCES articles (id)
	);`

	_, err = dbConn.Exec(schema)
	assert.NoError(t, err, "Failed to apply test schema")

	cleanup := func() {
		if err := dbConn.Close(); err != nil {
			t.Logf("Warning: Failed to close test database: %v", err)
		}
	}

	t.Cleanup(cleanup)

	return &LLMTestDB{
		DB:      dbConn,
		cleanup: cleanup,
	}
}

// MockLLMClientForIntegration for LLM integration testing
type MockLLMClientForIntegration struct {
	mock.Mock
}

func (m *MockLLMClientForIntegration) ValidateAPIKey() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockLLMClientForIntegration) ReanalyzeArticle(ctx context.Context, articleID int64, scoreManager *llm.ScoreManager) error {
	args := m.Called(ctx, articleID, scoreManager)
	return args.Error(0)
}

// MockScoreManagerForIntegration for LLM integration testing
type MockScoreManagerForIntegration struct {
	mock.Mock
	progressStates map[int64]*models.ProgressState
}

func NewMockScoreManagerForIntegration() *MockScoreManagerForIntegration {
	return &MockScoreManagerForIntegration{
		progressStates: make(map[int64]*models.ProgressState),
	}
}

func (m *MockScoreManagerForIntegration) GetProgress(articleID int64) *models.ProgressState {
	args := m.Called(articleID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*models.ProgressState)
}

func (m *MockScoreManagerForIntegration) SetProgress(articleID int64, state *models.ProgressState) {
	m.Called(articleID, state)
	m.progressStates[articleID] = state
}

// TestLLMHandlerValidation tests basic validation for LLM handlers
func TestLLMHandlerValidation(t *testing.T) {
	// Test getValidArticleID helper function which is used by LLM handlers
	t.Run("getValidArticleID_validation", func(t *testing.T) {
		router := gin.New()
		router.GET("/test/:id", func(c *gin.Context) {
			id, ok := getValidArticleID(c)
			if !ok {
				return // Error already sent
			}
			c.JSON(200, gin.H{"id": id})
		})

		// Test valid IDs
		validIDs := []string{"1", "123", "999999"}
		for _, id := range validIDs {
			req := httptest.NewRequest("GET", "/test/"+id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code, "Valid ID %s should return 200", id)
		}

		// Test invalid IDs
		invalidIDs := []string{"invalid", "0", "-1", "abc", "1.5"}
		for _, id := range invalidIDs {
			req := httptest.NewRequest("GET", "/test/"+id, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, 400, w.Code, "Invalid ID %s should return 400", id)
		}
	})

	// Test basic database operations for LLM handlers
	t.Run("llm_handler_database_operations", func(t *testing.T) {
		testDB := setupLLMTestDB(t)

		// Insert test article
		result, err := testDB.DB.Exec(`
			INSERT INTO articles (source, pub_date, url, title, content, status)
			VALUES (?, ?, ?, ?, ?, ?)
		`, "Test Source", time.Now(), "https://example.com/llm-test", "LLM Test Article", "Test content", "analyzed")
		assert.NoError(t, err)
		articleID, _ := result.LastInsertId()

		// Test that article exists and can be fetched
		var count int
		err = testDB.DB.Get(&count, "SELECT COUNT(*) FROM articles WHERE id = ?", articleID)
		assert.NoError(t, err)
		assert.Equal(t, 1, count, "Article should exist in database")

		// Test LLM scores table operations
		_, err = testDB.DB.Exec(`
			INSERT INTO llm_scores (article_id, model, score, metadata)
			VALUES (?, ?, ?, ?)
		`, articleID, "test-model", 0.5, `{"test": "metadata"}`)
		assert.NoError(t, err)

		// Verify LLM score was inserted
		var scoreCount int
		err = testDB.DB.Get(&scoreCount, "SELECT COUNT(*) FROM llm_scores WHERE article_id = ?", articleID)
		assert.NoError(t, err)
		assert.Equal(t, 1, scoreCount, "LLM score should exist in database")
	})
}

// Note: SSE handler tests are complex due to streaming nature and are covered in E2E tests
