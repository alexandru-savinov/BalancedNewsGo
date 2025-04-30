package llm

import (
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRealCalculator is a mock implementation of ScoreCalculator for direct ScoreManager testing
type MockRealCalculator struct {
	mock.Mock
}

func (m *MockRealCalculator) CalculateScore(scores []db.LLMScore) (float64, float64, error) {
	args := m.Called(scores)
	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

// TestIntegrationUpdateArticleScore tests the UpdateArticleScore method directly
func TestIntegrationUpdateArticleScore(t *testing.T) {
	// Create a mock DB that satisfies the sqlx.DB interface
	mockDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	// Wrap with sqlx
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	// Create the rest of our dependencies
	cache := NewCache()
	calculator := new(MockRealCalculator)
	progressMgr := NewProgressManager(time.Minute)

	// Create a real ScoreManager with our dependencies
	sm := NewScoreManager(sqlxDB, cache, calculator, progressMgr)

	// Test data
	articleID := int64(123)
	testScores := []db.LLMScore{
		{ArticleID: articleID, Model: "left", Score: -0.8, Metadata: `{"confidence":0.9}`},
		{ArticleID: articleID, Model: "center", Score: 0.1, Metadata: `{"confidence":0.7}`},
		{ArticleID: articleID, Model: "right", Score: 0.5, Metadata: `{"confidence":0.8}`},
	}
	config := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "left", Perspective: "left"},
			{ModelName: "center", Perspective: "center"},
			{ModelName: "right", Perspective: "right"},
		},
	}
	expectedScore := 0.1
	expectedConfidence := 0.8

	// Mock the calculator's behavior
	calculator.On("CalculateScore", testScores).Return(expectedScore, expectedConfidence, nil)

	// Mock database transaction
	sqlMock.ExpectBegin()

	// Mock the InsertLLMScore call using NamedExec like in the real implementation
	sqlMock.ExpectExec(`INSERT INTO llm_scores \(article_id, model, score, metadata, version, created_at\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock the UpdateArticleScoreLLM call
	sqlMock.ExpectExec(`UPDATE articles SET composite_score = \?, confidence = \?, score_source = 'llm' WHERE id = \?`).
		WithArgs(expectedScore, expectedConfidence, articleID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Mock the transaction commit
	sqlMock.ExpectCommit()

	// Call the method
	score, confidence, err := sm.UpdateArticleScore(articleID, testScores, config)

	// Verify the results
	assert.NoError(t, err, "Expected no error from UpdateArticleScore")
	assert.Equal(t, expectedScore, score, "Expected score to match")
	assert.Equal(t, expectedConfidence, confidence, "Expected confidence to match")

	// Verify that all database expectations were met
	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Errorf("Database expectations were not met: %v", err)
	}

	// Verify progress manager has the final state
	progressState := sm.GetProgress(articleID)
	assert.NotNil(t, progressState, "Expected progress state to be set")
	assert.Equal(t, "Success", progressState.Status, "Expected status to be Success")
	assert.Equal(t, "Complete", progressState.Step, "Expected step to be Complete")
	assert.Equal(t, 100, progressState.Percent, "Expected percent to be 100")
}

// TestIntegrationUpdateArticleScore_CalculationError tests error handling for calculation failures
func TestIntegrationUpdateArticleScore_CalculationError(t *testing.T) {
	// Create a mock DB that satisfies the sqlx.DB interface
	mockDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer mockDB.Close()

	// Wrap with sqlx
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	// Create the rest of our dependencies
	cache := NewCache()
	calculator := new(MockRealCalculator)
	progressMgr := NewProgressManager(time.Minute)

	// Create a real ScoreManager with our dependencies
	sm := NewScoreManager(sqlxDB, cache, calculator, progressMgr)

	// Test data
	articleID := int64(123)
	testScores := []db.LLMScore{
		{ArticleID: articleID, Model: "model1", Score: -0.5},
	}
	config := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "model1", Perspective: "left"},
		},
	}
	expectedError := fmt.Errorf("calculation failed")

	// Mock the calculator to return an error
	calculator.On("CalculateScore", testScores).Return(0.0, 0.0, expectedError)

	// Mock database transaction - only begin and rollback should be called
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	// Call the method
	score, confidence, err := sm.UpdateArticleScore(articleID, testScores, config)

	// Verify the results
	assert.Error(t, err, "Expected an error from UpdateArticleScore")
	assert.Contains(t, err.Error(), "calculation failed", "Error should mention calculation failure")
	assert.Equal(t, 0.0, score, "Score should be 0.0 on error")
	assert.Equal(t, 0.0, confidence, "Confidence should be 0.0 on error")

	// Verify that all database expectations were met
	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Errorf("Database expectations were not met: %v", err)
	}

	// Verify progress manager has error state
	progressState := sm.GetProgress(articleID)
	assert.NotNil(t, progressState, "Expected progress state to be set")
	assert.Equal(t, "Error", progressState.Status, "Expected status to be Error")
	assert.Contains(t, progressState.Error, "calculation failed", "Error message should be set")
}

// TestIntegrationTrackProgress tests the TrackProgress function
func TestIntegrationTrackProgress(t *testing.T) {
	// Create a new ScoreManager with just the progress manager
	progressMgr := NewProgressManager(time.Minute)
	sm := &ScoreManager{progressMgr: progressMgr}

	// Test data
	articleID := int64(123)
	step := "Initialize"
	status := "Pending"

	// Call TrackProgress
	sm.TrackProgress(articleID, step, status)

	// Verify that the progress state was set correctly
	progress := progressMgr.GetProgress(articleID)
	assert.NotNil(t, progress, "Progress state should have been set")
	assert.Equal(t, step, progress.Step, "Step should be 'Initialize'")
	assert.Equal(t, status, progress.Status, "Status should be 'Pending'")
	assert.Equal(t, 0, progress.Percent, "Percent should be 0")
	assert.Contains(t, progress.Message, step, "Message should contain the step")
}
