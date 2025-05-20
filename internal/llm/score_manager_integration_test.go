package llm

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// Use the correct package name for llm types
)

const errFailedToCreateMockDB = "Failed to create mock DB: %v"

// MockRealCalculator implements ScoreCalculator for integration tests
type MockRealCalculator struct {
	CalculatedScore      float64
	CalculatedConfidence float64
	ShouldError          error // Configurable error
}

// CalculateScore mocks the real calculation but doesn't need complex logic for integration test setup
func (m *MockRealCalculator) CalculateScore(scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
	if m.ShouldError != nil {
		return 0, 0, m.ShouldError // Return the configured error
	}
	// Simple placeholder logic for testing integration flow
	if len(scores) == 0 {
		// If specific behavior for empty scores is needed for a test:
		if m.CalculatedScore == 0 && m.CalculatedConfidence == 0 { // Default unconfigured behavior
			return 0, 0, nil
		}
	}
	// Return configured values or simple defaults
	// For more complex scenarios, tests can set CalculatedScore and CalculatedConfidence on the mock instance.
	// Defaulting to first score if not specifically set by test and no error.
	if m.CalculatedScore == 0 && m.CalculatedConfidence == 0 && len(scores) > 0 && m.ShouldError == nil {
		return scores[0].Score, 0.8, nil // Example: Return first score and fixed confidence
	}
	return m.CalculatedScore, m.CalculatedConfidence, nil
}

// TestIntegrationUpdateArticleScore tests the UpdateArticleScore method directly
func TestIntegrationUpdateArticleScore(t *testing.T) {
	// Create a mock DB that satisfies the sqlx.DB interface
	mockDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errFailedToCreateMockDB, err)
	}
	defer func() { _ = mockDB.Close() }()

	// Wrap with sqlx
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	// Create the rest of our dependencies
	cache := NewCache()
	calculator := &MockRealCalculator{
		CalculatedScore:      0.1,
		CalculatedConfidence: 0.8,
		ShouldError:          nil,
	}
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
	// calculator.On("CalculateScore", testScores, config).Return(expectedScore, expectedConfidence, nil)

	// Mock the UpdateArticleScoreLLM call
	sqlMock.ExpectExec("UPDATE articles SET composite_score = \\?, confidence = \\?, score_source = 'llm' WHERE id = \\?").
		WithArgs(expectedScore, expectedConfidence, articleID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock the subsequent UpdateArticleStatus call to models.ArticleStatusScored
	sqlMock.ExpectExec("UPDATE articles SET status = \\? WHERE id = \\?").
		WithArgs(models.ArticleStatusScored, articleID).
		WillReturnResult(sqlmock.NewResult(1, 1))

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

// TestIntegrationUpdateArticleScoreCalculationError tests error handling for calculation failures
func TestIntegrationUpdateArticleScore_ErrAllPerspectivesInvalid(t *testing.T) {
	// Create a mock DB that satisfies the sqlx.DB interface
	mockDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errFailedToCreateMockDB, err)
	}
	defer func() { _ = mockDB.Close() }()

	// Wrap with sqlx
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	// Create the rest of our dependencies
	cache := NewCache()
	// Configure MockRealCalculator to return ErrAllPerspectivesInvalid
	calculator := &MockRealCalculator{
		ShouldError: ErrAllPerspectivesInvalid,
	}
	progressMgr := NewProgressManager(time.Minute)

	// Create a real ScoreManager with our dependencies
	sm := NewScoreManager(sqlxDB, cache, calculator, progressMgr)

	// Test data
	articleID := int64(456) // Different ID for clarity
	// Scores that would lead to all invalid if processed by the real calculator with "ignore"
	testScores := []db.LLMScore{
		{ArticleID: articleID, Model: "left", Score: -999, Metadata: `{"confidence": 0.9}`},  // Invalid due to MinScore/MaxScore
		{ArticleID: articleID, Model: "center", Score: 999, Metadata: `{"confidence": 0.9}`}, // Invalid due to MinScore/MaxScore
		{ArticleID: articleID, Model: "right", Score: 1000, Metadata: `{"confidence": 0.9}`}, // Invalid due to MinScore/MaxScore
	}
	config := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "left", Perspective: "left"},
			{ModelName: "center", Perspective: "center"},
			{ModelName: "right", Perspective: "right"},
		},
		MinScore:      -1.0, // Example bounds
		MaxScore:      1.0,
		HandleInvalid: "ignore", // Important for the error to be ErrAllPerspectivesInvalid
	}

	// Expect UpdateArticleStatus to be called for models.ArticleStatusFailedAllInvalid
	sqlMock.ExpectExec("UPDATE articles SET status = \\? WHERE id = \\?").
		WithArgs(models.ArticleStatusFailedAllInvalid, articleID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Call the method
	score, confidence, err := sm.UpdateArticleScore(articleID, testScores, config)

	// Verify the results
	require.Error(t, err, "Expected an error from UpdateArticleScore")
	require.ErrorIs(t, err, ErrAllPerspectivesInvalid, "Error should be ErrAllPerspectivesInvalid")
	assert.Equal(t, 0.0, score, "Score should be 0.0 on ErrAllPerspectivesInvalid")
	assert.Equal(t, 0.0, confidence, "Confidence should be 0.0 on ErrAllPerspectivesInvalid")

	// Verify that all database expectations were met (only status update, no score update)
	if errDb := sqlMock.ExpectationsWereMet(); errDb != nil {
		t.Errorf("Database expectations were not met: %v", errDb)
	}

	// Verify progress manager has error state
	progressState := sm.GetProgress(articleID)
	assert.NotNil(t, progressState, "Expected progress state to be set")
	assert.Equal(t, "Error", progressState.Status, "Expected status to be Error")
	require.NotNil(t, progressState.Error, "progressState.Error should be set")
	assert.Equal(t, ErrAllPerspectivesInvalid.Error(), progressState.Error, "Error message should be ErrAllPerspectivesInvalid")
	assert.Equal(t, ErrAllPerspectivesInvalid.Error(), progressState.Message, "Message should also be ErrAllPerspectivesInvalid")
	assert.Equal(t, 100, progressState.Percent, "Expected percent to be 100")
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

func setupIntegrationTest(t *testing.T) (*sqlx.DB, *ScoreManager, *ProgressManager, sqlmock.Sqlmock) {
	// Create a mock DB that satisfies the sqlx.DB interface
	mockDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errFailedToCreateMockDB, err)
	}
	// No need to defer close here, handled by caller

	// Wrap with sqlx
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	// Create the rest of our dependencies
	cache := NewCache()
	calculator := &MockRealCalculator{}
	progressMgr := NewProgressManager(time.Minute)

	// Create a real ScoreManager with our dependencies
	sm := NewScoreManager(sqlxDB, cache, calculator, progressMgr)

	return sqlxDB, sm, progressMgr, sqlMock // Return sqlxDB and sqlMock
}

func TestScoreManagerIntegrationUpdateArticleScoreZeroConfidenceError(t *testing.T) {
	mockDB, scoreManager, _, _ := setupIntegrationTest(t) // Assign sqlMock to _
	defer func() { _ = mockDB.Close() }()                 // Close the sqlxDB wrapper

	// Prepare mock calculator to return zero confidence
	// (This mock setup might need adjustment based on how MockRealCalculator is implemented)
	// Since MockRealCalculator now has simple logic, we test the ScoreManager's handling
	// by providing scores that would lead to zero confidence via checkForAllZeroResponses

	articleID := int64(1)
	// Setup scores that trigger the zero confidence check in ScoreManager
	zeroScores := []db.LLMScore{
		{ArticleID: articleID, Model: "model1", Score: 0.0, Metadata: `{"confidence": 0.0}`},
		{ArticleID: articleID, Model: "model2", Score: 0.0, Metadata: `{"confidence": 0.0}`},
	}

	// Expect the calculator's CalculateScore NOT to be called directly
	// The error should be caught earlier by ScoreManager

	// Expect UpdateArticleScoreLLM NOT to be called

	cfg := &CompositeScoreConfig{ /* ... fill if needed ... */ }
	_, _, err := scoreManager.UpdateArticleScore(articleID, zeroScores, cfg) // Pass cfg

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all LLMs returned zero confidence")
}

func TestScoreManagerIntegrationCalculateScoreError(t *testing.T) {
	mockDB, scoreManager, progressMgr, sqlMock := setupIntegrationTest(t)
	defer func() { _ = mockDB.Close() }()

	articleID := int64(1)
	config := &CompositeScoreConfig{}
	testScores := []db.LLMScore{
		{ArticleID: articleID, Model: "model1", Score: 0.1, Metadata: `{"confidence": 0.8}`},
	}

	// Configure the mock calculator to return a generic error
	genericError := errors.New("some generic calculation error")
	if mockCalc, ok := scoreManager.calculator.(*MockRealCalculator); ok {
		mockCalc.ShouldError = genericError
	} else {
		t.Fatal("Could not cast calculator to *MockRealCalculator to set error")
	}

	// Expect UpdateArticleStatus to be called for models.ArticleStatusFailedError
	sqlMock.ExpectExec("UPDATE articles SET status = \\? WHERE id = \\?").
		WithArgs(models.ArticleStatusFailedError, articleID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Update the article score
	calcScore, calcConf, err := scoreManager.UpdateArticleScore(articleID, testScores, config)

	require.Error(t, err)
	assert.True(t, errors.Is(err, genericError), "Expected the generic error to be returned")
	assert.Equal(t, 0.0, calcScore)
	assert.Equal(t, 0.0, calcConf)

	// Verify progress
	progressState := progressMgr.GetProgress(articleID)
	assert.NotNil(t, progressState)
	assert.Equal(t, "Error", progressState.Status)
	assert.Contains(t, progressState.Message, "Internal error calculating score:")
	assert.Contains(t, progressState.Message, genericError.Error())

	err = sqlMock.ExpectationsWereMet()
	assert.NoError(t, err, "DB expectations not met")
}
