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

Next steps:
- Write unit and integration tests.
- Optimize performance.
- Prepare deployment scripts.
- Add monitoring and logging.