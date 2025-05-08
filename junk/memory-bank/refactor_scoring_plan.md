# Refactor Scoring Plan

## Status (as of April 19, 2025)

- **ScoreManager** is now fully integrated and used by all relevant API handlers (notably `reanalyzeHandler`).
- **All scoring, storage, and progress operations** are now centralized in ScoreManager, ensuring atomicity, testability, and maintainability.
- **Resolved implementation challenges:**
  - Modified `MapModelToPerspective` to accept a config parameter for better testability
  - Updated DB operations to use `sqlx.ExtContext` for transaction support
  - Fixed cache method calls to use correct implementation (`Remove` vs `Delete`)
  - Properly integrated progress tracking and SSE notification flow
- **Backend and integration tests** should be run using `run_backend_tests.cmd`, `run_all_tests.cmd`, or the Playwright/Postman suites to verify correctness after refactor.
- **Frontend and SSE flows** are confirmed to work with the new progress and final score notification mechanism.
- **Documentation** (this file, `progress.md`, and `README.md`) should be updated to reflect the new handler and ScoreManager usage, as well as the new SSE-driven UI update flow.
- **Next step:** Monitor production/staging, address any regressions, and continue to refine documentation and test coverage.

## Outstanding Issues

### Code Complexity Issues

Several methods in the codebase have high cognitive complexity, making them difficult to maintain and test:

1. **API Handlers** - Multiple handlers in `internal/api/api.go` exceed the cognitive complexity threshold:
   - `reanalyzeHandler`: Complexity 105/15 - Needs refactoring into smaller, focused functions
   - `getArticlesHandler`: Complexity 63/15 - Should be broken down into data fetching, processing, and response formatting
   - `getArticleHandler`: Complexity 33/15 - Can be simplified with helper functions
   - Other handlers with complexity 17-29/15

2. **Score Calculation** - `ComputeCompositeScoreWithConfidenceFixed` in `internal/llm/composite_score_fix.go` has complexity 64/15:
   - Should be refactored into smaller functions for perspective mapping, score calculation, confidence calculation

3. **String Constants** - Duplicate string literals like "Failed to fetch article" and "Invalid article ID" should be defined as constants

### Cache Method Inconsistencies

Methods in `score_manager.go` are using incorrect cache method names:
- Using `sm.cache.Remove` which should be `sm.cache.Delete`

### Unused Variables

- `confidence` variable declared but not used in API handlers

---

## High-Level Goals

1. **✅ Fix Core Scoring Logic**: Revised the calculation to preserve the original -1.0 to +1.0 political bias scale.
2. **✅ Unify Score Management**: Created a centralized component for all score-related operations.
3. **✅ Improve Data Consistency**: Implemented atomic score updates and accurate confidence metrics.
4. **✅ Enhance Reliability**: Added proper error handling, resource cleanup, and retry mechanisms.
5. **✅ Improve Testability**: Designed for better test coverage and observability.

---

## Architectural Changes

### 1. Score Manager Component ✅

```
┌─────────────────────────────┐
│        ScoreManager         │
├─────────────────────────────┤
│ - CalculateScore()          │
│ - StoreScore()              │
│ - UpdateScore()             │
│ - InvalidateCache()         │
│ - TrackProgress()           │
└───────────┬─────────────────┘
            │
            │ uses
            ▼
┌─────────────────────────────┐
│    Transaction Manager      │
└─────────────────────────────┘
```

- ✅ Created a central `ScoreManager` to handle all score-related operations.
- ✅ Used dependency injection for database, cache, and LLM clients.
- ✅ Implemented transaction support for atomic operations using `sqlx.ExtContext`.
- ✅ Added automatic cleanup for progress tracking.

### 2. Score Calculation Revision ✅

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  ScoreCalculator │     │ ConfidenceModel  │     │   ScoreStorage   │
├──────────────────┤     ├──────────────────┤     ├──────────────────┤
│ - Calculate()    │     │ - Calculate()    │     │ - Store()        │
└───────┬──────────┘     └────────┬─────────┘     └────────┬─────────┘
        │                         │                        │
        └─────────────┬───────────┘                        │
                      │                                    │
               ┌──────▼────────┐                           │
               │ ScoreManager  │◄──────────────────────────┘
               └───────────────┘
```

- ✅ Replaced scoring algorithm with a simple average of perspective scores.
- ✅ Implemented model-based confidence calculation using metadata.
- ✅ Added support for pluggable algorithms through interfaces.
- ✅ Made config injection explicit for better testability.

---

## Implementation Details

### 1. Core Score Calculation Logic ✅

```go
// ScoreCalculator interface
type ScoreCalculator interface {
    CalculateScore(scores []db.LLMScore) (float64, float64, error)
}

// DefaultScoreCalculator implementation
type DefaultScoreCalculator struct {
    Config *CompositeScoreConfig // Must be provided, not nil
}

// CalculateScore implements the correct averaging logic
func (c *DefaultScoreCalculator) CalculateScore(scores []db.LLMScore) (float64, float64, error) {
    // Maps scores to perspectives (left/center/right)
    // Calculates simple average without transformation
    // Extracts confidence from metadata or uses proportion of models present
    // Returns score (-1.0 to +1.0) and confidence value
}
```

### 2. Score Manager Implementation ✅

```go
// ScoreManager orchestrates score operations
type ScoreManager struct {
    db          *sqlx.DB
    cache       *Cache
    calculator  ScoreCalculator
    progressMgr *ProgressManager
}

// NewScoreManager creates a new score manager
func NewScoreManager(db *sqlx.DB, cache *Cache, calculator ScoreCalculator, progressMgr *ProgressManager) *ScoreManager {
    return &ScoreManager{
        db:          db,
        cache:       cache,
        calculator:  calculator,
        progressMgr: progressMgr,
    }
}

// UpdateArticleScore handles atomic update of score and confidence
func (sm *ScoreManager) UpdateArticleScore(articleID int64, scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
    // Begins database transaction
    // Calculates score using calculator
    // Stores ensemble score record
    // Updates article's composite_score and confidence
    // Commits transaction
    // Invalidates cache
    // Returns final score and confidence
}
```

### 3. Progress Manager with Cleanup ✅

```go
// ProgressManager tracks scoring progress with cleanup
type ProgressManager struct {
    progressMap     map[int64]*ProgressState
    progressMapLock sync.RWMutex
    cleanupInterval time.Duration
}

// NewProgressManager creates a progress manager with cleanup
func NewProgressManager(cleanupInterval time.Duration) *ProgressManager {
    pm := &ProgressManager{
        progressMap:     make(map[int64]*ProgressState),
        cleanupInterval: cleanupInterval,
    }
    go pm.startCleanupRoutine()
    return pm
}

// startCleanupRoutine periodically removes stale entries
func (pm *ProgressManager) startCleanupRoutine() {
    ticker := time.NewTicker(pm.cleanupInterval)
    defer ticker.Stop()
    
    for range ticker.C {
        pm.cleanup()
    }
}
```

### 4. Handler Integration ✅

```go
// Updated reanalyzeHandler using ScoreManager
func reanalyzeHandler(llmClient *llm.LLMClient, dbConn *sqlx.DB, scoreManager *llm.ScoreManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Parses article ID and validates request
        // Handles direct score update path with ScoreManager
        // Sets initial progress state
        // Starts background scoring process
        // Returns response to client
    }
}
```

---

## Testing Strategy

### 1. Unit Tests ✅

```go
func TestDefaultScoreCalculator_CalculateScore(t *testing.T) {
    tests := []struct {
        name          string
        scores        []db.LLMScore
        expectedScore float64
        expectedConf  float64
        expectError   bool
    }{
        {
            name: "All perspectives present",
            scores: []db.LLMScore{
                {Model: "left", Score: -0.8, Metadata: `{"confidence": 0.9}`},
                {Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`}, 
                {Model: "right", Score: 0.6, Metadata: `{"confidence": 0.7}`},
            },
            expectedScore: -0.067, // (-0.8 + 0.0 + 0.6) / 3
            expectedConf:  0.8,    // (0.9 + 0.8 + 0.7) / 3
            expectError:   false,
        },
        // More test cases...
    }

    cfg := &CompositeScoreConfig{
        MinScore: -1.0,
        MaxScore: 1.0,
    }
    calculator := &DefaultScoreCalculator{Config: cfg}
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            score, conf, err := calculator.CalculateScore(tt.scores)
            // Assert results
        })
    }
}
```

### 2. Integration Tests ⚠️

```go
func TestScoreManagerWithDatabase(t *testing.T) {
    // Setup test database
    // Create ScoreManager with dependencies
    // Test various score update scenarios
    // Verify database state after operations
    // Test cache invalidation
    // Test progress tracking
}
```

### 3. End-to-End Tests ⚠️

```go
func TestReanalyzeEndpoint(t *testing.T) {
    // Setup test server with ScoreManager
    // Create test article
    // Send reanalyze request
    // Monitor SSE progress events
    // Verify final article state matches expectations
}
```

---

## Migration Plan

1. ✅ Implement new `ScoreCalculator` interface and `DefaultScoreCalculator`.
2. ✅ Create `ProgressManager` with cleanup capabilities.
3. ✅ Implement `ScoreManager` with transaction support.
4. ✅ Update API handlers to use the new components.
5. ⚠️ Add database migration to convert existing scores (optional).
6. ⚠️ Deploy with feature flag for gradual rollout.
7. ⚠️ Verify in production with monitoring.

## Lessons Learned

- **Interface-Based Design**: Using interfaces for score calculation and progress tracking enabled better testability and flexibility.
- **Transaction Safety**: Properly handling database transactions with sqlx.ExtContext made operations atomic and consistent.
- **Testable Dependencies**: Making dependencies explicit (e.g., passing config to functions) greatly improved test isolation.
- **Centralized Logic**: Moving scoring logic to a dedicated manager reduced code duplication and improved reliability.
- **Progress Tracking**: Automated cleanup of progress states prevents memory leaks in long-running services.

---

## Next Phase: Code Complexity Resolution

To address the outstanding code complexity issues identified in the static analysis, we'll take a methodical approach:

### Phase 1: Fix Immediate Errors (April 19-20, 2025)

1. **Cache Method Fixes**
   ```go
   // Replace incorrect cache.Remove calls with cache.Delete in score_manager.go
   // From:
   sm.cache.Remove(cacheKey)
   // To:
   sm.cache.Delete(cacheKey)
   ```

2. **Unused Variable Elimination**
   ```go
   // Remove or use the confidence variable in API handlers
   // Either remove the declaration:
   // score, _ := llmClient.ComputeCompositeScoreWithConfidenceFixed(scores)
   // Or use the value:
   // score, confidence := llmClient.ComputeCompositeScoreWithConfidenceFixed(scores)
   // log.Printf("Computed score with confidence: %.2f", confidence)
   ```

3. **String Constant Definitions**
   ```go
   // Add constants at the top of api.go:
   const (
       errFailedFetchArticle = "Failed to fetch article"
       errInvalidArticleID   = "Invalid article ID"
       // Add other repeated strings...
   )
   ```

### Phase 2: Refactor High Complexity Methods (April 21-25, 2025)

1. **API Handler Decomposition Strategy**

   For each complex handler like `reanalyzeHandler`:
   
   ```go
   // Original pattern:
   func complexHandler(...) gin.HandlerFunc {
       return func(c *gin.Context) {
           // 50-100+ lines of code with many branches
       }
   }
   
   // New pattern:
   func complexHandler(...) gin.HandlerFunc {
       return func(c *gin.Context) {
           // Input validation
           article, err := validateAndGetArticle(c, db)
           if err != nil {
               handleError(c, err)
               return
           }
           
           // Core business logic
           result, err := processArticle(article)
           if err != nil {
               handleError(c, err)
               return
           }
           
           // Response formatting
           formatAndSendResponse(c, result)
       }
   }
   ```

2. **Score Calculation Refactoring**

   Break down `ComputeCompositeScoreWithConfidenceFixed` into:
   
   ```go
   // Main coordination function
   func ComputeCompositeScoreWithConfidenceFixed(scores []db.LLMScore) (float64, float64, error) {
       cfg, err := LoadCompositeScoreConfig()
       if err != nil {
           return 0, 0, fmt.Errorf("loading composite score config: %w", err)
       }
       
       // Map scores to perspectives
       perspectiveModels := mapScoresToPerspectives(scores, cfg)
       
       // Find best models per perspective
       bestScores := selectBestModelsPerPerspective(perspectiveModels)
       
       // Calculate composite score
       composite := calculateCompositeScore(bestScores, cfg)
       
       // Calculate confidence
       confidence := calculateConfidence(perspectiveModels, bestScores, cfg)
       
       return composite, confidence, nil
   }
   ```

### Phase 3: Comprehensive Testing (April 26-30, 2025)

1. **Unit Testing**
   - Create specific tests for each helper function to ensure behavior is preserved
   - Compare output of refactored functions with original functions

2. **Integration Testing**
   - Verify that API endpoints behave identically before and after refactoring
   - Test error cases and edge cases specifically

3. **Monitoring Plan**
   - Add specific logging for refactored functions
   - Track performance metrics before and after changes

---