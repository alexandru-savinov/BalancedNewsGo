# Source Management Implementation - Task Breakdown

## Task Organization
Each task is designed to take 15-30 minutes with clear deliverables, acceptance criteria, and rollback steps.

---

## Phase 1: Database Foundation

### Task 1.1: Create Source Database Schema
**Deliverable:** Sources and source_stats tables created with proper indexes
**Time Estimate:** 20 minutes
**Dependencies:** None

**Acceptance Criteria:**
- [ ] Sources table created with all required columns
- [ ] Source_stats table created with foreign key constraint
- [ ] All indexes created (enabled, channel_type, category)
- [ ] Schema validation passes
- [ ] No breaking changes to existing tables

**Implementation Steps:**
1. Modify `internal/db/db.go` - add source schema to `createTables()`
2. Add source table creation SQL
3. Add source_stats table creation SQL  
4. Add indexes for performance
5. Test schema creation with fresh database

**Validation:**
```bash
# Test database creation
rm -f test.db
go run cmd/server/main.go &
sleep 2
sqlite3 test.db ".schema sources"
sqlite3 test.db ".schema source_stats"
pkill -f "cmd/server/main.go"
```

**Rollback:** Remove source table creation from schema

---

### Task 1.2: Implement Source CRUD Operations
**Deliverable:** Database operations for source management
**Time Estimate:** 25 minutes
**Dependencies:** Task 1.1

**Acceptance Criteria:**
- [ ] InsertSource() function with retry logic
- [ ] UpdateSource() function with validation
- [ ] FetchSources() with filtering and pagination
- [ ] FetchSourceByID() function
- [ ] SoftDeleteSource() function (disable, don't delete)
- [ ] All functions follow existing db.go patterns

**Implementation Steps:**
1. Add Source struct to `internal/db/db.go`
2. Implement InsertSource() with SQLite retry logic
3. Implement UpdateSource() with partial updates
4. Implement FetchSources() with filters
5. Implement FetchSourceByID()
6. Implement SoftDeleteSource() (set enabled=false)

**Validation:**
```go
// Unit test in internal/db/source_test.go
func TestSourceCRUD(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // Test create
    source := &Source{Name: "Test", ChannelType: "rss", FeedURL: "https://test.com", Category: "center"}
    id, err := InsertSource(db, source)
    assert.NoError(t, err)
    
    // Test read
    fetched, err := FetchSourceByID(db, id)
    assert.NoError(t, err)
    assert.Equal(t, "Test", fetched.Name)
    
    // Test update
    err = UpdateSource(db, id, &UpdateSourceRequest{Name: stringPtr("Updated")})
    assert.NoError(t, err)
    
    // Test soft delete
    err = SoftDeleteSource(db, id)
    assert.NoError(t, err)
}
```

**Rollback:** Remove source CRUD functions

---

### Task 1.3: Create Migration from JSON Config
**Deliverable:** Migration function to import existing sources
**Time Estimate:** 20 minutes
**Dependencies:** Task 1.2

**Acceptance Criteria:**
- [ ] MigrateSourcesFromJSON() function created
- [ ] Reads existing `configs/feed_sources.json`
- [ ] Imports sources with proper mapping
- [ ] Handles duplicate sources gracefully
- [ ] Preserves existing source categories
- [ ] Migration is idempotent (can run multiple times safely)

**Implementation Steps:**
1. Create `internal/db/migrations.go`
2. Implement MigrateSourcesFromJSON() function
3. Map JSON structure to Source model
4. Handle existing sources (skip duplicates)
5. Add migration call to server startup

**Validation:**
```bash
# Test migration
cp configs/feed_sources.json configs/feed_sources.json.backup
go run cmd/server/main.go &
sleep 3
sqlite3 news.db "SELECT COUNT(*) FROM sources;"
# Should show count matching JSON sources
pkill -f "cmd/server/main.go"
```

**Rollback:** Remove migration call from startup

---

## Phase 2: API Layer

### Task 2.1: Create Source Model and DTOs
**Deliverable:** Source models and API request/response structures
**Time Estimate:** 15 minutes
**Dependencies:** Task 1.2

**Acceptance Criteria:**
- [ ] Source model with JSON tags
- [ ] CreateSourceRequest DTO with validation
- [ ] UpdateSourceRequest DTO with optional fields
- [ ] SourceWithStats composite model
- [ ] All models follow existing API patterns

**Implementation Steps:**
1. Create `internal/models/source.go`
2. Define Source struct with proper tags
3. Define API request/response DTOs
4. Add validation tags (binding:"required", etc.)
5. Add helper functions for pointer conversions

**Validation:**
```go
// Test JSON marshaling/unmarshaling
func TestSourceSerialization(t *testing.T) {
    source := Source{Name: "Test", ChannelType: "rss"}
    data, err := json.Marshal(source)
    assert.NoError(t, err)
    
    var unmarshaled Source
    err = json.Unmarshal(data, &unmarshaled)
    assert.NoError(t, err)
    assert.Equal(t, source.Name, unmarshaled.Name)
}
```

**Rollback:** Remove source model file

---

### Task 2.2: Implement Source API Handlers
**Deliverable:** REST API endpoints for source management
**Time Estimate:** 30 minutes
**Dependencies:** Task 2.1

**Acceptance Criteria:**
- [ ] GET /api/sources with filtering and pagination
- [ ] POST /api/sources with validation
- [ ] PUT /api/sources/:id with partial updates
- [ ] DELETE /api/sources/:id (soft delete)
- [ ] GET /api/sources/:id/stats
- [ ] All handlers use SafeHandler wrapper
- [ ] All responses use StandardResponse format
- [ ] Proper error handling and validation

**Implementation Steps:**
1. Create `internal/api/source_handlers.go`
2. Implement getSourcesHandler() with filters
3. Implement createSourceHandler() with validation
4. Implement updateSourceHandler() with partial updates
5. Implement deleteSourceHandler() (soft delete)
6. Implement getSourceStatsHandler()

**Validation:**
```bash
# Test API endpoints
curl -X POST http://localhost:8080/api/sources \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","channel_type":"rss","feed_url":"https://test.com","category":"center"}'

curl http://localhost:8080/api/sources
curl -X PUT http://localhost:8080/api/sources/1 -d '{"enabled":false}'
curl -X DELETE http://localhost:8080/api/sources/1
```

**Rollback:** Remove source handlers file

---

### Task 2.3: Register Source API Routes
**Deliverable:** Source endpoints registered in API router
**Time Estimate:** 10 minutes
**Dependencies:** Task 2.2

**Acceptance Criteria:**
- [ ] All source endpoints registered in RegisterRoutes()
- [ ] Swagger annotations added for documentation
- [ ] Routes follow existing API patterns
- [ ] Proper middleware applied (SafeHandler)

**Implementation Steps:**
1. Modify `internal/api/api.go`
2. Add source route registrations to RegisterRoutes()
3. Add Swagger annotations for each endpoint
4. Test route registration

**Validation:**
```bash
# Check Swagger docs
curl http://localhost:8080/swagger/doc.json | jq '.paths | keys | map(select(contains("sources")))'
```

**Rollback:** Remove source route registrations

---

## Phase 3: RSS Integration

### Task 3.1: Modify RSS Collector for Database Sources
**Deliverable:** RSS collector loads sources from database
**Time Estimate:** 25 minutes
**Dependencies:** Task 1.3

**Acceptance Criteria:**
- [ ] LoadSourcesFromDB() method implemented
- [ ] RSS collector uses database sources instead of JSON
- [ ] Backward compatibility maintained during transition
- [ ] Source health checking updated to use source IDs
- [ ] Error tracking updates source error_streak

**Implementation Steps:**
1. Modify `internal/rss/rss.go`
2. Add LoadSourcesFromDB() method
3. Update FetchAndStore() to use database sources
4. Update health checking to track by source ID
5. Update error handling to increment error_streak

**Validation:**
```bash
# Test RSS collection with database sources
go run cmd/server/main.go &
sleep 5
# Trigger manual refresh
curl -X POST http://localhost:8080/api/refresh
# Check articles were created
sqlite3 news.db "SELECT COUNT(*) FROM articles WHERE created_at > datetime('now', '-1 minute');"
pkill -f "cmd/server/main.go"
```

**Rollback:** Revert RSS collector to use JSON config

---

### Task 3.2: Update Server Initialization
**Deliverable:** Server startup uses database sources
**Time Estimate:** 15 minutes
**Dependencies:** Task 3.1

**Acceptance Criteria:**
- [ ] Server initialization calls source migration
- [ ] RSS collector initialized with database sources
- [ ] Fallback to JSON config if migration fails
- [ ] Proper error handling and logging

**Implementation Steps:**
1. Modify `cmd/server/main.go`
2. Add migration call to initServices()
3. Update RSS collector initialization
4. Add fallback mechanism
5. Add proper error logging

**Validation:**
```bash
# Test server startup
go run cmd/server/main.go
# Check logs for migration success
# Check RSS collector is working
```

**Rollback:** Remove migration call and database source loading

---

## Phase 4: Admin Interface

### Task 4.1: Create Source Management UI Components
**Deliverable:** HTMX fragments for source management
**Time Estimate:** 25 minutes
**Dependencies:** Task 2.3

**Acceptance Criteria:**
- [ ] Source list fragment with pagination
- [ ] Source create/edit form fragment
- [ ] Source statistics display fragment
- [ ] Delete confirmation modal
- [ ] All fragments use existing CSS classes
- [ ] HTMX interactions work properly

**Implementation Steps:**
1. Create `templates/fragments/sources.html`
2. Implement source list fragment
3. Implement source form fragment
4. Implement source stats fragment
5. Add HTMX attributes for dynamic updates

**Validation:**
```bash
# Test HTMX fragments
curl http://localhost:8080/htmx/sources
curl http://localhost:8080/htmx/sources/1/form
```

**Rollback:** Remove source fragments file

---

### Task 4.2: Enhance Admin Dashboard
**Deliverable:** Source management section in admin interface
**Time Estimate:** 20 minutes
**Dependencies:** Task 4.1

**Acceptance Criteria:**
- [ ] Source management section added to admin.html
- [ ] Integration with HTMX fragments
- [ ] Source statistics displayed
- [ ] Consistent with existing admin UI style
- [ ] Responsive design maintained

**Implementation Steps:**
1. Modify `templates/admin.html`
2. Add source management section
3. Integrate HTMX fragments
4. Add source statistics display
5. Test responsive design

**Validation:**
```bash
# Visual test
go run cmd/server/main.go &
# Open http://localhost:8080/admin in browser
# Test source management functionality
```

**Rollback:** Remove source management section from admin.html

---

### Task 4.3: Implement Admin Source Handlers
**Deliverable:** Admin-specific handlers for source management UI
**Time Estimate:** 20 minutes
**Dependencies:** Task 4.2

**Acceptance Criteria:**
- [ ] Admin source list handler
- [ ] Admin source form handlers
- [ ] Source statistics aggregation
- [ ] HTMX fragment handlers
- [ ] Proper error handling for UI

**Implementation Steps:**
1. Modify `internal/api/admin_handlers.go`
2. Add adminSourcesHandler()
3. Add adminSourceFormHandler()
4. Add adminSourceStatsHandler()
5. Add HTMX fragment handlers

**Validation:**
```bash
# Test admin handlers
curl http://localhost:8080/admin/sources
curl http://localhost:8080/htmx/sources
```

**Rollback:** Remove admin source handlers

---

## Phase 5: Testing & Validation

### Task 5.1: Create Unit Tests
**Deliverable:** Comprehensive unit test coverage
**Time Estimate:** 30 minutes
**Dependencies:** All previous tasks

**Acceptance Criteria:**
- [ ] Database CRUD operation tests
- [ ] API handler tests
- [ ] Migration tests
- [ ] Model validation tests
- [ ] Test coverage > 80%

**Implementation Steps:**
1. Create `internal/db/source_test.go`
2. Create `internal/api/source_handlers_test.go`
3. Create migration tests
4. Run test coverage analysis
5. Fix any failing tests

**Validation:**
```bash
go test ./internal/... -v -cover
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out
```

**Rollback:** Remove test files

---

### Task 5.2: Create E2E Tests
**Deliverable:** End-to-end tests for source management
**Time Estimate:** 25 minutes
**Dependencies:** Task 5.1

**Acceptance Criteria:**
- [ ] Admin UI source management test
- [ ] API integration tests
- [ ] Source creation/update/delete flow
- [ ] RSS integration test
- [ ] All tests pass consistently

**Implementation Steps:**
1. Create `tests/e2e/source-management.spec.ts`
2. Test admin UI interactions
3. Test API endpoints
4. Test RSS integration
5. Run full test suite

**Validation:**
```bash
npx playwright test tests/e2e/source-management.spec.ts
```

**Rollback:** Remove E2E test files

---

## Final Integration Checklist

### Pre-Deployment Validation
- [ ] All unit tests pass: `go test ./... -v`
- [ ] All E2E tests pass: `npx playwright test`
- [ ] No linting errors: `golangci-lint run`
- [ ] Migration successful with existing data
- [ ] RSS collection works with database sources
- [ ] Admin UI fully functional
- [ ] API endpoints documented in Swagger
- [ ] Performance requirements met
- [ ] Security validation complete

### Rollback Plan
Each task has specific rollback steps. Full rollback:
1. Revert server initialization changes
2. Remove source API routes
3. Remove source handlers
4. Remove admin UI changes
5. Remove database schema changes
6. Restore JSON-based RSS collection

### Success Metrics
- Source management fully functional via admin UI
- API endpoints working and documented
- RSS collection unchanged from user perspective
- Zero downtime during migration
- All existing functionality preserved
