# Refactor Scoring Plan

## High-Level Goals

1. **Fix Core Scoring Logic**: Revise the calculation to preserve the original -1.0 to +1.0 political bias scale.
2. **Unify Score Management**: Create a centralized component for all score-related operations.
3. **Improve Data Consistency**: Ensure atomic score updates and accurate confidence metrics.
4. **Enhance Reliability**: Add proper error handling, resource cleanup, and retry mechanisms.
5. **Improve Testability**: Design for better test coverage and observability.

---

## Architectural Changes

### 1. Score Manager Component

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

- Create a central `ScoreManager` to handle all score-related operations.
- Use dependency injection for database, cache, and LLM clients.
- Implement transaction support for atomic operations.
- Add automatic cleanup for progress tracking.

### 2. Score Calculation Revision

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

- Replace current scoring algorithm with a simple average of perspective scores.
- Implement model-based confidence calculation using metadata.
- Support pluggable algorithms through interfaces.

---

## Implementation Details

### 1. Core Score Calculation Logic

```go
// ScoreCalculator interface
type ScoreCalculator interface {
    CalculateScore(scores []db.LLMScore) (float64, float64, error)
}

// DefaultScoreCalculator implementation
type DefaultScoreCalculator struct {
    Config *ScoreConfig
}

// CalculateScore implements the correct averaging logic
func (c *DefaultScoreCalculator) CalculateScore(scores []db.LLMScore) (float64, float64, error) {
    // Map scores to perspectives (left/center/right)
    // Calculate simple average without transformation
    // Extract confidence from metadata or use proportion of models present
    // Return score (-1.0 to +1.0) and confidence value
}
```

### 2. Score Manager Implementation

```go
// ScoreManager orchestrates score operations
type ScoreManager struct {
    db          *sqlx.DB
    cache       *Cache
    calculator  ScoreCalculator
    progressMgr *ProgressManager
}

// NewScoreManager creates a new score manager
func NewScoreManager(db *sqlx.DB, cache *Cache, calculator ScoreCalculator) *ScoreManager {
    return &ScoreManager{
        db:          db,
        cache:       cache,
        calculator:  calculator,
        progressMgr: NewProgressManager(5*time.Minute), // 5-minute cleanup interval
    }
}

// UpdateArticleScore handles atomic update of score and confidence
func (sm *ScoreManager) UpdateArticleScore(articleID int64, scores []db.LLMScore) (float64, float64, error) {
    // Begin database transaction
    // Calculate score using calculator
    // Store ensemble score record
    // Update article's composite_score and confidence
    // Commit transaction
    // Invalidate cache
    // Return final score and confidence
}
```

### 3. Progress Manager with Cleanup

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

### 4. Handler Integration

```go
// Updated reanalyzeHandler using ScoreManager
func reanalyzeHandler(scoreManager *ScoreManager, dbConn *sqlx.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Parse article ID and validate request
        // Handle direct score update path with ScoreManager
        // Set initial progress state
        // Start background scoring process
        // Return response to client
    }
}
```

---

## Testing Strategy

### 1. Unit Tests

```go
func TestScoreCalculation(t *testing.T) {
    tests := []struct {
        name            string
        scores          []db.LLMScore
        expectedScore   float64
        expectedConf    float64
        expectError     bool
    }{
        {
            name: "All perspectives present",
            scores: []db.LLMScore{
                {Model: "left", Score: -0.8, Metadata: `{"confidence": 0.9}`},
                {Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`}, 
                {Model: "right", Score: 0.6, Metadata: `{"confidence": 0.7}`},
            },
            expectedScore: -0.067, // (-0.8 + 0.0 + 0.6) / 3
            expectedConf: 0.8,     // (0.9 + 0.8 + 0.7) / 3
            expectError: false,
        },
        // More test cases...
    }

    calculator := &DefaultScoreCalculator{}
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            score, conf, err := calculator.CalculateScore(tt.scores)
            // Assert results
        })
    }
}
```

### 2. Integration Tests

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

### 3. End-to-End Tests

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

1. Implement new `ScoreCalculator` interface and `DefaultScoreCalculator`.
2. Create `ProgressManager` with cleanup capabilities.
3. Implement `ScoreManager` with transaction support.
4. Update API handlers to use the new components.
5. Add database migration to convert existing scores (optional).
6. Deploy with feature flag for gradual rollout.
7. Verify in production with monitoring.