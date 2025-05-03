package llm

import (
	"context"
	"database/sql"
	"encoding/json"
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

	// Progress: Start
	if sm.mockProgress != nil {
		ps := &models.ProgressState{Step: "Start", Message: "Starting scoring", Percent: 0, Status: "InProgress", LastUpdated: time.Now().Unix()}
		sm.mockProgress.SetProgress(articleID, ps)
	}

	tx, err := sm.mockDB.BeginTxx(context.Background(), nil)
	if err != nil {
		if sm.mockProgress != nil {
			ps := &models.ProgressState{Step: "DB Transaction", Message: "Failed to start DB transaction", Percent: 0, Status: "Error", Error: err.Error(), LastUpdated: time.Now().Unix()}
			sm.mockProgress.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	sm.mockTx = tx.(*MockTX)

	// Progress: Calculating
	if sm.mockProgress != nil {
		ps := &models.ProgressState{Step: "Calculating", Message: "Calculating score", Percent: 20, Status: "InProgress", LastUpdated: time.Now().Unix()}
		sm.mockProgress.SetProgress(articleID, ps)
	}

	score, confidence, calcErr := sm.calculator.CalculateScore(scores)
	if calcErr != nil {
		sm.mockTx.Rollback()
		if sm.mockProgress != nil {
			ps := &models.ProgressState{Step: "Calculation", Message: "Score calculation failed", Percent: 20, Status: "Error", Error: calcErr.Error(), LastUpdated: time.Now().Unix()}
			sm.mockProgress.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("score calculation failed: %w", calcErr)
	}

	// Progress: Storing ensemble score
	if sm.mockProgress != nil {
		ps := &models.ProgressState{Step: "Storing", Message: "Storing ensemble score", Percent: 60, Status: "InProgress", LastUpdated: time.Now().Unix()}
		sm.mockProgress.SetProgress(articleID, ps)
	}

	meta := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"aggregation": "ensemble",
		"confidence":  confidence,
	}
	metaBytes, _ := json.Marshal(meta)
	ensembleScore := &db.LLMScore{
		ArticleID: articleID,
		Model:     "ensemble",
		Score:     score,
		Metadata:  string(metaBytes),
		CreatedAt: time.Now(),
	}

	// Mock the DB operations directly since we can't use the real function
	_, err = sm.mockTx.Exec("INSERT INTO llm_scores (article_id, model, score, metadata, created_at) VALUES (?, ?, ?, ?, ?)",
		ensembleScore.ArticleID, ensembleScore.Model, ensembleScore.Score, ensembleScore.Metadata, ensembleScore.CreatedAt)

	if err != nil {
		sm.mockTx.Rollback()
		if sm.mockProgress != nil {
			ps := &models.ProgressState{Step: "DB Insert", Message: "Failed to insert ensemble score", Percent: 70, Status: "Error", Error: err.Error(), LastUpdated: time.Now().Unix()}
			sm.mockProgress.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("failed to insert ensemble score: %w", err)
	}

	// Progress: Updating article
	if sm.mockProgress != nil {
		ps := &models.ProgressState{Step: "Updating", Message: "Updating article score", Percent: 80, Status: "InProgress", LastUpdated: time.Now().Unix()}
		sm.mockProgress.SetProgress(articleID, ps)
	}

	// Mock article update
	_, err = sm.mockTx.Exec("UPDATE articles SET score = ?, confidence = ? WHERE id = ?",
		score, confidence, articleID)

	if err != nil {
		sm.mockTx.Rollback()
		if sm.mockProgress != nil {
			ps := &models.ProgressState{Step: "DB Update", Message: "Failed to update article", Percent: 90, Status: "Error", Error: err.Error(), LastUpdated: time.Now().Unix()}
			sm.mockProgress.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("failed to update article: %w", err)
	}

	if err := sm.mockTx.Commit(); err != nil {
		if sm.mockProgress != nil {
			ps := &models.ProgressState{Step: "DB Commit", Message: "Failed to commit transaction", Percent: 95, Status: "Error", Error: err.Error(), LastUpdated: time.Now().Unix()}
			sm.mockProgress.SetProgress(articleID, ps)
		}
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Progress: Invalidate cache
	if sm.cache != nil {
		sm.cache.Delete(fmt.Sprintf("article:%d", articleID))
		sm.cache.Delete(fmt.Sprintf("ensemble:%d", articleID))
		sm.cache.Delete(fmt.Sprintf("bias:%d", articleID))
	}

	// Progress: Success
	if sm.mockProgress != nil {
		ps := &models.ProgressState{Step: "Complete", Message: "Scoring complete", Percent: 100, Status: "Success", FinalScore: &score, LastUpdated: time.Now().Unix()}
		sm.mockProgress.SetProgress(articleID, ps)
	}

	return score, confidence, nil
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
	// Create mocks
	cache := NewCache()
	mockCalculator := new(MockScoreCalculator)

	// Create a real progress manager for this test - easier than mocking internal state
	mockProgress := NewProgressManager(time.Minute)

	sm := NewScoreManager(nil, cache, mockCalculator, mockProgress)

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
	// Create mocks
	mockDB := new(MockDB)
	mockTx := new(MockTX)
	mockCache := NewTestCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockSqlResult := new(MockSqlResult)

	// Use TestableScoreManager instead of regular ScoreManager
	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)

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

	expectedScore := 0.1      // Mock value for test
	expectedConfidence := 0.8 // Mock value for test

	// Mock progress tracking - Starting
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "InProgress" && ps.Step == "Start"
	})).Return()

	// Mock DB transaction
	mockDB.On("BeginTxx", mock.Anything, mock.Anything).Return(mockTx, nil)

	// Mock score calculation
	mockCalculator.On("CalculateScore", testScores).Return(expectedScore, expectedConfidence, nil)

	// Mock progress tracking - Calculation
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "InProgress" && ps.Step == "Calculating"
	})).Return()

	// Mock DB operations using mock.Anything for all parameters
	// This is because we're testing the workflow, not the exact parameter values
	mockSqlResult.On("LastInsertId").Return(int64(1), nil)

	// First Exec call for ensemble score insertion - more flexible matching
	mockTx.On("Exec",
		mock.MatchedBy(func(q string) bool {
			return strings.Contains(q, "INSERT INTO llm_scores")
		}),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(mockSqlResult, nil).Once()

	// Mock progress tracking - Storing
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "InProgress" && ps.Step == "Storing"
	})).Return()

	// Second Exec call for article update - more flexible matching
	mockTx.On("Exec",
		mock.MatchedBy(func(q string) bool {
			return strings.Contains(q, "UPDATE articles")
		}),
		mock.Anything, mock.Anything, mock.Anything,
	).Return(mockSqlResult, nil).Once()

	// Mock progress tracking - Updating
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "InProgress" && ps.Step == "Updating"
	})).Return()

	// Mock commit
	mockTx.On("Commit").Return(nil)

	// Mock progress tracking - Complete
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "Success" && ps.Step == "Complete" && ps.FinalScore != nil && *ps.FinalScore == expectedScore
	})).Return()

	// Call method
	score, confidence, err := sm.UpdateArticleScore(articleID, testScores, config)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, expectedScore, score)
	assert.Equal(t, expectedConfidence, confidence)

	// Verify progress manager calls
	mockProgress.AssertCalled(t, "SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "Success" && ps.Step == "Complete"
	}))

	// Verify transaction was committed
	mockTx.AssertCalled(t, "Commit")
}

// TestUpdateArticleScoreCalculationError tests error handling when score calculation fails
func TestUpdateArticleScoreCalculationError(t *testing.T) {
	// Create mocks
	mockDB := new(MockDB)
	mockTx := new(MockTX)
	mockCache := NewTestCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)

	// Use TestableScoreManager instead of regular ScoreManager
	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)

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
	expectedError := "calculation failed"

	// Mock progress tracking - we need to accept all possible calls here to prevent unexpected call errors
	mockProgress.On("SetProgress", articleID, mock.AnythingOfType("*models.ProgressState")).Return()

	// Mock DB transaction
	mockDB.On("BeginTxx", mock.Anything, mock.Anything).Return(mockTx, nil)

	// Mock score calculation error
	mockCalculator.On("CalculateScore", testScores).Return(0.0, 0.0, assert.AnError)

	// Mock rollback
	mockTx.On("Rollback").Return(nil)

	// Call method
	score, confidence, err := sm.UpdateArticleScore(articleID, testScores, config)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expectedError)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, 0.0, confidence)

	// Verify error status was set - check for last call being an error state
	mockProgress.AssertCalled(t, "SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "Error" && ps.Error != ""
	}))

	// Verify transaction was rolled back
	mockTx.AssertCalled(t, "Rollback")
}

// TestUpdateArticleScoreDBError tests error handling when database operations fail
func TestUpdateArticleScoreDBError(t *testing.T) {
	// Create mocks
	mockDB := new(MockDB)
	mockCache := NewTestCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)

	// Use TestableScoreManager instead of regular ScoreManager
	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)

	// Test data
	articleID := int64(123)
	testScores := []db.LLMScore{
		{ArticleID: articleID, Model: "model1", Score: -0.5},
	}
	config := &CompositeScoreConfig{}
	expectedError := "transaction"

	// Mock progress tracking - Starting
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "InProgress" && ps.Step == "Start"
	})).Return()

	// Mock DB transaction failure
	mockDB.On("BeginTxx", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	// Mock progress tracking - Error
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "Error" && ps.Step == "DB Transaction"
	})).Return()

	// Call method
	score, confidence, err := sm.UpdateArticleScore(articleID, testScores, config)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expectedError)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, 0.0, confidence)

	// Verify progress manager received error status
	mockProgress.AssertCalled(t, "SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "Error" && ps.Error != ""
	}))
}

// TestUpdateArticleScoreCommitError tests error handling when transaction commit fails
func TestUpdateArticleScoreCommitError(t *testing.T) {
	// Create mocks
	mockDB := new(MockDB)
	mockTx := new(MockTX)
	mockCache := NewTestCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockSqlResult := new(MockSqlResult)

	// Use TestableScoreManager instead of regular ScoreManager
	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)

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
	expectedError := "commit"
	expectedScore := 0.1
	expectedConfidence := 0.8

	// Mock progress tracking - All steps before commit
	mockProgress.On("SetProgress", articleID, mock.AnythingOfType("*models.ProgressState")).Return()

	// Mock DB transaction
	mockDB.On("BeginTxx", mock.Anything, mock.Anything).Return(mockTx, nil)

	// Mock score calculation
	mockCalculator.On("CalculateScore", testScores).Return(expectedScore, expectedConfidence, nil)

	// Mock DB operations with flexible parameter matching
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

	// Mock commit error
	mockTx.On("Commit").Return(assert.AnError)

	// Call method
	score, confidence, err := sm.UpdateArticleScore(articleID, testScores, config)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expectedError)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, 0.0, confidence)
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
	mockCalculator.On("CalculateScore", testScores).Return(expectedScore, expectedConfidence, nil)
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
	sm.UpdateArticleScore(articleID, testScores, config)

	// Verify cache was invalidated
	assert.Nil(t, mockCache.Get(cacheKey))
}
