name: "Admin Dashboard Backend Implementation PRP"
description: |

## Purpose
Implement missing backend API endpoints for the admin dashboard to make the existing frontend functional. The admin page UI exists with comprehensive controls but the JavaScript functions fail because the backend endpoints don't exist.

## Core Principles
1. **Context is King**: Include ALL necessary documentation, examples, and caveats
2. **Validation Loops**: Provide executable tests/lints the AI can run and fix
3. **Information Dense**: Use keywords and patterns from the codebase
4. **Progressive Success**: Start simple, validate, then enhance
5. **Global rules**: Be sure to follow all rules in CLAUDE.md

---

## Goal
Implement complete backend API endpoints for the admin dashboard to support all existing frontend functionality including system monitoring, feed management, analysis control, database management, and system health checks.

## Why
- **Business value**: Enable administrators to manage the NewsBalancer system effectively
- **Integration**: Connect existing comprehensive frontend UI with backend services
- **Problems solved**: Currently admin buttons don't work because endpoints are missing
- **User impact**: System administrators need functional tools to monitor and manage the application

## What
Implement missing admin API endpoints that the frontend JavaScript already calls, ensuring proper error handling, logging, and integration with existing services.

### Success Criteria
- [ ] All admin dashboard buttons work and call functional backend endpoints
- [ ] System status monitoring displays real data from backend services
- [ ] Feed management operations execute successfully
- [ ] Database management functions work properly
- [ ] Error handling provides meaningful feedback to users
- [ ] All endpoints follow existing API patterns and security practices

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- file: templates/admin.html
  why: Frontend implementation showing all required endpoints and expected responses
  
- file: internal/api/api.go
  why: Existing API patterns, error handling, and route registration approach
  
- file: cmd/server/template_handlers.go
  why: Template handler patterns and internal API client usage
  
- file: cmd/server/main.go
  why: Service initialization and dependency injection patterns

- file: internal/rss/collector.go
  why: RSS collector interface for feed management operations
  
- file: internal/llm/client.go
  why: LLM client interface for analysis operations
  
- file: internal/db/operations.go
  why: Database operations interface for management functions

- url: https://gin-gonic.com/docs/examples/
  why: Gin framework patterns for REST API implementation
  section: Middleware and error handling
  critical: Proper HTTP status codes and JSON responses

- url: https://github.com/gin-gonic/gin#api-examples
  why: Standard patterns for route handlers and middleware
  section: Route grouping and handler patterns
  critical: Consistent response format and error handling
```

### Current Codebase tree (relevant sections)
```bash
internal/
├── api/
│   ├── api.go              # Main API route registration
│   ├── handlers.go         # Existing handler patterns
│   └── responses.go        # Standard response formats
├── rss/
│   └── collector.go        # RSS feed management interface
├── llm/
│   ├── client.go          # LLM service interface
│   └── score_manager.go   # Analysis management
└── db/
    └── operations.go       # Database operations interface

cmd/server/
├── main.go                # Service initialization
└── template_handlers.go   # Template handler patterns

templates/
└── admin.html            # Frontend requiring backend endpoints
```

### Desired Codebase tree with files to be added
```bash
internal/api/
├── admin_handlers.go      # New admin-specific API handlers
└── admin_routes.go        # Admin route registration (optional)

# Modified files:
internal/api/api.go        # Add admin routes to RegisterRoutes function
```

### Known Gotchas of our codebase & Library Quirks
```go
// CRITICAL: All API handlers must use SafeHandler wrapper
router.POST("/api/admin/endpoint", SafeHandler(adminHandler(dependencies)))

// CRITICAL: Use RespondSuccess and RespondError for consistent responses
RespondSuccess(c, map[string]interface{}{"status": "completed"})
RespondError(c, http.StatusInternalServerError, "Operation failed", err)

// CRITICAL: Context timeouts for long operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// CRITICAL: Gin framework requires specific parameter binding
// Use c.ShouldBindJSON() for POST bodies, c.Param() for path params

// CRITICAL: Database operations use sqlx.DB interface
// Follow existing patterns in internal/db/operations.go

// CRITICAL: RSS collector operations are async
// Use go routines for feed refresh operations
```

## Implementation Blueprint

### Data models and structure
```go
// Admin operation response models
type AdminOperationResponse struct {
    Status    string                 `json:"status"`
    Message   string                 `json:"message"`
    Data      map[string]interface{} `json:"data,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
}

type SystemHealthResponse struct {
    DatabaseOK   bool `json:"database_ok"`
    LLMServiceOK bool `json:"llm_service_ok"`
    RSSServiceOK bool `json:"rss_service_ok"`
    ServerOK     bool `json:"server_ok"`
}

type SystemStatsResponse struct {
    TotalArticles    int    `json:"total_articles"`
    ArticlesToday    int    `json:"articles_today"`
    PendingAnalysis  int    `json:"pending_analysis"`
    ActiveSources    int    `json:"active_sources"`
    DatabaseSize     string `json:"database_size"`
    LeftCount        int    `json:"left_count"`
    CenterCount      int    `json:"center_count"`
    RightCount       int    `json:"right_count"`
    LeftPercentage   float64 `json:"left_percentage"`
    CenterPercentage float64 `json:"center_percentage"`
    RightPercentage  float64 `json:"right_percentage"`
}
```

### list of tasks to be completed to fulfill the PRP in the order they should be completed

```yaml
Task 1: Create admin handlers file
CREATE internal/api/admin_handlers.go:
  - MIRROR pattern from: internal/api/api.go (existing handlers)
  - IMPLEMENT admin-specific handlers with proper error handling
  - USE existing SafeHandler wrapper pattern
  - FOLLOW RespondSuccess/RespondError response patterns

Task 2: Implement feed management endpoints
MODIFY internal/api/admin_handlers.go:
  - ADD refreshFeedsHandler for POST /api/admin/refresh-feeds
  - ADD resetFeedErrorsHandler for POST /api/admin/reset-feed-errors  
  - ADD getSourcesStatusHandler for GET /api/admin/sources
  - INTEGRATE with existing rss.CollectorInterface

Task 3: Implement analysis control endpoints
MODIFY internal/api/admin_handlers.go:
  - ADD reanalyzeRecentHandler for POST /api/admin/reanalyze-recent
  - ADD clearAnalysisErrorsHandler for POST /api/admin/clear-analysis-errors
  - ADD validateBiasScoresHandler for POST /api/admin/validate-bias-scores
  - INTEGRATE with existing llm.LLMClient and llm.ScoreManager

Task 4: Implement database management endpoints
MODIFY internal/api/admin_handlers.go:
  - ADD optimizeDatabaseHandler for POST /api/admin/optimize-db
  - ADD exportDataHandler for POST /api/admin/export-data
  - ADD cleanupOldArticlesHandler for POST /api/admin/cleanup-old-articles
  - INTEGRATE with existing database operations

Task 5: Implement monitoring endpoints
MODIFY internal/api/admin_handlers.go:
  - ADD getMetricsHandler for GET /api/admin/metrics
  - ADD getLogsHandler for GET /api/admin/logs
  - ADD runHealthCheckHandler for POST /api/admin/health-check
  - RETURN structured data for frontend consumption

Task 6: Register admin routes
MODIFY internal/api/api.go:
  - FIND RegisterRoutes function
  - ADD admin route registrations after existing routes
  - PRESERVE existing route patterns and middleware
  - USE consistent route grouping approach

Task 7: Update template handler data fetching
MODIFY cmd/server/template_handlers.go:
  - ENHANCE getDetailedStats to use new admin endpoints
  - IMPROVE getSystemStatus to use health check endpoint
  - ENSURE proper error handling and fallbacks
```

### Per task pseudocode as needed added to each task

```go
// Task 1: Admin handlers file structure
// PATTERN: Follow existing handler patterns in internal/api/api.go
func adminRefreshFeedsHandler(rssCollector rss.CollectorInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        // PATTERN: Always use context with timeout
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        // GOTCHA: RSS operations are async, start in goroutine
        go rssCollector.ManualRefresh()

        // PATTERN: Use standard response format
        RespondSuccess(c, map[string]interface{}{
            "status": "refresh_initiated",
            "message": "Feed refresh started successfully",
        })
    }
}

// Task 3: Analysis control pattern
func reanalyzeRecentHandler(llmClient *llm.LLMClient, dbConn *sqlx.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // PATTERN: Validate request and check LLM availability
        if !llmClient.IsAvailable() {
            RespondError(c, http.StatusServiceUnavailable, "LLM service unavailable", nil)
            return
        }

        // CRITICAL: Long operation requires async processing
        go func() {
            // Implement reanalysis logic
        }()

        RespondSuccess(c, map[string]interface{}{"status": "reanalysis_started"})
    }
}

// Task 4: Database management pattern
func optimizeDatabaseHandler(dbConn *sqlx.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // PATTERN: Database operations need proper error handling
        _, err := dbConn.Exec("VACUUM; ANALYZE;")
        if err != nil {
            RespondError(c, http.StatusInternalServerError, "Database optimization failed", err)
            return
        }

        RespondSuccess(c, map[string]interface{}{"status": "optimization_completed"})
    }
}
```

### Integration Points
```yaml
DATABASE:
  - operations: "Use existing sqlx.DB connection from main.go"
  - patterns: "Follow internal/db/operations.go interface patterns"

RSS_COLLECTOR:
  - interface: "Use rss.CollectorInterface from dependency injection"
  - methods: "ManualRefresh(), GetFeedHealth(), ResetErrors()"

LLM_CLIENT:
  - interface: "Use llm.LLMClient from dependency injection"
  - methods: "IsAvailable(), ReanalyzeArticle(), ValidateScores()"

ROUTES:
  - add to: internal/api/api.go RegisterRoutes function
  - pattern: "router.POST('/api/admin/endpoint', SafeHandler(handler(deps)))"
  - grouping: "Add admin routes after existing API routes"
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Run these FIRST - fix any errors before proceeding
go fmt ./internal/api/admin_handlers.go
go vet ./internal/api/admin_handlers.go
golangci-lint run ./internal/api/admin_handlers.go

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: Unit Tests each new feature/file/function use existing test patterns
```go
// CREATE internal/api/admin_handlers_test.go with these test cases:
func TestAdminRefreshFeedsHandler(t *testing.T) {
    // Test successful feed refresh initiation
    mockCollector := &MockRSSCollector{}
    handler := adminRefreshFeedsHandler(mockCollector)

    req := httptest.NewRequest("POST", "/api/admin/refresh-feeds", nil)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = req

    handler(c)

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "refresh_initiated")
}

func TestReanalyzeRecentHandler_LLMUnavailable(t *testing.T) {
    // Test LLM service unavailable scenario
    mockLLM := &MockLLMClient{available: false}
    handler := reanalyzeRecentHandler(mockLLM, nil)

    req := httptest.NewRequest("POST", "/api/admin/reanalyze-recent", nil)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = req

    handler(c)

    assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
```

```bash
# Run and iterate until passing:
go test ./internal/api/admin_handlers_test.go -v
# If failing: Read error, understand root cause, fix code, re-run
```

### Level 3: Integration Test
```bash
# Start the service
go run cmd/server/main.go

# Test admin endpoints
curl -X POST http://localhost:8080/api/admin/refresh-feeds \
  -H "Content-Type: application/json"

curl -X GET http://localhost:8080/api/admin/sources

curl -X POST http://localhost:8080/api/admin/health-check

# Expected: {"status": "success", "data": {...}} for each
# If error: Check logs for stack trace and fix issues
```

### Level 4: Frontend Integration Test
```bash
# Open admin page and test buttons
# Navigate to http://localhost:8080/admin
# Click each button and verify:
# - No JavaScript errors in console
# - Proper success/error messages displayed
# - Page reloads show updated data
```

## Final validation Checklist
- [ ] All tests pass: `go test ./internal/api/... -v`
- [ ] No linting errors: `golangci-lint run ./internal/api/`
- [ ] No formatting issues: `go fmt ./internal/api/`
- [ ] Manual test successful: All admin buttons work in browser
- [ ] Error cases handled gracefully with proper HTTP status codes
- [ ] Logs are informative but not verbose
- [ ] Frontend JavaScript functions complete without errors

---

## Anti-Patterns to Avoid
- ❌ Don't create new response formats when RespondSuccess/RespondError exist
- ❌ Don't skip SafeHandler wrapper for route handlers
- ❌ Don't use sync operations for long-running tasks (feeds, analysis)
- ❌ Don't ignore context timeouts for database operations
- ❌ Don't hardcode values that should come from configuration
- ❌ Don't catch all errors generically - be specific about error types
- ❌ Don't modify existing API patterns - follow established conventions

## Confidence Score: 9/10
This PRP provides comprehensive context for one-pass implementation success. The existing codebase patterns are well-documented, all required endpoints are identified from the frontend, and validation steps ensure proper integration with existing services.
