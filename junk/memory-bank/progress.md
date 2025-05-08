# Progress & Actionable Next Steps (April 2025)

_Last updated: April 11, 2025_  
_Reference: [score_data_flow_analysis.md](../score_data_flow_analysis.md)_

---

## Overarching Objective

**Deploy the scoring system into production with robust, transparent, and maintainable data flow, error handling, and documentation.**

---

## Milestone Achieved: Production-Ready Scoring System (April 2025)

- **Scoring pipeline issues resolved:**  
  The end-to-end scoring pipeline (article selection, LLM model mapping, scoring job execution) was debugged and fixed. Logical model labels ("left", "center", "right") are now correctly mapped to actual LLM model names, and all required models are registered and available in the backend.
- **Verification:**  
  Article 788, which previously failed to receive a score, now displays a real, nonzero `CompositeScore` on the main page. Manual and automated checks confirm that scores are computed and displayed correctly for all main page articles.
- **Production-ready status:**  
  As of April 11, 2025, the scoring system is fully functional and production-ready. The pipeline has been verified in a production environment, and ongoing monitoring/documentation will continue as the system evolves.

---

## Actionable Next Steps

### 1. Clarify and Justify Handling of Missing Perspective Scores
- Review and document the logic in `llm.ComputeCompositeScore` for missing perspective scores.
- Justify or update the defaulting-to-zero approach; consider alternatives (e.g., exclude missing, flag incomplete).
- Update both code comments and [score_data_flow_analysis.md] to reflect the rationale and impact.

### 2. Disambiguate Frontend Data Retrieval Methods
- Audit frontend code to specify whether detail view uses JS fetch, htmx, or both for `/articles/{id}`.
- Update documentation to clarify the exact retrieval method(s) and ensure consistent error handling.

### 3. Complete Documentation of Score Display Logic
- Review and document how `CompositeScore` is formatted and displayed in both the main list and detail view.
- Ensure any differences are explained and justified in [score_data_flow_analysis.md].

### 4. Provide Rationale and Alternatives for Composite Score Formula **Done:**
- Add a section to documentation explaining the choice of `1.0 - abs(average)` for the composite score.

---

## Memory Bank Update Log

[2025-04-12 12:06:27] - Memory Bank Update (UMB) completed at user request. All session context and clarifications synchronized to memory-bank files.
[2025-04-12 16:09:58] - Debug session: Identified OpenRouter rate limit as root cause of scoring failure. Multiple tool attempts to patch reanalyzeHandler; provided manual patch instructions. Memory Bank updated (UMB) at user request.

[2025-04-12 16:31:00] - Added secondary API key to .env file.
[2025-04-12 16:31:00] - Added secondary API key to .env file.
[2025-04-12 18:41:45] - UMB: Updated models in config, added secondary API key. New JSON parsing error (markdown backticks) identified with tokyotech-llm model. Debug task cancelled by user; decided to ignore parsing error for now.
[2025-04-12 19:28:30] - Fixed score display issue: Modified getArticlesHandler in internal/api/api.go to use stored ensemble score instead of recalculating.
[2025-04-12 19:28:30] - Fixed score display issue: Modified getArticlesHandler in internal/api/api.go to use stored ensemble score instead of recalculating.

[2025-04-13 17:23:23] - Added and documented a comprehensive Postman-based rescoring workflow test plan (see memory-bank/postman_rescoring_test_plan.md). Plan covers test case design, environment setup, data preparation, request sequencing, response validation, edge case handling, expected outcomes, and automation for regression testing.

---

# Scoring Refactoring Plan

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
