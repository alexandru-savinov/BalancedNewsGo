package llm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDB implements database operations for testing
type MockDB struct {
	mock.Mock
}

func (m *MockDB) BeginTxx(ctx context.Context, opts interface{}) (*sqlx.Tx, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqlx.Tx), args.Error(1)
}

// MockScoreCalculator implements ScoreCalculator for testing
type MockScoreCalculator struct {
	mock.Mock
}

func (m *MockScoreCalculator) CalculateScore(scores []db.LLMScore) (float64, float64, error) {
	args := m.Called(scores)
	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

// MockProgressManager implements progress tracking for testing
type MockProgressManager struct {
	mock.Mock
}

func (m *MockProgressManager) SetProgress(articleID int64, state *models.ProgressState) {
	m.Called(articleID, state)
}

func (m *MockProgressManager) GetProgress(articleID int64) *models.ProgressState {
	args := m.Called(articleID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*models.ProgressState)
}

// MockTX implements sqlx.Tx for testing
type MockTX struct {
	mock.Mock
}

func (m *MockTX) Exec(query string, args ...interface{}) (sql.Result, error) {
	callArgs := m.Called(append([]interface{}{query}, args...)...)
	return callArgs.Get(0).(sql.Result), callArgs.Error(1)
}

func (m *MockTX) Commit() error {
	return m.Called().Error(0)
}

func (m *MockTX) Rollback() error {
	return m.Called().Error(0)
}

// MockSqlResult implements sql.Result for testing
type MockSqlResult struct {
	mock.Mock
}

func (m *MockSqlResult) LastInsertId() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSqlResult) RowsAffected() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// mockScoreCalculator is a mock implementation of the ScoreCalculator interface.
type mockScoreCalculator struct {
	calculateScoreFunc func(scores []db.LLMScore) (float64, float64, error)
}

func (m *mockScoreCalculator) CalculateScore(scores []db.LLMScore) (float64, float64, error) {
	if m.calculateScoreFunc != nil {
		return m.calculateScoreFunc(scores)
	}
	return 0, 0, fmt.Errorf("mockScoreCalculator.calculateScoreFunc not set")
}

// Helper function to set up an in-memory DB for LLM tests, if not already present
// This is similar to the one in db_test.go but might need adjustments for llm tests.
// For now, this is a placeholder if a specific one for llm_test isn't there.
func setupLLMTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	// Use a unique name for the in-memory database to avoid conflicts if tests run in parallel
	// and don't properly clean up or if they share the same :memory: instance.
	// However, for SQLite :memory:, each connection is distinct unless "cache=shared" is used.
	// Standard :memory: is fine.
	db, err := sqlx.Open("sqlite", ":memory:?_foreign_keys=on&_busy_timeout=5000")
	require.NoError(t, err, "Failed to open in-memory database for llm test")
	require.NotNil(t, db, "DB connection should not be nil for llm test")

	err = db.Ping()
	require.NoError(t, err, "Failed to ping in-memory database for llm test")

	// Create articles table with status, similar to db_test.go's TestUpdateArticleStatusFailed
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT,
			pub_date DATETIME,
			url TEXT UNIQUE,
			title TEXT,
			content TEXT,
			created_at DATETIME,
			composite_score REAL,
			confidence REAL,
			score_source TEXT,
			status TEXT
		);
	`)
	require.NoError(t, err, "Failed to create articles table for llm test")

	// Create llm_scores table as ScoreManager.UpdateArticleScore inserts into it
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS llm_scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			article_id INTEGER NOT NULL,
			model TEXT NOT NULL,
			score REAL NOT NULL,
			metadata TEXT,
			version TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (article_id) REFERENCES articles (id)
		);
	`)
	require.NoError(t, err, "Failed to create llm_scores table for llm test")

	return db
}

func TestNewScoreManager(t *testing.T) {
	mockDB := &sqlx.DB{}
	mockCache := NewCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := NewProgressManager(time.Minute)

	sm := NewScoreManager(mockDB, mockCache, mockCalculator, mockProgress)

	assert.NotNil(t, sm)
	assert.Equal(t, mockDB, sm.db)
	assert.Equal(t, mockCache, sm.cache)
	assert.Equal(t, mockCalculator, sm.calculator)
	assert.Equal(t, mockProgress, sm.progressMgr)
}

func TestInvalidateScoreCache(t *testing.T) {
	// Create mocks
	mockDB := &sqlx.DB{}
	cache := NewCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := NewProgressManager(time.Minute)

	sm := NewScoreManager(mockDB, cache, mockCalculator, mockProgress)

	// Test data
	articleID := int64(123)

	// Call method
	sm.InvalidateScoreCache(articleID)

	// Since we can't easily mock the Cache.Delete method (it's not an interface)
	// we'll just verify the function runs without error
	assert.NotNil(t, sm)
}

func TestSetGetProgress(t *testing.T) {
	// Create mocks
	mockDB := &sqlx.DB{}
	cache := NewCache()
	mockCalculator := new(MockScoreCalculator)

	// Create a real progress manager for this test - easier than mocking internal state
	mockProgress := NewProgressManager(time.Minute)

	sm := NewScoreManager(mockDB, cache, mockCalculator, mockProgress)

	// Test data
	articleID := int64(123)
	progressState := &models.ProgressState{
		Step:        "Testing",
		Message:     "Unit testing progress",
		Percent:     50,
		Status:      "InProgress",
		LastUpdated: time.Now().Unix(),
	}

	// Call methods
	sm.SetProgress(articleID, progressState)
	result := sm.GetProgress(articleID)

	// Verify
	assert.Equal(t, progressState, result)
}

func TestUpdateArticleScore_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	testDB := setupLLMTestDB(t)
	defer testDB.Close()

	// Insert a dummy article
	currentTime := time.Now()
	articleRes, err := testDB.ExecContext(ctx, `
		INSERT INTO articles (source, pub_date, url, title, content, created_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test_source", currentTime, "http://example.com/llm-error-test-unique", "LLM Error Test", "Content", currentTime, "new")
	require.NoError(t, err)
	articleID, err := articleRes.LastInsertId()
	require.NoError(t, err)

	// Define test cases
	testCases := []struct {
		name                  string
		mockCalculatorError   error
		expectedArticleStatus string
	}{
		{
			name:                  "ErrAllPerspectivesInvalid",
			mockCalculatorError:   ErrAllPerspectivesInvalid,
			expectedArticleStatus: ArticleStatusFailedAllInvalid,
		},
		{
			name:                  "ErrAllScoresZeroConfidence",
			mockCalculatorError:   ErrAllScoresZeroConfidence,
			expectedArticleStatus: ArticleStatusFailedZeroConfidence,
		},
		{
			name:                  "OtherError",
			mockCalculatorError:   fmt.Errorf("some other calculation error"),
			expectedArticleStatus: "new", // Status should remain unchanged by the new logic for other errors
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset article state for "OtherError" case to ensure it wasn't modified by previous specific error runs
			// This ensures the article starts from a known "new" state for this sub-test.
			_, err := testDB.ExecContext(ctx, `
				UPDATE articles SET composite_score = NULL, confidence = NULL, status = ?, score_source = NULL
				WHERE id = ?
			`, "new", articleID)
			require.NoError(t, err, "Failed to reset article state for test: %s", tc.name)

			mockCalc := &mockScoreCalculator{
				calculateScoreFunc: func(scores []db.LLMScore) (float64, float64, error) {
					return 0, 0, tc.mockCalculatorError
				},
			}

			scoreManager := NewScoreManager(testDB, nil, mockCalc, nil)

			dummyScores := []db.LLMScore{{ArticleID: articleID, Model: "test", Score: 0.1}}
			// Assuming CompositeScoreConfig is a struct and an empty one is valid for this test's purpose.
			// If it has required fields, this might need adjustment.
			dummyConfig := &CompositeScoreConfig{}

			_, _, updateErr := scoreManager.UpdateArticleScore(articleID, dummyScores, dummyConfig)

			require.Error(t, updateErr, "UpdateArticleScore should return an error for %s", tc.name)
			assert.True(t, errors.Is(updateErr, tc.mockCalculatorError), "UpdateArticleScore error for %s should wrap: %v, got: %v", tc.name, tc.mockCalculatorError, updateErr)

			type TestArticleData struct { // Renamed to avoid conflict with other TestArticle structs if any
				ID             int64    `db:"id"`
				CompositeScore *float64 `db:"composite_score"`
				Confidence     *float64 `db:"confidence"`
				ScoreSource    *string  `db:"score_source"`
				Status         *string  `db:"status"`
			}
			var articleData TestArticleData
			err = testDB.GetContext(ctx, &articleData, "SELECT id, composite_score, confidence, score_source, status FROM articles WHERE id = ?", articleID)
			require.NoError(t, err, "Failed to fetch article data from DB for %s", tc.name)

			assert.Equal(t, articleID, articleData.ID)

			if tc.mockCalculatorError == ErrAllPerspectivesInvalid || tc.mockCalculatorError == ErrAllScoresZeroConfidence {
				assert.Nil(t, articleData.CompositeScore, "CompositeScore should be NULL for %s", tc.name)
				assert.Nil(t, articleData.Confidence, "Confidence should be NULL for %s", tc.name)
				require.NotNil(t, articleData.Status, "Status should not be nil for %s", tc.name)
				assert.Equal(t, tc.expectedArticleStatus, *articleData.Status, "Status not updated correctly for %s", tc.name)
				require.NotNil(t, articleData.ScoreSource, "ScoreSource should not be nil for %s", tc.name)
				assert.Equal(t, "error", *articleData.ScoreSource, "ScoreSource should be 'error' for %s", tc.name)
			} else { // For "OtherError"
				require.NotNil(t, articleData.Status, "Status should not be nil for %s", tc.name)
				assert.Equal(t, tc.expectedArticleStatus, *articleData.Status, "Status should be %s for other errors, got %s", tc.expectedArticleStatus, *articleData.Status)
				if articleData.ScoreSource != nil {
					assert.NotEqual(t, "error", *articleData.ScoreSource, "ScoreSource should not be 'error' for other errors")
				}
				// For "OtherError", scores might be null if they were never set or null from the reset.
				// The key is that UpdateArticleStatusFailed didn't run and overwrite ScoreSource to "error".
			}
		})
	}
}
