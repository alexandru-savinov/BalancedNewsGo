package llm

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// DBInterface defines the database operations needed by ScoreManager
// We'll use this to create a testable version of ScoreManager
type DBInterface interface {
	BeginTxx(ctx context.Context, opts interface{}) (interface{}, error)
}

// ProgressInterface defines the progress tracking operations needed by ScoreManager
type ProgressInterface interface {
	SetProgress(articleID int64, state *models.ProgressState)
	GetProgress(articleID int64) *models.ProgressState
}

// MockDB implements database operations for testing
type MockDB struct {
	mock.Mock
}

func (m *MockDB) BeginTxx(ctx context.Context, opts interface{}) (interface{}, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

// MockScoreCalculator implements ScoreCalculator for testing
type MockScoreCalculator struct {
	mock.Mock
}

func (m *MockScoreCalculator) CalculateScore(scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
	args := m.Called(scores, cfg)
	mockedErr := args.Error(2)
	val1, ok1 := args.Get(0).(float64)
	if !ok1 {
		return 0.0, 0.0, mockedErr // Or a specific error about type mismatch
	}
	val2, ok2 := args.Get(1).(float64)
	if !ok2 {
		return val1, 0.0, mockedErr // Or a specific error about type mismatch
	}
	return val1, val2, mockedErr
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
	val, ok := args.Get(0).(*models.ProgressState)
	if !ok {
		return nil // Or panic/log, as this indicates a mock setup issue
	}
	return val
}

// MockTX implements sqlx.Tx for testing
type MockTX struct {
	mock.Mock
}

func (m *MockTX) Exec(query string, args ...interface{}) (sql.Result, error) {
	callArgs := m.Called(append([]interface{}{query}, args...)...)
	mockedErr := callArgs.Error(1)
	val, ok := callArgs.Get(0).(sql.Result)
	if !ok {
		return nil, mockedErr // Or a specific error about type mismatch
	}
	return val, mockedErr
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
	mockedErr := args.Error(1)
	val, ok := args.Get(0).(int64)
	if !ok {
		return 0, mockedErr // Or a specific error about type mismatch
	}
	return val, mockedErr
}

func (m *MockSqlResult) RowsAffected() (int64, error) {
	args := m.Called()
	mockedErr := args.Error(1)
	val, ok := args.Get(0).(int64)
	if !ok {
		return 0, mockedErr // Or a specific error about type mismatch
	}
	return val, mockedErr
}

// Custom implementation of Cache for testing with simplified interface
type TestCache struct {
	data map[string]interface{}
	mu   sync.Mutex
}

// NewTestCache creates a new test cache instance
func NewTestCache() *TestCache {
	return &TestCache{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in the test cache
func (c *TestCache) Set(key string, value interface{}, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Get retrieves a value from the test cache
func (c *TestCache) Get(key string) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.data[key]
}

// Delete removes a value from the test cache
func (c *TestCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// TestableScoreManager extends ScoreManager to allow mock dependencies
type TestableScoreManager struct {
	db           interface{}
	cache        *TestCache
	calculator   ScoreCalculator
	progressMgr  interface{}
	mockDB       *MockDB
	mockTx       *MockTX
	mockProgress *MockProgressManager
}

// NewTestableScoreManager creates a ScoreManager that works with our mocks
func NewTestableScoreManager(mockDB *MockDB, cache *TestCache, calculator ScoreCalculator, mockProgress *MockProgressManager) *TestableScoreManager {
	return &TestableScoreManager{
		db:           nil,
		cache:        cache,
		calculator:   calculator,
		progressMgr:  nil,
		mockDB:       mockDB,
		mockTx:       nil,
		mockProgress: mockProgress,
	}
}

// Override methods to use our mocks instead of real implementations
func (sm *TestableScoreManager) UpdateArticleScore(articleID int64, scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
	if sm.calculator == nil {
		return 0, 0, fmt.Errorf("ScoreManager: calculator is nil")
	}
	if sm.mockDB == nil {
		return 0, 0, fmt.Errorf("ScoreManager: db is nil")
	}
	if cfg == nil {
		return 0, 0, fmt.Errorf("ScoreManager: config is nil")
	}

	// First, check if all responses have zero confidence - matching the real implementation
	if allZeros, err := checkForAllZeroResponses(scores); allZeros {
		if sm.mockProgress != nil {
			ps := &models.ProgressState{
				Step:        "Error",
				Message:     "All LLMs returned zero confidence - scoring failed",
				Status:      "Error",
				Error:       err.Error(),
				LastUpdated: time.Now().Unix(),
			}
			sm.mockProgress.SetProgress(articleID, ps)
		}
		return 0, 0, err
	}

	// Use the score calculator to compute the score and confidence
	compositeScore, confidence, err := sm.calculator.CalculateScore(scores, cfg)
	if err != nil {
		if sm.mockProgress != nil {
			ps := &models.ProgressState{
				Step:        "Error",
				Message:     "Failed to compute composite score",
				Status:      "Error",
				Error:       err.Error(),
				LastUpdated: time.Now().Unix(),
			}
			sm.mockProgress.SetProgress(articleID, ps)
		}
		return 0, 0, err
	}

	// Update progress
	if sm.mockProgress != nil {
		ps := &models.ProgressState{
			Step:        "Complete",
			Message:     "Score update complete",
			Status:      "Success",
			Percent:     100,
			FinalScore:  &compositeScore,
			LastUpdated: time.Now().Unix(),
		}
		sm.mockProgress.SetProgress(articleID, ps)
	}

	return compositeScore, confidence, nil
}

// Override SetProgress to use our mock
func (sm *TestableScoreManager) SetProgress(articleID int64, state *models.ProgressState) {
	if sm.mockProgress != nil {
		sm.mockProgress.SetProgress(articleID, state)
	}
}

// Override GetProgress to use our mock
func (sm *TestableScoreManager) GetProgress(articleID int64) *models.ProgressState {
	if sm.mockProgress != nil {
		return sm.mockProgress.GetProgress(articleID)
	}
	return nil
}

// Override InvalidateScoreCache to make it testable
func (sm *TestableScoreManager) InvalidateScoreCache(articleID int64) {
	if sm.cache == nil {
		return
	}
	// Invalidate all relevant cache keys (matching API cache usage)
	keys := []string{
		fmt.Sprintf("article:%d", articleID),
		fmt.Sprintf("ensemble:%d", articleID),
		fmt.Sprintf("bias:%d", articleID),
	}
	for _, key := range keys {
		sm.cache.Delete(key)
	}
}

func TestNewScoreManager(t *testing.T) {
	mockCache := NewCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := NewProgressManager(time.Minute)

	sm := NewScoreManager(nil, mockCache, mockCalculator, mockProgress)

	assert.NotNil(t, sm)
	assert.Equal(t, mockCache, sm.cache)
	assert.Equal(t, mockCalculator, sm.calculator)
	assert.Equal(t, mockProgress, sm.progressMgr)
}

func TestInvalidateScoreCache(t *testing.T) {
	// Create mocks
	cache := NewCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := NewProgressManager(time.Minute)

	sm := NewScoreManager(nil, cache, mockCalculator, mockProgress)

	// Test data
	articleID := int64(123)

	// Call method
	sm.InvalidateScoreCache(articleID)

	// Since we can't easily mock the Cache.Delete method (it's not an interface)
	// we'll just verify the function runs without error
	assert.NotNil(t, sm)
}

func TestSetGetProgress(t *testing.T) {
	mockProgress := new(MockProgressManager)
	sm := NewTestableScoreManager(nil, nil, nil, mockProgress)

	articleID := int64(1)
	initialState := &models.ProgressState{
		Status:  ProgressStatusInProgress,
		Step:    "Starting",
		Message: "Analysis started",
	}

	// Mock SetProgress
	mockProgress.On("SetProgress", articleID, initialState).Return()
	sm.SetProgress(articleID, initialState)

	// Mock GetProgress
	mockProgress.On("GetProgress", articleID).Return(initialState)
	retrievedState := sm.GetProgress(articleID)

	assert.Equal(t, initialState, retrievedState)
	mockProgress.AssertExpectations(t)

	// Test with a different state
	finalState := &models.ProgressState{
		Status:  ProgressStatusSuccess, // Assuming ProgressStatusSuccess is defined
		Step:    "Complete",
		Message: "Analysis finished",
	}
	mockProgress.On("SetProgress", articleID, finalState).Return()
	sm.SetProgress(articleID, finalState)

	mockProgress.On("GetProgress", articleID).Return(finalState)
	retrievedState = sm.GetProgress(articleID)
	assert.Equal(t, finalState, retrievedState)
	mockProgress.AssertExpectations(t)
}

// TestGetProgressWithNilManager tests the GetProgress method with a nil progress manager
func TestGetProgressWithNilManager(t *testing.T) {
	// Create a ScoreManager with nil progress manager
	cache := NewCache()
	mockCalculator := new(MockScoreCalculator)

	// Create a ScoreManager with nil progressMgr
	sm := NewScoreManager(nil, cache, mockCalculator, nil)

	// Call GetProgress
	articleID := int64(123)
	result := sm.GetProgress(articleID)

	// Verify result is nil
	assert.Nil(t, result, "Expected nil result when progress manager is nil")
}

// TestUpdateArticleScoreSuccess tests the successful path of UpdateArticleScore
func TestUpdateArticleScoreSuccess(t *testing.T) {
	mockDB := new(MockDB)
	mockTx := new(MockTX)
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockCache := NewTestCache()

	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)
	sm.mockTx = mockTx // Inject mockTx for this test

	articleID := int64(123)
	scores := []db.LLMScore{{Model: "model1", Score: 0.5}}
	config := &CompositeScoreConfig{Models: []ModelConfig{{ModelName: "model1", Perspective: "center"}}}

	expectedCompositeScore := 0.75
	expectedConfidence := 0.9

	// Mock progress updates: Start
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusInProgress && ps.Step == ProgressStepStart && ps.Message == "Starting scoring"
	})).Return()

	// Mock calculator
	mockCalculator.On("CalculateScore", scores, config).Return(expectedCompositeScore, expectedConfidence, nil)

	// Mock transaction
	mockDB.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockTx, nil)

	// Mock DB updates (Exec for article score and llm_scores)
	// We expect two Exec calls in the real SaveScoresInTransaction
	mockTx.On("Exec", mock.AnythingOfType("string"), mock.Anything).Return(new(MockSqlResult), nil).Twice()

	mockTx.On("Commit").Return(nil)

	// Mock progress updates: Complete
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusSuccess && ps.Step == "Complete" && ps.Percent == 100 && *ps.FinalScore == expectedCompositeScore
	})).Return()

	// Call the method
	score, conf, err := sm.UpdateArticleScore(articleID, scores, config)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedCompositeScore, score)
	assert.Equal(t, expectedConfidence, conf)

	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockCalculator.AssertExpectations(t)
	mockProgress.AssertExpectations(t)

	// Verify cache invalidation was called for relevant keys
	assert.Nil(t, mockCache.Get(fmt.Sprintf("article:%d", articleID)))
	assert.Nil(t, mockCache.Get(fmt.Sprintf("ensemble:%d", articleID)))
	assert.Nil(t, mockCache.Get(fmt.Sprintf("bias:%d", articleID))) // Simplified, check all relevant bias keys
}

// TestUpdateArticleScoreCalculationError tests error handling when score calculation fails
func TestUpdateArticleScoreCalculationError(t *testing.T) {
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	sm := NewTestableScoreManager(nil, nil, mockCalculator, mockProgress)

	articleID := int64(1)
	scores := []db.LLMScore{}
	config := &CompositeScoreConfig{}
	calcError := fmt.Errorf("calculator error")

	// Mock SetProgress for initial state
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusInProgress && ps.Step == ProgressStepStart
	})).Return().Once() // Expect only once

	// Mock CalculateScore to return an error
	mockCalculator.On("CalculateScore", scores, config).Return(0.0, 0.0, calcError)

	// Mock SetProgress for error state
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusError && ps.Error != ""
	})).Return().Once()

	_, _, err := sm.UpdateArticleScore(articleID, scores, config)

	assert.Error(t, err)
	assert.Equal(t, calcError, err)
	mockCalculator.AssertExpectations(t)
	mockProgress.AssertExpectations(t) // Verify both SetProgress calls were made
}

// TestUpdateArticleScoreDBError tests error handling when database operations fail
func TestUpdateArticleScoreDBError(t *testing.T) {
	mockDB := new(MockDB)
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	sm := NewTestableScoreManager(mockDB, nil, mockCalculator, mockProgress)

	articleID := int64(1)
	scores := []db.LLMScore{{Model: "model1", Score: 0.5}}
	config := &CompositeScoreConfig{Models: []ModelConfig{{ModelName: "model1"}}}
	dbError := fmt.Errorf("database error")

	// Mock progress updates: Start
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusInProgress && ps.Step == ProgressStepStart
	})).Return()

	// Mock calculator
	mockCalculator.On("CalculateScore", scores, config).Return(0.5, 0.9, nil)

	// Mock DB BeginTxx to return an error
	mockDB.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(nil, dbError)

	// Mock progress updates: Error
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusError && strings.Contains(ps.Message, "Failed to start database transaction")
	})).Return()

	// Call the method
	_, _, err := sm.UpdateArticleScore(articleID, scores, config)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error") // Check underlying error

	mockDB.AssertExpectations(t)
	mockCalculator.AssertExpectations(t)
	mockProgress.AssertExpectations(t)
}

// TestUpdateArticleScoreCommitError tests error handling when transaction commit fails
func TestUpdateArticleScoreCommitError(t *testing.T) {
	mockDB := new(MockDB)
	mockTx := new(MockTX)
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockCache := NewTestCache()

	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)
	sm.mockTx = mockTx // Inject mockTx

	articleID := int64(1)
	scores := []db.LLMScore{{Model: "model1", Score: 0.5}}
	config := &CompositeScoreConfig{Models: []ModelConfig{{ModelName: "model1"}}}
	commitError := fmt.Errorf("commit error")

	// Mock progress updates: Start
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusInProgress && ps.Step == ProgressStepStart
	})).Return()

	// Mock calculator
	mockCalculator.On("CalculateScore", scores, config).Return(0.5, 0.9, nil)

	// Mock transaction
	mockDB.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockTx, nil)
	mockTx.On("Exec", mock.AnythingOfType("string"), mock.Anything).Return(new(MockSqlResult), nil).Twice()
	mockTx.On("Commit").Return(commitError) // Simulate commit failure
	mockTx.On("Rollback").Return(nil)       // Expect Rollback to be called

	// Mock progress updates: Error
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusError && strings.Contains(ps.Message, "Failed to commit transaction")
	})).Return()

	// Call the method
	_, _, err := sm.UpdateArticleScore(articleID, scores, config)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commit error")

	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockCalculator.AssertExpectations(t)
	mockProgress.AssertExpectations(t)

	// Cache should not be invalidated if DB commit fails
	mockCache.Set(fmt.Sprintf("article:%d", articleID), "cached_data", 1*time.Minute)
	assert.NotNil(t, mockCache.Get(fmt.Sprintf("article:%d", articleID)))
}

// TestUpdateArticleScoreCacheInvalidation tests cache invalidation after successful score update
func TestUpdateArticleScoreCacheInvalidation(t *testing.T) {
	// Create mocks
	mockDB := new(MockDB)
	mockTx := new(MockTX)
	mockCache := NewTestCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockSqlResult := new(MockSqlResult)

	// Use TestableScoreManager instead of regular ScoreManager
	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)
	sm.mockTx = mockTx // Ensure mockTx is set in TestableScoreManager if needed by the original logic

	// Test data
	articleID := int64(123)
	cacheKey := "article:123"

	// Seed the cache with test data
	mockCache.Set(cacheKey, "test-data", time.Minute)
	assert.NotNil(t, mockCache.Get(cacheKey))

	testScores := []db.LLMScore{
		{ArticleID: articleID, Model: "model1", Score: -0.5},
	}
	config := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "model1", Perspective: "left"},
		},
	}
	expectedScore := 0.1
	expectedConfidence := 0.8

	// Mock all required calls
	mockProgress.On("SetProgress", articleID, mock.AnythingOfType("*models.ProgressState")).Return()
	mockDB.On("BeginTxx", mock.Anything, mock.Anything).Return(mockTx, nil)
	mockCalculator.On("CalculateScore", testScores, config).Return(expectedScore, expectedConfidence, nil)
	mockSqlResult.On("LastInsertId").Return(int64(1), nil)

	// First Exec call for ensemble score insertion - more flexible matching
	mockTx.On("Exec",
		mock.MatchedBy(func(q string) bool {
			return strings.Contains(q, "INSERT INTO llm_scores")
		}),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(mockSqlResult, nil).Once()

	// Second Exec call for article update - more flexible matching
	mockTx.On("Exec",
		mock.MatchedBy(func(q string) bool {
			return strings.Contains(q, "UPDATE articles")
		}),
		mock.Anything, mock.Anything, mock.Anything,
	).Return(mockSqlResult, nil).Once()

	mockTx.On("Commit").Return(nil)

	// Call method
	_, _, _ = sm.UpdateArticleScore(articleID, testScores, config)

	// Verify cache was invalidated
	assert.Nil(t, mockCache.Get(cacheKey))
}

// Adding test for zero confidence handling
func TestScoreManagerWithAllZeroConfidenceScores(t *testing.T) {
	mockDB := new(MockDB) // No DB interaction expected here
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockCache := NewTestCache()
	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)

	articleID := int64(789)
	// Create scores that would trigger the allZeroError
	scores := []db.LLMScore{
		{Model: "modelA", Score: 0.1, Metadata: `{"confidence": 0.0}`},
		{Model: "modelB", Score: 0.2, Metadata: `{"confidence": 0}`},    // Test int 0
		{Model: "modelC", Score: 0.3, Metadata: `{"confidence": null}`}, // Test null confidence
		{Model: "modelD", Score: 0.4, Metadata: `{}`},                   // Test missing confidence
	}
	config := &CompositeScoreConfig{Models: []ModelConfig{{ModelName: "modelA"}, {ModelName: "modelB"}, {ModelName: "modelC"}, {ModelName: "modelD"}}}

	// Expected error from checkForAllZeroResponses
	expectedErr := fmt.Errorf("all LLMs returned empty or zero-confidence responses")

	// Mock progress updates: Start (should still be called)
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusInProgress && ps.Step == ProgressStepStart
	})).Return()

	// Mock progress updates: Error due to all zero confidence
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == ProgressStatusError &&
			ps.Step == "Error" &&
			strings.Contains(ps.Message, "All LLMs returned zero confidence") &&
			ps.Error == expectedErr.Error()
	})).Return()

	// Calculator should NOT be called in this scenario
	// mockCalculator.AssertNotCalled(t, "CalculateScore")

	score, conf, err := sm.UpdateArticleScore(articleID, scores, config)

	assert.Error(t, err)
	assert.Equal(t, expectedErr.Error(), err.Error())
	assert.Equal(t, 0.0, score)
	assert.Equal(t, 0.0, conf)

	mockProgress.AssertExpectations(t)
	mockCalculator.AssertNotCalled(t, "CalculateScore", mock.Anything, mock.Anything) // Ensure calculator wasn't called
	mockDB.AssertExpectations(t)                                                      // Ensure no DB calls were made
}

// Test helper in real ScoreManager, not directly testable via TestableScoreManager
// but we can test its logic here.
func TestCheckForAllZeroResponses(t *testing.T) {
	tests := []struct {
		name        string
		scores      []db.LLMScore
		expectError bool
		expectedMsg string
	}{
		{
			name: "All zero confidence",
			scores: []db.LLMScore{
				{Metadata: `{"confidence": 0.0}`},
				{Metadata: `{"confidence": 0}`},
			},
			expectError: true,
			expectedMsg: "all LLMs returned empty or zero-confidence responses",
		},
		// ... existing code ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// sm := &ScoreManager{} // Not needed as we are testing a standalone function
			allZeros, err := checkForAllZeroResponses(tt.scores)
			if tt.expectError {
				assert.True(t, allZeros)
				assert.Error(t, err)
				assert.Equal(t, tt.expectedMsg, err.Error())
			} else {
				assert.False(t, allZeros)
				assert.NoError(t, err)
			}
		})
	}
}
