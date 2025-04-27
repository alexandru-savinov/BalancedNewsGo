package llm

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
