                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               # Politically Balanced News Aggregator - LLM Subsystem Task Context

## Task Scope
- Implement LLM analysis subsystem in Go
- Use `internal/llm/` package
- Batch process articles missing LLM scores
- For each article, call 3 LLM API endpoints (left, center, right) using resty
- Parse/store scores + metadata in `llm_scores` table
- Cache responses (in-memory or SQLite)
- Handle API failures with retries and logging
- Provide function to reanalyze a specific article
- No API endpoints or frontend integration yet

---

## Current Implementation Status

- Created `internal/llm/llm.go` with:
  - In-memory cache
  - LLM API client with retries
  - Batch processing of unscored articles
  - Reanalyze function
  - Integration with existing DB layer (`internal/db/db.go`)

- **DB Layer (`internal/db/db.go`):**

- **RSS Pipeline (`internal/rss/rss.go`):**
  - Fetches and deduplicates articles from feeds
  - After inserting new articles, immediately calls LLM analysis service
  - Stores political scores linked to articles
  - Fully integrated with LLM subsystem
  - Defines `Article` and `LLMScore` structs
  - Has insert/query functions
  - Schema includes `articles` and `llm_scores` tables

---

## API Layer (`internal/api/api.go`):
- `/api/articles` GET with filters and pagination
- `/api/articles/:id` GET with article details and scores
- `/api/refresh` POST to trigger RSS refresh
- `/api/llm/reanalyze/:id` POST to re-run analysis
- Fully integrated with DB, RSS, and LLM subsystems

---

## Outstanding Issues

- **Go command not found:** `go` CLI is not recognized, so module init and dependency installation failed.
- **Missing dependencies:**
  - `github.com/go-resty/resty/v2`
  - `github.com/jmoiron/sqlx`
  - `github.com/mattn/go-sqlite3`
- **Unused import:** `"fmt"` in `llm.go` should be removed.
- **Local import error:** `internal/db` import may fail until Go modules are initialized properly.

---

## Next Steps After Restart

1. **Ensure Go is installed and added to PATH.**
2. Initialize Go module in project root:
   ```
   go mod init newbalancer_go
   ```
3. Install dependencies:
   ```
   go get github.com/go-resty/resty/v2
   go get github.com/jmoiron/sqlx
   go get github.com/mattn/go-sqlite3
   ```
4. Remove unused `"fmt"` import from `internal/llm/llm.go`.
5. Build and test the LLM subsystem.

---

## Environment Details

- Project root: `c:/Users/user/Documents/dev/news_filter/newbalancer_go`
- Key files:
  - `internal/db/db.go`
  - `internal/llm/llm.go`
- No existing Go module (`go.mod`) yet
- SQLite database initialized via `InitDB` in `db.go`

---

## Summary

The entire backend, LLM subsystem, RSS pipeline, and API are now implemented and successfully compiled.

Go module and dependencies are properly configured.

---

## Next Phase: Testing, Optimization, Deployment

### Chunk 3.1: Write unit tests for DB, LLM, RSS packages
### Chunk 3.2: Write integration tests for API endpoints
---

### Progress Update:
- **Chunk 3.1:** Unit tests for DB package (`internal/db/db_test.go`) created, covering article and LLM score insert/fetch.

Next: unit tests for LLM package.
### Chunk 3.3: Optimize performance (caching, concurrency)
### Chunk 3.4: Prepare deployment scripts/configs
---

## UI Implementation Progress (as of 2025-04-07)

- Basic UI with htmx is functional.
- Supports filtering by source and leaning.
- Pagination controls implemented.
- Refresh button triggers RSS feed update.
- Article detail view loads dynamically.
- Integration tests for API and UI endpoints pass.
- Backend political analysis calls fail if LLM microservices are offline (expected).

Next steps:
- Add reanalyze buttons.
- Improve styling.
- Deploy LLM microservices for full functionality.
---

## Upcoming Phase: Simple UI with htmx

### Phase 1: Basic UI Setup
- Chunk 1.1: Set up static file serving in Gin
- Chunk 1.2: Create base HTML template with htmx included
- Chunk 1.3: Implement homepage listing articles using htmx GET

### Phase 2: Article Details and Filtering
- Chunk 2.1: Add article detail view with htmx
- Chunk 2.2: Add filters (source, leaning) with htmx requests
- Chunk 2.3: Add pagination with htmx

### Phase 3: Interactivity and Enhancements
- Chunk 3.1: Add refresh button triggering `/api/refresh` via htmx POST
- Chunk 3.2: Add reanalyze button per article triggering `/api/llm/reanalyze/:id`
- Chunk 3.3: Style with minimal CSS for usability

Will proceed chunk by chunk and update this log accordingly.
### Chunk 3.5: Add monitoring and logging enhancements

Will proceed chunk by chunk and update this log accordingly.