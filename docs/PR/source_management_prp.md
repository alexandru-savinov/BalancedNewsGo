name: "Source Management PRP - Database-Backed CRUD with Admin UI"
description: |
  Migrate from JSON-based source configuration to database-backed source management 
  with admin UI and API endpoints, preparing for multi-channel support and analytics.

## Purpose
Enable dynamic source management through admin interface and API, moving away from 
static JSON configuration to support future extensibility and operational efficiency.

## Core Principles
1. **Context is King**: Leverage existing Go/Gin/SQLite patterns and admin infrastructure
2. **Validation Loops**: Comprehensive testing at database, API, and UI levels
3. **Information Dense**: Follow established codebase patterns and conventions
4. **Progressive Success**: Migrate existing sources, then enhance with new features
5. **Global rules**: Follow all rules in CLAUDE.md and existing code patterns

---

## Goal
Transform source management from static JSON configuration to dynamic database-backed 
system with full CRUD operations via admin UI and REST API, while maintaining 
backward compatibility and preparing for multi-channel support (RSS, Telegram, etc.).

## Why
- **Operational Efficiency**: Non-developers can manage sources without code deployments
- **Future Extensibility**: Prepare for multi-channel support (Telegram, Twitter, etc.)
- **Analytics Foundation**: Enable source-level metrics (avg scores, popular tags)
- **Better Monitoring**: Track source health, error rates, and performance
- **Scalability**: Support dynamic source addition/removal without restarts

## What
Database-backed source management system with:
- New `sources` table with extensible schema
- REST API endpoints for full CRUD operations  
- Enhanced admin UI with source management interface
- Migration from existing JSON configuration
- Source analytics and health monitoring
- Preparation for future multi-channel architecture

### Success Criteria
- [ ] Sources stored in database with full metadata
- [ ] Admin UI allows CRUD operations on sources
- [ ] REST API provides programmatic source management
- [ ] Existing RSS functionality unchanged
- [ ] Source analytics (avg scores, article counts) displayed
- [ ] Health monitoring for individual sources
- [ ] Seamless migration from JSON configuration

## All Needed Context

### Documentation & References
```yaml
# MUST READ - Include these in your context window
- file: internal/db/db.go
  why: Current database schema patterns, Article model, transaction handling
  critical: SQLite retry logic, schema validation patterns

- file: internal/api/api.go  
  why: API registration patterns, SafeHandler usage, Swagger annotations
  critical: StandardResponse format, error handling patterns

- file: internal/api/admin_handlers.go
  why: Existing admin handler patterns, metrics collection
  critical: SystemStatsResponse structure, health check patterns

- file: cmd/server/main.go
  why: Service initialization, RSS collector setup, feed config loading
  critical: loadFeedSourcesConfig() function, service dependency injection

- file: internal/rss/rss.go
  why: Current RSS collection patterns, feed URL handling
  critical: Collector struct, FeedURLs field, health checking

- file: templates/admin.html
  why: Existing admin UI patterns, HTMX integration, styling
  critical: Control sections, button groups, JavaScript patterns

- file: configs/feed_sources.json
  why: Current source configuration structure
  critical: Migration data format and category mapping
```

### Current Codebase Tree
```bash
cmd/server/main.go          # Main entry point, service initialization
internal/
├── api/
│   ├── api.go             # Route registration, handler patterns
│   ├── admin_handlers.go  # Admin functionality, metrics
│   ├── models.go          # API DTOs, StandardResponse
│   └── response.go        # Response helpers, error handling
├── db/
│   └── db.go              # Database schema, Article model, CRUD patterns
├── rss/
│   └── rss.go             # RSS collection, feed health checking
└── models/                # Shared data models
configs/
└── feed_sources.json     # Current source configuration
templates/
└── admin.html            # Admin UI template
```

### Desired Codebase Tree with New Files
```bash
internal/
├── db/
│   ├── db.go              # Enhanced with sources table, source CRUD
│   └── migrations.go      # Source table migration, JSON data import
├── api/
│   ├── source_handlers.go # New: Source CRUD API endpoints
│   └── admin_handlers.go  # Enhanced: Source management UI handlers
├── models/
│   └── source.go          # New: Source model and DTOs
└── rss/
    └── rss.go             # Modified: Load sources from database
templates/
├── admin.html             # Enhanced: Source management section
└── fragments/
    └── sources.html       # New: HTMX source management fragments
```

### Known Gotchas of Codebase & Library Quirks
```go
// CRITICAL: SQLite concurrency requires careful transaction handling
// Pattern: Use WithRetry() for SQLITE_BUSY errors (see db.go:426)
config := DefaultRetryConfig()
err := WithRetry(config, func() error {
    return insertSourceTransaction(db, source, &resultID)
})

// CRITICAL: Gin SafeHandler wrapper required for all API endpoints
// Pattern: router.POST("/api/sources", SafeHandler(createSourceHandler(dbConn)))

// CRITICAL: StandardResponse format must be consistent
// Pattern: RespondSuccess(c, data) and RespondError(c, NewAppError(...))

// CRITICAL: RSS Collector expects []string of URLs
// Pattern: Must convert database sources to URL slice for compatibility

// CRITICAL: HTMX fragments require specific template structure
// Pattern: Use gin.H{} for template data, match existing admin patterns
```

## Implementation Blueprint

### Data Models and Structure

Create extensible source models that support current RSS and future channels:

```go
// Source represents a news source with channel-specific configuration
type Source struct {
    ID           int64     `db:"id" json:"id"`
    Name         string    `db:"name" json:"name"`                    // Display name
    ChannelType  string    `db:"channel_type" json:"channel_type"`   // "rss", "telegram", etc.
    FeedURL      string    `db:"feed_url" json:"feed_url"`           // RSS URL or channel identifier
    Category     string    `db:"category" json:"category"`           // "left", "center", "right"
    Enabled      bool      `db:"enabled" json:"enabled"`
    DefaultWeight float64  `db:"default_weight" json:"default_weight"`
    LastFetchedAt *time.Time `db:"last_fetched_at" json:"last_fetched_at"`
    ErrorStreak   int      `db:"error_streak" json:"error_streak"`
    CreatedAt    time.Time `db:"created_at" json:"created_at"`
    UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// SourceStats represents aggregated statistics for a source
type SourceStats struct {
    SourceID     int64   `db:"source_id" json:"source_id"`
    ArticleCount int64   `db:"article_count" json:"article_count"`
    AvgScore     float64 `db:"avg_score" json:"avg_score"`
    ComputedAt   time.Time `db:"computed_at" json:"computed_at"`
}

// API DTOs
type CreateSourceRequest struct {
    Name         string  `json:"name" binding:"required"`
    ChannelType  string  `json:"channel_type" binding:"required"`
    FeedURL      string  `json:"feed_url" binding:"required"`
    Category     string  `json:"category" binding:"required"`
    DefaultWeight float64 `json:"default_weight"`
}

type UpdateSourceRequest struct {
    Name         *string  `json:"name,omitempty"`
    FeedURL      *string  `json:"feed_url,omitempty"`
    Category     *string  `json:"category,omitempty"`
    Enabled      *bool    `json:"enabled,omitempty"`
    DefaultWeight *float64 `json:"default_weight,omitempty"`
}
```

### List of Tasks to Complete the PRP

```yaml
Task 1: Database Schema & Migration
MODIFY internal/db/db.go:
  - ADD sources table schema to createTables()
  - ADD source_stats table for analytics
  - IMPLEMENT Source CRUD operations following Article patterns
  - ADD migration function to import from feed_sources.json

CREATE internal/db/source_migrations.go:
  - IMPLEMENT MigrateSourcesFromJSON() function
  - HANDLE existing JSON data import with validation
  - PRESERVE existing source categories and URLs

Task 2: Source Model & Business Logic
CREATE internal/models/source.go:
  - DEFINE Source struct with validation tags
  - IMPLEMENT SourceStats aggregation logic
  - ADD source health checking methods
  - FOLLOW existing model patterns from Article

Task 3: Source API Endpoints
CREATE internal/api/source_handlers.go:
  - IMPLEMENT GET /api/sources (list with pagination)
  - IMPLEMENT POST /api/sources (create new source)
  - IMPLEMENT PUT /api/sources/:id (update source)
  - IMPLEMENT DELETE /api/sources/:id (soft delete/disable)
  - IMPLEMENT GET /api/sources/:id/stats (source analytics)
  - FOLLOW SafeHandler pattern and StandardResponse format

MODIFY internal/api/api.go:
  - REGISTER new source endpoints in RegisterRoutes()
  - ADD Swagger annotations for documentation
  - MAINTAIN existing API patterns

Task 4: RSS Collector Integration
MODIFY internal/rss/rss.go:
  - REPLACE FeedURLs []string with database source loading
  - IMPLEMENT LoadSourcesFromDB() method
  - UPDATE health checking to use source IDs
  - MAINTAIN backward compatibility during transition

MODIFY cmd/server/main.go:
  - UPDATE RSS collector initialization
  - IMPLEMENT source migration on startup
  - PRESERVE existing feed loading fallback

Task 5: Admin UI Enhancement
MODIFY templates/admin.html:
  - ADD source management section
  - IMPLEMENT HTMX-powered source CRUD interface
  - ADD source statistics display
  - FOLLOW existing admin UI patterns

CREATE templates/fragments/sources.html:
  - IMPLEMENT source list fragment
  - ADD source form fragments (create/edit)
  - IMPLEMENT source stats fragments
  - USE existing CSS classes and styling

Task 6: Admin API Handlers
MODIFY internal/api/admin_handlers.go:
  - ADD source management handlers for admin UI
  - IMPLEMENT source statistics aggregation
  - ADD source health monitoring endpoints
  - FOLLOW existing admin handler patterns

Task 7: Integration & Testing
CREATE tests for all new functionality:
  - Unit tests for source CRUD operations
  - API integration tests for source endpoints
  - Admin UI E2E tests with Playwright
  - Migration testing with sample data
```

### Per Task Pseudocode

```go
// Task 1: Database Schema & Migration
func createSourcesTable(db *sqlx.DB) error {
    schema := `
    CREATE TABLE IF NOT EXISTS sources (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL UNIQUE,
        channel_type TEXT NOT NULL DEFAULT 'rss',
        feed_url TEXT NOT NULL,
        category TEXT NOT NULL,
        enabled BOOLEAN NOT NULL DEFAULT 1,
        default_weight REAL NOT NULL DEFAULT 1.0,
        last_fetched_at TIMESTAMP,
        error_streak INTEGER NOT NULL DEFAULT 0,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS source_stats (
        source_id INTEGER PRIMARY KEY,
        article_count INTEGER NOT NULL DEFAULT 0,
        avg_score REAL,
        computed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (source_id) REFERENCES sources (id)
    );`

    // PATTERN: Follow existing createTables() structure
    _, err := db.Exec(schema)
    return err
}

// Task 2: Source CRUD Operations
func InsertSource(db *sqlx.DB, source *Source) (int64, error) {
    // PATTERN: Follow InsertArticle retry logic for SQLITE_BUSY
    config := DefaultRetryConfig()
    var resultID int64

    err := WithRetry(config, func() error {
        return insertSourceTransaction(db, source, &resultID)
    })
    return resultID, err
}

// Task 3: API Handlers
func createSourceHandler(dbConn *sqlx.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req CreateSourceRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            RespondError(c, NewAppError(ErrValidation, "Invalid request body"))
            return
        }

        // PATTERN: Follow createArticleHandler validation
        source := &Source{
            Name:         req.Name,
            ChannelType:  req.ChannelType,
            FeedURL:      req.FeedURL,
            Category:     req.Category,
            Enabled:      true,
            DefaultWeight: req.DefaultWeight,
            CreatedAt:    time.Now(),
            UpdatedAt:    time.Now(),
        }

        id, err := InsertSource(dbConn, source)
        if err != nil {
            RespondError(c, NewAppError(ErrDatabase, "Failed to create source"))
            return
        }

        source.ID = id
        RespondSuccess(c, source)
    }
}

// Task 4: RSS Integration
func (c *Collector) LoadSourcesFromDB() error {
    // CRITICAL: Maintain []string interface for backward compatibility
    sources, err := FetchEnabledSources(c.DB)
    if err != nil {
        return err
    }

    // Convert to URL slice for existing RSS logic
    urls := make([]string, len(sources))
    for i, source := range sources {
        urls[i] = source.FeedURL
    }

    c.FeedURLs = urls
    return nil
}

// Task 5: Admin UI Handler
func adminSourcesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        sources, err := FetchAllSources(dbConn)
        if err != nil {
            c.HTML(http.StatusInternalServerError, "admin.html", gin.H{
                "Error": "Failed to load sources",
            })
            return
        }

        // PATTERN: Follow existing admin handler structure
        c.HTML(http.StatusOK, "admin.html", gin.H{
            "Sources": sources,
            "Title":   "Source Management",
        })
    }
}
```

### Integration Points
```yaml
DATABASE:
  - migration: "CREATE sources and source_stats tables"
  - index: "CREATE INDEX idx_sources_enabled ON sources(enabled)"
  - index: "CREATE INDEX idx_sources_channel_type ON sources(channel_type)"

CONFIG:
  - modify: cmd/server/main.go
  - pattern: "Add source migration call in initServices()"

ROUTES:
  - add to: internal/api/api.go
  - pattern: "router.GET('/api/sources', SafeHandler(getSourcesHandler(dbConn)))"
  - pattern: "router.POST('/api/sources', SafeHandler(createSourceHandler(dbConn)))"

TEMPLATES:
  - enhance: templates/admin.html
  - add: templates/fragments/sources.html
  - pattern: "HTMX fragments for dynamic source management"
```

## Validation Loop

### Level 1: Syntax & Style
```bash
# Run these FIRST - fix any errors before proceeding
go fmt ./internal/...                    # Format code
go vet ./internal/...                    # Static analysis
golangci-lint run ./internal/...         # Comprehensive linting

# Expected: No errors. If errors, READ the error and fix.
```

### Level 2: Unit Tests
```go
// CREATE internal/db/source_test.go
func TestInsertSource(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    source := &Source{
        Name:        "Test Source",
        ChannelType: "rss",
        FeedURL:     "https://example.com/feed.xml",
        Category:    "center",
        Enabled:     true,
        DefaultWeight: 1.0,
    }

    id, err := InsertSource(db, source)
    assert.NoError(t, err)
    assert.Greater(t, id, int64(0))
}

// CREATE internal/api/source_handlers_test.go
func TestCreateSourceHandler(t *testing.T) {
    router := setupTestRouter()

    reqBody := CreateSourceRequest{
        Name:        "Test Source",
        ChannelType: "rss",
        FeedURL:     "https://example.com/feed.xml",
        Category:    "center",
        DefaultWeight: 1.0,
    }

    w := performRequest(router, "POST", "/api/sources", reqBody)
    assert.Equal(t, http.StatusOK, w.Code)

    var response StandardResponse
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.True(t, response.Success)
}
```

```bash
# Run and iterate until passing:
go test ./internal/db -v
go test ./internal/api -v
# If failing: Read error, understand root cause, fix code, re-run
```

### Level 3: Integration Test
```bash
# Start the service with migration
go run cmd/server/main.go

# Test source creation
curl -X POST http://localhost:8080/api/sources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test RSS Source",
    "channel_type": "rss",
    "feed_url": "https://example.com/feed.xml",
    "category": "center",
    "default_weight": 1.0
  }'

# Expected: {"success": true, "data": {"id": 1, ...}}

# Test source listing
curl http://localhost:8080/api/sources

# Expected: {"success": true, "data": [{"id": 1, ...}]}
```

### Level 4: E2E Admin UI Test
```typescript
// CREATE tests/e2e/source-management.spec.ts
test('admin can create and manage sources', async ({ page }) => {
  await page.goto('/admin');

  // Navigate to source management section
  await page.click('[data-testid="source-management"]');

  // Create new source
  await page.click('[data-testid="add-source-btn"]');
  await page.fill('[name="name"]', 'Test Source');
  await page.fill('[name="feed_url"]', 'https://example.com/feed.xml');
  await page.selectOption('[name="category"]', 'center');
  await page.click('[data-testid="save-source-btn"]');

  // Verify source appears in list
  await expect(page.locator('[data-testid="source-list"]')).toContainText('Test Source');
});
```

```bash
# Run E2E tests
npx playwright test tests/e2e/source-management.spec.ts
# Expected: All tests pass with source CRUD functionality working
```

## Final Validation Checklist
- [ ] All tests pass: `go test ./... -v`
- [ ] No linting errors: `golangci-lint run`
- [ ] Migration successful: Sources imported from JSON
- [ ] API endpoints functional: All CRUD operations work
- [ ] Admin UI responsive: Source management interface works
- [ ] RSS collection unchanged: Existing feeds still work
- [ ] Source analytics displayed: Stats show in admin UI
- [ ] Health monitoring active: Source errors tracked
- [ ] E2E tests pass: `npx playwright test`

---

## Anti-Patterns to Avoid
- ❌ Don't break existing RSS functionality during migration
- ❌ Don't skip database transaction retry logic for SQLite
- ❌ Don't ignore SafeHandler wrapper for API endpoints
- ❌ Don't use different response format than StandardResponse
- ❌ Don't hardcode channel types - make them extensible
- ❌ Don't skip source validation - ensure URLs are valid
- ❌ Don't forget HTMX patterns for admin UI fragments
