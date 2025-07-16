name: "Test Coverage Improvement Plan - Admin & Source Handlers"
description: |

## Purpose
Increase test coverage for internal/api/admin_handlers.go and internal/api/source_handlers.go from current ~20.8% to ≥80% to meet SonarCloud quality gate requirements.

## Core Principles
1. **Comprehensive Coverage**: Test all critical paths, error conditions, and edge cases
2. **Real Database Testing**: Use actual SQLite connections with test data
3. **Mock External Dependencies**: Mock RSS collectors, LLM clients appropriately
4. **Validation Loops**: Ensure tests are reliable and maintainable
5. **Follow Existing Patterns**: Use established test patterns from the codebase

---

## Goal
Achieve ≥80% test coverage for:
- `internal/api/admin_handlers.go` (895 lines, currently ~20% covered)
- `internal/api/source_handlers.go` (470 lines, currently ~20% covered)

## Why
- **SonarCloud Quality Gate**: Requires ≥80% coverage on new lines for PR approval
- **Code Reliability**: Ensure admin operations and source management work correctly
- **Regression Prevention**: Catch breaking changes in critical admin functionality
- **Maintainability**: Well-tested code is easier to refactor and extend

## What
Comprehensive test suites covering:
- All HTTP handlers with various input scenarios
- Database operations with real SQLite connections
- Error handling and edge cases
- Mock integrations with external services (RSS, LLM)
- HTMX template rendering for admin interfaces

### Success Criteria
- [ ] admin_handlers.go achieves ≥80% line coverage
- [ ] source_handlers.go achieves ≥80% line coverage
- [ ] All tests pass consistently in CI/CD pipeline
- [ ] Tests use real database connections (not mocks)
- [ ] Error paths and edge cases are thoroughly tested

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- file: internal/api/admin_handlers_basic_test.go
  why: Existing test patterns and mock structures to follow
  
- file: internal/api/source_handlers_test.go
  why: Current test approach and gaps to address
  
- file: internal/api/admin_handlers.go
  why: All functions that need comprehensive test coverage
  
- file: internal/api/source_handlers.go
  why: All CRUD operations and validation logic to test

- file: internal/db/db.go
  why: Database operations and connection patterns for tests

- file: internal/models/models.go
  why: Data structures and validation methods used in handlers
```

### Current Test Coverage Analysis
Based on coverage.out analysis:
- **admin_handlers.go**: Many functions have 0% coverage (lines 1188-1295)
- **source_handlers.go**: Basic handler structure tested but missing comprehensive scenarios
- **Missing Coverage Areas**:
  - Database error scenarios
  - Input validation edge cases
  - Async operation handling
  - HTMX template rendering
  - Complex admin operations (reanalysis, cleanup, export)

### Known Gotchas of our codebase & Library Quirks
```go
// CRITICAL: Use modernc.org/sqlite driver consistently (not mattn/go-sqlite3)
// CRITICAL: Tests need real database connections for proper coverage
// CRITICAL: Admin operations use goroutines - need proper synchronization in tests
// CRITICAL: HTMX handlers return HTML templates - need template loading in tests
// CRITICAL: Mock RSS collectors and LLM clients but test database operations with real DB
// CRITICAL: Use testify/assert and testify/mock for consistent test patterns
```

## Implementation Blueprint

### Data models and structure
```go
// Test database setup with proper isolation
type TestDB struct {
    *sqlx.DB
    cleanup func()
}

// Mock structures for external dependencies
type MockRSSCollector struct {
    mock.Mock
}

type MockLLMClient struct {
    mock.Mock
}

type MockScoreManager struct {
    mock.Mock
}
```

### List of tasks to be completed to fulfill the PRP

```yaml
Task 1: Enhance admin_handlers_basic_test.go
  - ADD comprehensive database setup/teardown
  - ADD tests for all uncovered admin functions
  - ADD error scenario testing
  - ADD async operation validation

Task 2: Expand source_handlers_test.go  
  - ADD full CRUD operation testing with real database
  - ADD input validation edge cases
  - ADD error handling scenarios
  - ADD pagination and filtering tests

Task 3: Create admin_handlers_database_test.go
  - ADD tests for database-heavy operations
  - ADD transaction testing
  - ADD cleanup and optimization operations
  - ADD export functionality testing

Task 4: Create source_handlers_integration_test.go
  - ADD end-to-end source management workflows
  - ADD HTMX template rendering tests
  - ADD complex validation scenarios
  - ADD concurrent operation testing

Task 5: Add test utilities and helpers
  - CREATE test database factory
  - CREATE common mock setups
  - CREATE test data generators
  - CREATE assertion helpers
```

### Per task pseudocode

```go
// Task 1: Enhanced admin handler tests
func TestAdminReanalyzeRecentHandler(t *testing.T) {
    // PATTERN: Setup test database with articles
    db := setupTestDB(t)
    defer db.cleanup()
    
    // PATTERN: Insert test articles for reanalysis
    insertTestArticles(db, 10, time.Now().Add(-3*24*time.Hour))
    
    // PATTERN: Mock LLM client with expectations
    mockLLM := &MockLLMClient{}
    mockLLM.On("ValidateAPIKey").Return(nil)
    mockLLM.On("ReanalyzeArticle", mock.Anything, mock.Anything, mock.Anything).Return(nil)
    
    // CRITICAL: Test async operation completion
    handler := adminReanalyzeRecentHandler(mockLLM, mockScoreManager, db.DB)
    
    // Test success case, error cases, validation failures
}

// Task 2: Source handler comprehensive tests  
func TestCreateSourceHandler_ValidationScenarios(t *testing.T) {
    tests := []struct {
        name           string
        request        models.CreateSourceRequest
        expectedStatus int
        expectedError  string
    }{
        // PATTERN: Test all validation rules
        {"valid_request", validRequest, 201, ""},
        {"empty_name", emptyNameRequest, 400, "name is required"},
        {"invalid_url", invalidURLRequest, 400, "invalid feed URL"},
        // ... comprehensive validation scenarios
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // PATTERN: Fresh database for each test
            db := setupTestDB(t)
            defer db.cleanup()
            
            // Test with real database operations
        })
    }
}
```

### Integration Points
```yaml
DATABASE:
  - setup: "Use modernc.org/sqlite with in-memory databases for tests"
  - pattern: "CREATE TABLE IF NOT EXISTS for test schema setup"
  - cleanup: "DROP TABLE and close connections after each test"

TEMPLATES:
  - setup: "Load HTML templates for HTMX handler testing"
  - pattern: "gin.LoadHTMLGlob for template rendering tests"

MOCKS:
  - rss: "Mock RSS collector for feed operations"
  - llm: "Mock LLM client for analysis operations"
  - external: "Mock external API calls and timeouts"
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Run these FIRST - fix any errors before proceeding
go fmt ./internal/api/...                    # Format code
go vet ./internal/api/...                    # Static analysis
golangci-lint run ./internal/api/...         # Comprehensive linting

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: Unit Tests - Incremental Coverage Improvement
```bash
# Test individual functions with coverage tracking
go test ./internal/api -coverprofile=coverage.out -v

# Check coverage for specific files
go tool cover -func=coverage.out | grep -E "(admin_handlers|source_handlers)"

# Target: Each iteration should increase coverage by 10-15%
# Run after each batch of new tests to track progress
```

### Level 3: Integration Testing
```bash
# Run full test suite with race detection
go test ./internal/api -race -v

# Test with real database operations
go test ./internal/api -tags=integration -v

# Expected: All tests pass, no race conditions detected
```

## Detailed Implementation Tasks

### Task 1: Enhanced admin_handlers_basic_test.go (Target: +30% coverage)

**Functions to add comprehensive tests for:**
- `adminReanalyzeRecentHandler` - Test async reanalysis workflow
- `adminClearAnalysisErrorsHandler` - Test database error clearing
- `adminValidateBiasScoresHandler` - Test score validation logic
- `adminOptimizeDatabaseHandler` - Test database optimization
- `adminExportDataHandler` - Test CSV export functionality
- `adminCleanupOldArticlesHandler` - Test transaction-based cleanup
- `adminGetMetricsHandler` - Test metrics aggregation
- `adminRunHealthCheckHandler` - Test system health checks

**Test scenarios to cover:**
```go
// Database connection failures
// Invalid input parameters
// Timeout scenarios
// Concurrent operation handling
// Transaction rollback scenarios
// Large dataset handling
```

### Task 2: Expanded source_handlers_test.go (Target: +25% coverage)

**Functions needing comprehensive coverage:**
- `getSourcesHandler` - All query parameter combinations
- `createSourceHandler` - Validation and conflict scenarios
- `getSourceByIDHandler` - Error cases and edge conditions
- `updateSourceHandler` - Partial updates and validation
- `deleteSourceHandler` - Soft delete operations
- `getSourceStatsHandler` - Statistics calculation

**Critical test scenarios:**
```go
// Pagination edge cases (offset > total, limit = 0)
// Invalid query parameters
// Database constraint violations
// Concurrent source modifications
// Large result set handling
// Template rendering for HTMX endpoints
```

### Task 3: New admin_handlers_database_test.go (Target: +20% coverage)

**Focus on database-intensive operations:**
- Transaction handling in cleanup operations
- Bulk data operations (export, analysis)
- Database optimization and maintenance
- Error recovery and rollback scenarios
- Performance with large datasets

### Task 4: New source_handlers_integration_test.go (Target: +15% coverage)

**End-to-end workflow testing:**
- Complete source lifecycle (create → update → delete)
- HTMX form submission and response rendering
- Source validation with external feed checking
- Statistics computation and caching
- Concurrent source management operations

### Task 5: Test utilities and helpers

**Create reusable test infrastructure:**
```go
// testutil/database.go
func SetupTestDB(t *testing.T) *TestDB
func InsertTestSources(db *sqlx.DB, count int) []models.Source
func InsertTestArticles(db *sqlx.DB, sourceID int64, count int) []models.Article

// testutil/mocks.go
func NewMockRSSCollector() *MockRSSCollector
func NewMockLLMClient() *MockLLMClient
func SetupMockExpectations(mocks ...interface{})

// testutil/assertions.go
func AssertHTMLResponse(t *testing.T, w *httptest.ResponseRecorder, expectedElements []string)
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expected interface{})
```

## Detailed Test Data Generation Strategy

### Test Data Factories
```go
// testutil/factories.go - Comprehensive test data generation
type SourceFactory struct {
    db *sqlx.DB
    counter int64
}

func (f *SourceFactory) CreateSource(overrides ...func(*models.Source)) *models.Source {
    f.counter++
    source := &models.Source{
        Name:          fmt.Sprintf("Test Source %d", f.counter),
        ChannelType:   "rss",
        FeedURL:       fmt.Sprintf("https://example%d.com/feed.xml", f.counter),
        Category:      []string{"left", "center", "right"}[f.counter%3],
        Enabled:       true,
        DefaultWeight: 1.0,
        CreatedAt:     time.Now().Add(-time.Duration(f.counter) * time.Hour),
        UpdatedAt:     time.Now(),
    }

    // Apply overrides for specific test scenarios
    for _, override := range overrides {
        override(source)
    }

    return source
}

type ArticleFactory struct {
    db *sqlx.DB
    counter int64
}

func (f *ArticleFactory) CreateArticle(sourceID int64, overrides ...func(*models.Article)) *models.Article {
    f.counter++
    article := &models.Article{
        Title:          fmt.Sprintf("Test Article %d", f.counter),
        URL:            fmt.Sprintf("https://example.com/article-%d", f.counter),
        SourceID:       sourceID,
        PubDate:        time.Now().Add(-time.Duration(f.counter) * time.Hour),
        CompositeScore: generateRandomScore(),
        Confidence:     0.8 + (rand.Float64() * 0.2), // 0.8-1.0
        Status:         "analyzed",
        CreatedAt:      time.Now().Add(-time.Duration(f.counter) * time.Hour),
    }

    for _, override := range overrides {
        override(article)
    }

    return article
}

// Scenario-specific generators
func GenerateErrorArticles(count int) []models.Article {
    // Articles with error status for testing error clearing
}

func GenerateOldArticles(daysOld int, count int) []models.Article {
    // Articles older than specified days for cleanup testing
}

func GenerateBiasDistributionArticles() []models.Article {
    // Articles with specific bias scores for metrics testing
    return []models.Article{
        {CompositeScore: -0.8}, // Left
        {CompositeScore: -0.1}, // Center-left
        {CompositeScore: 0.0},  // Center
        {CompositeScore: 0.1},  // Center-right
        {CompositeScore: 0.8},  // Right
    }
}
```

### Database State Management
```go
// testutil/database_states.go - Predefined database states for testing
func SetupEmptyDatabase(t *testing.T) *TestDB {
    // Clean database for testing creation operations
}

func SetupDatabaseWithSources(t *testing.T, sourceCount int) (*TestDB, []models.Source) {
    // Database with N sources for testing listing/filtering
}

func SetupDatabaseWithArticles(t *testing.T, articleCount int) (*TestDB, []models.Article) {
    // Database with articles for analysis operations
}

func SetupDatabaseForCleanup(t *testing.T) (*TestDB, CleanupTestData) {
    // Database with old articles, LLM scores for cleanup testing
    return db, CleanupTestData{
        OldArticles:    generateOldArticles(30, 50),  // 50 articles >30 days old
        RecentArticles: generateRecentArticles(10),   // 10 recent articles
        OrphanedScores: generateOrphanedScores(25),   // 25 orphaned LLM scores
    }
}

func SetupDatabaseForMetrics(t *testing.T) (*TestDB, MetricsTestData) {
    // Database with known distribution for metrics testing
    return db, MetricsTestData{
        LeftArticles:   25,  // Expected left count
        CenterArticles: 50,  // Expected center count
        RightArticles:  25,  // Expected right count
        TotalSize:      "2.5 MB", // Expected DB size
    }
}
```

## Risk Assessment & Mitigation Strategies

### High-Risk Areas & Mitigation Plans

**Risk 1: Async Operation Testing Complexity**
- **Issue**: Admin operations use goroutines that are hard to test deterministically
- **Mitigation**:
  ```go
  // Use channels for synchronization in tests
  func TestAsyncOperation(t *testing.T) {
      done := make(chan bool, 1)
      mockLLM.On("ReanalyzeArticle").Run(func(args mock.Arguments) {
          done <- true
      }).Return(nil)

      // Test with timeout to prevent hanging
      select {
      case <-done:
          // Success
      case <-time.After(5 * time.Second):
          t.Fatal("Async operation timeout")
      }
  }
  ```

**Risk 2: Database Connection Exhaustion**
- **Issue**: Many concurrent tests could exhaust SQLite connections
- **Mitigation**:
  ```go
  // Connection pool management in tests
  func SetupTestDB(t *testing.T) *TestDB {
      db, err := sqlx.Open("sqlite", ":memory:")
      db.SetMaxOpenConns(1) // Force serialization for tests
      db.SetMaxIdleConns(1)
      // ... setup and return with cleanup
  }
  ```

**Risk 3: Template Loading Failures**
- **Issue**: HTMX tests require template files that may not exist in test environment
- **Mitigation**:
  ```go
  // Fallback template loading
  func setupTestRouter(t *testing.T) *gin.Engine {
      router := gin.New()

      // Try to load templates, fallback to embedded templates
      if err := router.LoadHTMLGlob("../../templates/**/*"); err != nil {
          // Use embedded test templates
          router.SetHTMLTemplate(getEmbeddedTestTemplates())
      }
      return router
  }
  ```

**Risk 4: Test Data Pollution Between Tests**
- **Issue**: Tests may interfere with each other due to shared state
- **Mitigation**:
  ```go
  // Strict test isolation
  func TestWithIsolation(t *testing.T) {
      t.Parallel() // Run in parallel when safe

      db := SetupTestDB(t)
      defer func() {
          db.cleanup() // Always cleanup
          if t.Failed() {
              t.Logf("Test failed, DB state: %+v", db.dumpState())
          }
      }()
  }
  ```

**Risk 5: Flaky Tests Due to Timing Issues**
- **Issue**: Race conditions in concurrent operations
- **Mitigation**:
  ```go
  // Deterministic timing with retries
  func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration) {
      ticker := time.NewTicker(10 * time.Millisecond)
      defer ticker.Stop()

      timeoutCh := time.After(timeout)
      for {
          select {
          case <-ticker.C:
              if condition() {
                  return
              }
          case <-timeoutCh:
              t.Fatal("Condition not met within timeout")
          }
      }
  }
  ```

## Coverage Target Fallback Strategies

### Progressive Coverage Milestones
```yaml
Primary Target (80%):
  - admin_handlers.go: 80% line coverage
  - source_handlers.go: 80% line coverage
  - timeline: 2 weeks

Fallback Level 1 (70%):
  - Focus on critical paths only
  - Skip complex edge cases temporarily
  - timeline: +3 days
  - triggers: "If primary target not met by day 10"

Fallback Level 2 (60%):
  - Cover only happy paths and major error cases
  - Defer HTMX template testing
  - timeline: +2 days
  - triggers: "If Level 1 not met by day 13"

Minimum Viable (50%):
  - Core CRUD operations only
  - Basic error handling
  - timeline: +1 day
  - triggers: "Emergency fallback for SonarCloud compliance"
```

### Adaptive Testing Strategy
```go
// Coverage-driven test prioritization
type TestPriority struct {
    Function     string
    CurrentCov   float64
    TargetCov    float64
    Complexity   int // 1-5 scale
    BusinessCrit int // 1-5 scale
}

func PrioritizeTests(coverageReport CoverageReport) []TestPriority {
    // Algorithm to prioritize tests based on:
    // 1. Current coverage gap
    // 2. Business criticality
    // 3. Implementation complexity
    // 4. Time remaining
}
```

### Incremental Delivery Plan
```yaml
Week 1 - Foundation (Target: 40% coverage):
  - Basic CRUD operations
  - Happy path scenarios
  - Essential error handling
  - Deliverable: Core functionality tested

Week 2 - Enhancement (Target: 65% coverage):
  - Edge cases and validation
  - Database error scenarios
  - Async operation testing
  - Deliverable: Robust error handling

Week 2.5 - Polish (Target: 80% coverage):
  - HTMX template rendering
  - Complex admin operations
  - Performance edge cases
  - Deliverable: Comprehensive coverage
```

### Emergency Protocols
```yaml
If SonarCloud blocks PR (Day 14):
  - IMMEDIATE: Focus only on new/changed lines
  - TACTIC: Add minimal tests for SonarCloud compliance
  - TIMELINE: 4 hours maximum
  - FOLLOW-UP: Schedule technical debt ticket for full coverage

If Tests Are Flaky:
  - IMMEDIATE: Disable parallel execution
  - TACTIC: Add retry logic and better synchronization
  - TIMELINE: 2 hours per flaky test
  - ESCALATION: Skip problematic tests temporarily with TODO comments

If Performance Issues:
  - IMMEDIATE: Profile test execution time
  - TACTIC: Optimize database setup/teardown
  - TIMELINE: Half day for optimization
  - FALLBACK: Split tests into separate files for parallel execution
```

## Progress Tracking

### Current Status (2025-07-16)
- **Overall API Package Coverage**: 26.2% (improved from 20.8% baseline - +5.4 percentage points!)
- **Task 1**: ✅ COMPLETE - Enhanced admin_handlers_basic_test.go
  - **Task 1.1**: ✅ COMPLETE - Enhanced test database infrastructure with TestDB struct, real SQLite connections, and proper cleanup
  - **Task 1.2**: ✅ COMPLETE - Added comprehensive tests for adminReanalyzeRecentHandler with 3 test scenarios (success, LLM unavailable, no recent articles)
  - **Task 1.3**: ✅ COMPLETE - Added tests for database admin operations (adminClearAnalysisErrorsHandler, adminOptimizeDatabaseHandler, adminCleanupOldArticlesHandler)
  - **Task 1.4**: ✅ COMPLETE - Added tests for metrics and health check handlers (adminGetMetricsHandler, adminRunHealthCheckHandler)
  - **Task 1.5**: ✅ COMPLETE - Added tests for export functionality (adminExportDataHandler with CSV validation)
- **Task 2**: ✅ COMPLETE - Expanded source_handlers_test.go
  - **Task 2.1**: ✅ COMPLETE - Enhanced getSourcesHandler tests with 11 comprehensive scenarios (pagination, filtering, validation, edge cases)
  - **Task 2.2**: ✅ COMPLETE - Enhanced createSourceHandler tests with 8 scenarios (success, validation, conflicts, edge cases)
  - **Task 2.3**: ✅ COMPLETE - Enhanced CRUD operation tests (getSourceByIDHandler, updateSourceHandler, deleteSourceHandler) with 15 scenarios
  - **Task 2.4**: ✅ COMPLETE - Added getSourceStatsHandler tests with 5 scenarios (success, not found, invalid ID, empty source, disabled source)

### Next Steps
- Continue with database admin operations testing (adminClearAnalysisErrorsHandler, adminOptimizeDatabaseHandler, adminCleanupOldArticlesHandler)
- Target: Achieve significant coverage increase with each completed task

## Final Validation Checklist
- [ ] admin_handlers.go coverage ≥80%: `go tool cover -func=coverage.out | grep admin_handlers`
- [ ] source_handlers.go coverage ≥80%: `go tool cover -func=coverage.out | grep source_handlers`
- [ ] All tests pass: `go test ./internal/api -v`
- [ ] No race conditions: `go test ./internal/api -race`
- [ ] Linting clean: `golangci-lint run ./internal/api`
- [ ] Integration tests pass: `go test ./internal/api -tags=integration`
- [ ] SonarCloud quality gate passes in CI/CD
- [ ] Test execution time reasonable (<30s for full suite)
- [ ] Risk mitigation strategies implemented for identified high-risk areas
- [ ] Fallback plans documented and ready if coverage targets not met
- [ ] Test data generation produces consistent, isolated test scenarios

---

## Anti-Patterns to Avoid
- ❌ Don't mock database operations - use real SQLite connections
- ❌ Don't skip error path testing - these are critical for admin operations
- ❌ Don't use hardcoded test data - generate dynamic test scenarios
- ❌ Don't ignore goroutine synchronization in async operation tests
- ❌ Don't test only happy paths - admin operations must handle failures gracefully
- ❌ Don't create flaky tests - ensure deterministic test execution
- ❌ Don't skip template rendering tests for HTMX endpoints
- ❌ Don't proceed without fallback plans if coverage targets aren't achievable
- ❌ Don't ignore test performance - slow tests reduce development velocity
- ❌ Don't create tests without proper cleanup - database pollution causes failures
