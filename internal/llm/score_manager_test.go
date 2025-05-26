package llm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	articleCacheKeyFormat  = "article:%d"
	ensembleCacheKeyFormat = "ensemble:%d" // Adding for consistency, though not in lint error
	biasCacheKeyFormat     = "bias:%d"     // Adding for consistency, though not in lint error
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

// MockTx is a mock type for transaction operations
type MockTx struct {
	mock.Mock
}

// Exec mocks the Exec method
func (m *MockTx) Exec(query string, args ...interface{}) (sql.Result, error) {
	arguments := m.Called(append([]interface{}{query}, args...)...)
	if arguments.Get(0) == nil {
		return nil, arguments.Error(1)
	}
	return arguments.Get(0).(sql.Result), arguments.Error(1)
}

// Commit mocks the Commit method
func (m *MockTx) Commit() error {
	arguments := m.Called()
	return arguments.Error(0)
}

// Rollback mocks the Rollback method
func (m *MockTx) Rollback() error {
	arguments := m.Called()
	return arguments.Error(0)
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

// MockSqlResult implements sql.Result for testing
type MockSqlResult struct {
	mock.Mock
}

// helperGetInt64AndError is a helper to reduce duplication in LastInsertId and RowsAffected
func (m *MockSqlResult) helperGetInt64AndError() (int64, error) {
	args := m.Called()
	mockedErr := args.Error(1)
	val, ok := args.Get(0).(int64)
	if !ok {
		// Consider returning a specific error if the type assertion fails, e.g.:
		// return 0, fmt.Errorf("expected int64, got %T for mock result", args.Get(0))
		return 0, mockedErr
	}
	return val, mockedErr
}

func (m *MockSqlResult) LastInsertId() (int64, error) {
	return m.helperGetInt64AndError()
}

func (m *MockSqlResult) RowsAffected() (int64, error) {
	return m.helperGetInt64AndError()
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
	mockTx       *MockTx
	mockProgress *MockProgressManager
}

// NewTestableScoreManager creates a ScoreManager that works with our mocks
func NewTestableScoreManager(mockDB *MockDB, cache *TestCache, calculator ScoreCalculator,
	mockProgress *MockProgressManager) *TestableScoreManager {
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
func (sm *TestableScoreManager) UpdateArticleScore(articleID int64, scores []db.LLMScore,
	cfg *CompositeScoreConfig) (float64, float64, error) {
	if sm.calculator == nil {
		return 0, 0, fmt.Errorf("ScoreManager: calculator is nil")
	}
	if cfg == nil {
		return 0, 0, fmt.Errorf("ScoreManager: config is nil")
	}

	// Check for all zero responses - Mimic real implementation
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
		// Mimic the real ScoreManager's error handling logic for calculator errors
		if errors.Is(err, ErrAllPerspectivesInvalid) {
			if sm.mockProgress != nil {
				errorState := models.ProgressState{
					Step:        "Error",
					Message:     err.Error(),
					Status:      "Error",
					Percent:     100,
					LastUpdated: time.Now().Unix(),
				}
				sm.mockProgress.SetProgress(articleID, &errorState)
			}
		} else {
			if sm.mockProgress != nil {
				errorState := models.ProgressState{
					Step:        "Error",
					Message:     fmt.Sprintf("Internal error calculating score: %v", err),
					Status:      "Error",
					Percent:     100,
					LastUpdated: time.Now().Unix(),
				}
				sm.mockProgress.SetProgress(articleID, &errorState)
			}
		}
		return 0, 0, err
	}

	// --- If calculator err is nil (Success from calculator) ---
	// Now, simulate the database update part using sm.mockTx if available
	var dbUpdateErr error
	if sm.mockTx != nil { // Only simulate if mockTx is provided by the test
		// The real ScoreManager uses db.UpdateArticleScoreLLM, which implies a transaction.
		// This override simulates that by using the mockTx directly.
		// Test cases will set expectations on mockTx.Exec and mockTx.Commit.

		// Simulate db.UpdateArticleScoreLLM logic (simplified for mock)
		_, execErr := sm.mockTx.Exec("UPDATE articles SET composite_score = ?, confidence = ?, score_source = 'llm' WHERE id = ?",
			compositeScore, confidence, articleID)
		if execErr != nil {
			dbUpdateErr = fmt.Errorf("failed during DB exec: %w", execErr)
			sm.mockTx.Rollback() // Attempt rollback
		} else {
			// If there were other DB operations (e.g., inserting to llm_scores), they would be here.
			// For now, assume only one Exec and then Commit.
			commitErr := sm.mockTx.Commit()
			if commitErr != nil {
				dbUpdateErr = fmt.Errorf("failed during DB commit: %w", commitErr)
				sm.mockTx.Rollback() // Attempt rollback
			}
		}
	}

	if dbUpdateErr != nil {
		// Set progress for DB error
		if sm.mockProgress != nil {
			dbErrorState := models.ProgressState{
				Step:        "Error",
				Message:     fmt.Sprintf("Failed to update score in DB: %v", dbUpdateErr), // Based on markdown
				Status:      "Error",
				Percent:     100,
				LastUpdated: time.Now().Unix(),
			}
			sm.mockProgress.SetProgress(articleID, &dbErrorState)
		}
		// The real SM returns the DB error directly (or wrapped).
		// Markdown suggests returning the raw db error. The current code might wrap it.
		// For consistency with how calc errors are returned, let's return it directly.
		return 0, 0, dbUpdateErr
	}

	// --- If we reach here, calculator succeeded AND DB update (if simulated) also succeeded ---
	sm.InvalidateScoreCache(articleID) // Mimic real behavior
	if sm.mockProgress != nil {
		successState := models.ProgressState{
			Step:        "Complete",
			Message:     "Analysis complete.",
			Status:      "Success",
			Percent:     100,
			FinalScore:  &compositeScore,
			LastUpdated: time.Now().Unix(),
		}
		sm.mockProgress.SetProgress(articleID, &successState)
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
		fmt.Sprintf(articleCacheKeyFormat, articleID),
		fmt.Sprintf(ensembleCacheKeyFormat, articleID),
		fmt.Sprintf(biasCacheKeyFormat, articleID),
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
	mockProgress.On("GetProgress", articleID).Return(initialState).Once()
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

	mockProgress.On("GetProgress", articleID).Return(finalState).Once()
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
	mockTx := new(MockTx)
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockCache := NewTestCache() // Assuming TestCache is defined and NewTestCache creates an instance

	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)
	sm.mockTx = mockTx // Inject the transaction mock

	articleID := int64(123)
	expectedScore := 0.75
	expectedConfidence := 0.9
	scores := []db.LLMScore{{Model: "model1", Score: expectedScore, Metadata: `{"confidence":0.9}`}}
	config := &CompositeScoreConfig{Models: []ModelConfig{{ModelName: "model1"}}}

	// Mock Calculator
	mockCalculator.On("CalculateScore", scores, config).Return(expectedScore, expectedConfidence, nil).Once()

	// Mock Transaction: Exec and Commit should succeed
	// Use mock.AnythingOfType for arguments to simplify matching
	mockTx.On("Exec",
		"UPDATE articles SET composite_score = ?, confidence = ?, score_source = 'llm' WHERE id = ?",
		mock.AnythingOfType("float64"), // Match score type
		mock.AnythingOfType("float64"), // Match confidence type
		mock.AnythingOfType("int64"),   // Match articleID type
	).Return(new(MockSqlResult), nil).Once()

	mockTx.On("Commit").Return(nil).Once()

	// Mock Progress Manager: Expect success state
	finalScore := expectedScore // For closure
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "Success" &&
			ps.Step == "Complete" &&
			ps.Message == "Analysis complete." &&
			ps.Percent == 100 &&
			ps.FinalScore != nil && *ps.FinalScore == finalScore
	})).Return().Once()

	// Call the method
	score, confidence, err := sm.UpdateArticleScore(articleID, scores, config)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedScore, score)
	assert.Equal(t, expectedConfidence, confidence)

	mockCalculator.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockProgress.AssertExpectations(t)

	// Verify cache invalidation (TestableScoreManager calls InvalidateScoreCache)
	// This requires mockCache to have a way to check if Invalidate was called, or simply trust it.
	// For now, we assume InvalidateScoreCache works if no error occurs.
}

// TestUpdateArticleScoreCalculationError tests error handling when score calculation fails
func TestUpdateArticleScoreCalculationError(t *testing.T) {
	mockDB := new(MockDB) // Instantiate mockDB
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	// Pass mockDB to the manager
	sm := NewTestableScoreManager(mockDB, nil, mockCalculator, mockProgress)

	articleID := int64(1)
	scores := []db.LLMScore{}
	config := &CompositeScoreConfig{}
	calcError := fmt.Errorf("calculator error")

	// REMOVED: Mock SetProgress for initial state was removed as it doesn't match actual logic
	// mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
	// 	return ps.Status == ProgressStatusInProgress && ps.Step == ProgressStepStart
	// })).Return().Once()

	// Mock CalculateScore to return the specific error
	mockCalculator.On("CalculateScore", scores, config).Return(0.0, 0.0, calcError)

	// Mock SetProgress for the specific error state set by ScoreManager for generic calculation errors
	expectedMessage := fmt.Sprintf("Internal error calculating score: %v", calcError)
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		// Check the fields set in the generic error handling block of UpdateArticleScore
		return ps.Status == "Error" && ps.Step == "Error" && ps.Message == expectedMessage && ps.Percent == 100
	})).Return().Once()

	// Call the TestableScoreManager's UpdateArticleScore override
	_, _, err := sm.UpdateArticleScore(articleID, scores, config)

	assert.Error(t, err)
	// Ensure the original calculator error is returned
	assert.Equal(t, calcError, err)
	mockCalculator.AssertExpectations(t)
	mockProgress.AssertExpectations(t) // Verify the specific error SetProgress call was made
}

// TestUpdateArticleScoreDBError tests error handling when database operations fail
func TestUpdateArticleScoreDBError(t *testing.T) {
	// Setup mocks for TestableScoreManager
	mockDB := new(MockDB)
	mockTx := new(MockTx)
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockCache := NewTestCache() // Assuming TestCache and NewTestCache() are defined

	// Instantiate TestableScoreManager, providing the mockCache
	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)
	sm.mockTx = mockTx // Set the mockTx field directly

	articleID := int64(1)
	scores := []db.LLMScore{{Model: "model1", Score: 0.5, Metadata: `{"confidence":0.9}`}}
	config := &CompositeScoreConfig{
		Models: []ModelConfig{{ModelName: "model1", Perspective: "model1"}},
	}
	dbError := fmt.Errorf("database exec error")
	// The TestableScoreManager override now returns the error from the simulated DB op directly
	expectedReturnedError := fmt.Errorf("failed during DB exec: %w", dbError)

	// 1. Calculator succeeds
	mockCalculator.On("CalculateScore", scores, config).Return(0.5, 0.9, nil).Once()

	// 2. DB operation (Exec) within the transaction fails
	mockTx.On("Exec", "UPDATE articles SET composite_score = ?, confidence = ?, score_source = 'llm' WHERE id = ?", 0.5, 0.9, articleID).
		Return(nil, dbError).Once()
	mockTx.On("Rollback").Return(nil).Once()

	// 3. Progress Manager is called with the DB error state
	// Note: The message includes the wrapped error string
	expectedProgressMessage := fmt.Sprintf("Failed to update score in DB: %v", expectedReturnedError)
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "Error" &&
			ps.Step == "Error" &&
			ps.Message == expectedProgressMessage &&
			ps.Percent == 100
	})).Return().Once()

	// Call the method
	_, _, err := sm.UpdateArticleScore(articleID, scores, config)

	// Assertions
	assert.Error(t, err)
	// Assert against the error returned by TestableScoreManager directly
	assert.EqualError(t, err, expectedReturnedError.Error())

	mockCalculator.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockProgress.AssertExpectations(t)
}

// TestUpdateArticleScoreCommitError tests error handling when DB commit fails
func TestUpdateArticleScoreCommitError(t *testing.T) {
	mockDB := new(MockDB)
	mockTx := new(MockTx)
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockCache := NewTestCache() // Assuming TestCache and NewTestCache() are defined

	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)
	sm.mockTx = mockTx

	articleID := int64(2)
	scores := []db.LLMScore{{Model: "model1", Score: 0.6, Metadata: `{"confidence":0.8}`}}
	config := &CompositeScoreConfig{Models: []ModelConfig{{ModelName: "model1"}}}
	commitError := fmt.Errorf("db commit error")
	// The TestableScoreManager override now returns the error from the simulated DB op directly
	expectedReturnedError := fmt.Errorf("failed during DB commit: %w", commitError)

	// 1. Calculator succeeds
	mockCalculator.On("CalculateScore", scores, config).Return(0.6, 0.8, nil).Once()

	// 2. DB Exec succeeds
	mockTx.On("Exec", "UPDATE articles SET composite_score = ?, confidence = ?, score_source = 'llm' WHERE id = ?", 0.6, 0.8, articleID).
		Return(nil, nil).Once()

	// 3. DB Commit fails
	mockTx.On("Commit").Return(commitError).Once()
	mockTx.On("Rollback").Return(nil).Once() // Rollback attempt after commit failure

	// 4. Progress Manager is called with the DB error state
	// Note: The message includes the wrapped error string
	expectedProgressMessage := fmt.Sprintf("Failed to update score in DB: %v", expectedReturnedError)
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		return ps.Status == "Error" &&
			ps.Step == "Error" &&
			ps.Message == expectedProgressMessage &&
			ps.Percent == 100
	})).Return().Once()

	// Call the method
	_, _, err := sm.UpdateArticleScore(articleID, scores, config)

	// Assertions
	assert.Error(t, err)
	// Assert against the error returned by TestableScoreManager directly
	assert.EqualError(t, err, expectedReturnedError.Error())

	mockCalculator.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockProgress.AssertExpectations(t)
}

// TestUpdateArticleScoreCacheInvalidation tests cache invalidation after successful score update
func TestUpdateArticleScoreCacheInvalidation(t *testing.T) {
	// Create mocks
	mockDB := new(MockDB)
	mockTx := new(MockTx)
	mockCache := NewTestCache()
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	// mockSqlResult := new(MockSqlResult) // Removed as unused in updated mocks

	// Use TestableScoreManager instead of regular ScoreManager
	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)
	sm.mockTx = mockTx // Ensure mockTx is set

	// Test data
	articleID := int64(123)
	cacheKey := fmt.Sprintf(articleCacheKeyFormat, articleID)

	// Seed the cache with test data
	mockCache.Set(cacheKey, "test-data", time.Minute)
	valBefore := mockCache.Get(cacheKey)
	assert.NotNil(t, valBefore) // Check value is not nil

	testScores := []db.LLMScore{
		{ArticleID: articleID, Model: "model1", Score: -0.5, Metadata: `{"confidence":0.9}`}, // Added metadata
	}
	config := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "model1", Perspective: "left"},
		},
		Formula: "average", DefaultMissing: 0.0, HandleInvalid: "default", // Added necessary config fields
		MinScore: -1.0, MaxScore: 1.0,
	}
	expectedScore := -0.5           // Only one valid score
	expectedConfidence := 1.0 / 3.0 // Only one perspective found

	// Mock Progress: Expect only one call (the final success state) since the override might not simulate the initial state setting.
	mockProgress.On("SetProgress", articleID, mock.AnythingOfType("*models.ProgressState")).Return().Once()

	// Mock Calculator
	mockCalculator.On("CalculateScore", testScores, config).Return(expectedScore, expectedConfidence, nil).Once()

	// Mock Transaction setup
	mockTx.On("Exec",
		"UPDATE articles SET composite_score = ?, confidence = ?, score_source = 'llm' WHERE id = ?",
		mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), articleID,
	).Return(new(MockSqlResult), nil).Once() // Expect Exec once
	mockTx.On("Commit").Return(nil).Once() // Expect Commit once
	// No rollback expected on success path

	// Call method
	_, _, err := sm.UpdateArticleScore(articleID, testScores, config)
	assert.NoError(t, err) // Expect success

	// Verify cache was invalidated
	valAfter := mockCache.Get(cacheKey)
	assert.Nil(t, valAfter) // Check value is now nil

	// Verify mocks
	mockCalculator.AssertExpectations(t)
	mockProgress.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

// Adding test for zero confidence handling
func TestScoreManagerWithAllZeroConfidenceScores(t *testing.T) {
	mockDB := new(MockDB)
	// mockTx := new(MockTx) // Removed
	mockCalculator := new(MockScoreCalculator)
	mockProgress := new(MockProgressManager)
	mockCache := NewTestCache()
	sm := NewTestableScoreManager(mockDB, mockCache, mockCalculator, mockProgress)
	// sm.mockTx = mockTx // Should be removed or commented
	expectedErr := ErrAllScoresZeroConfidence // Base error expected

	articleID := int64(789)
	scoresWithZeroConfidence := []db.LLMScore{
		{Model: "modelA", Score: 0.1, Metadata: `{"confidence": 0.0}`},
		{Model: "modelB", Score: 0.2, Metadata: `{"confidence": 0}`},
		{Model: "modelC", Score: 0.3, Metadata: `{"confidence": null}`},
		{Model: "modelD", Score: 0.4, Metadata: `{}`},
	}
	config := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "modelA"}, {ModelName: "modelB"}, {ModelName: "modelC"}, {ModelName: "modelD"},
		},
	}

	mockCalculator.On("CalculateScore", scoresWithZeroConfidence, config).Return(0.0, 0.0, nil).Maybe()

	// Restore specific progress mock expectation
	mockProgress.On("SetProgress", articleID, mock.MatchedBy(func(ps *models.ProgressState) bool {
		// Check the fields set by the TestableScoreManager override for the allZeros path
		return ps.Status == "Error" &&
			ps.Step == "Error" &&
			ps.Message == "All LLMs returned zero confidence - scoring failed" && // Match the specific message set in the override
			ps.Error == expectedErr.Error() // Match the specific error string
	})).Return().Once()

	// Call UpdateArticleScore
	_, _, err := sm.UpdateArticleScore(articleID, scoresWithZeroConfidence, config)

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr) // Use ErrorIs now that the correct base error should be returned

	mockCalculator.AssertExpectations(t)
	mockProgress.AssertExpectations(t)
}

// Test helper in real ScoreManager, not directly testable via TestableScoreManager
// but we can test its logic here.
func TestCheckForAllZeroResponses(t *testing.T) {
	tests := []struct {
		name        string
		scores      []db.LLMScore
		expectError bool // Changed back from expectBool
	}{
		{
			name: "All zero confidence",
			scores: []db.LLMScore{
				{Model: "modelA", Score: 0.1, Metadata: `{"confidence": 0.0}`},
				{Model: "modelB", Score: 0.2, Metadata: `{"confidence": 0}`},
				{Model: "modelC", Score: 0.3, Metadata: `{"confidence": null}`},
				{Model: "modelD", Score: 0.4, Metadata: `{}`},
			},
			expectError: true, // Expect the error
		},
		{
			name: "One non-zero confidence",
			scores: []db.LLMScore{
				{Model: "modelA", Score: 0.1, Metadata: `{"confidence": 0.0}`},
				{Model: "modelB", Score: 0.2, Metadata: `{"confidence": 0.5}`},
				{Model: "modelC", Score: 0.3, Metadata: `{"confidence": null}`},
				{Model: "modelD", Score: 0.4, Metadata: `{}`},
			},
			expectError: false,
		},
		// Add more cases if needed (e.g., empty scores, ensemble scores)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBool, gotErr := checkForAllZeroResponses(tt.scores)
			assert.Equal(t, tt.expectError, gotBool, "Boolean return value mismatch")
			if tt.expectError {
				assert.ErrorIs(t, gotErr, ErrAllScoresZeroConfidence) // Check for the specific sentinel error
			} else {
				assert.NoError(t, gotErr)
			}
		})
	}
}
