# Plan to Resolve SQLITE_BUSY Errors

## 1. Understanding the Problem

The `SQLITE_BUSY` errors indicate that multiple parts of the application are trying to write to the SQLite database concurrently, or one operation is holding a lock for too long, preventing others from acquiring it. This is frequently observed during the Postman API tests, which likely issue several write-heavy requests in parallel or rapid succession (e.g., creating multiple articles).

The primary goal of this plan is to systematically investigate and implement solutions to allow the Go backend to handle concurrent database operations more gracefully, specifically those triggered by the Postman test suite.

## 2. Investigation Steps

The following steps will be taken to pinpoint the exact causes and locations of the `SQLITE_BUSY` errors:

*   **A. Review SQLite Configuration in `internal/db/db.go`:**
    *   **Examine `InitDB()` function:**
        *   Verify if `PRAGMA journal_mode=WAL;` is being set. Write-Ahead Logging (WAL) is crucial for improving concurrency in SQLite by allowing readers to operate while a writer is active.
        *   Check if `PRAGMA busy_timeout = <value>;` is configured. This pragma instructs SQLite to wait for a specified duration if the database is locked, rather than immediately returning an `SQLITE_BUSY` error. The current value (if any) will be noted.
        *   Identify any other PRAGMAs that might affect locking behavior or concurrency (e.g., `PRAGMA synchronous`).
*   **B. Analyze Database Transaction Usage and Scope:**
    *   **Article Creation Workflow:**
        *   Trace the execution path from `internal/api/article_handlers.go` (specifically `createArticleHandler`) to `internal/db/db.go` (specifically `InsertArticle` and any related functions).
        *   Map out the boundaries of database transactions: when they begin, when they commit or roll back.
        *   Identify any long-running operations (e.g., complex computations, external API calls *before* all DB writes are done) that might be holding transactions open unnecessarily.
    *   **Article Re-analysis Workflow:**
        *   Trace the execution path in `internal/llm/llm.go` (specifically `ReanalyzeArticle`) and its interactions with database functions in `internal/db/db.go` (e.g., `InsertLLMScore`, `FetchLLMScores`, `UpdateArticle`).
        *   Analyze transaction management: Are there multiple small transactions, or one large transaction spanning potentially slow operations like LLM API calls?
        *   Determine if LLM API calls are made *within* an active database transaction.
    *   **Other Write Operations:**
        *   Review other API handlers that are part of the failing Postman tests and perform database writes.
        *   Assess their transaction management for similar patterns of long-held locks or overly broad transaction scopes.
*   **C. Review Database Connection Management (`internal/db/db.go`):**
    *   Confirm that the `sqlx.DB` instance is initialized as a singleton and shared correctly across goroutines (this is standard Go practice, as `sql.DB` it wraps is a connection pool and safe for concurrent use).
    *   Check the value of `db.SetMaxOpenConns()`. While SQLite is an embedded database, this setting might still influence behavior if misconfigured. The default (0, meaning unlimited) is generally appropriate for SQLite.
*   **D. Examine Postman Test Structure and Execution:**
    *   Identify precisely which Postman test requests are failing due to `SQLITE_BUSY` errors.
    *   Analyze the Postman collection to understand if these failing tests are intentionally run in parallel or in very rapid succession, targeting write operations that could conflict.

## 3. Proposed Solutions & Implementation Strategy

Based on the investigation, the following solutions will be considered and implemented, generally in order of preference (from least invasive/highest impact to more complex):

*   **A. Enable WAL (Write-Ahead Logging) Mode (High Priority):**
    *   **Action:** If not already enabled, modify `InitDB()` in `internal/db/db.go` to execute `PRAGMA journal_mode=WAL;` as one ofthe first operations after opening the database connection.
    *   **Rationale:** WAL mode is the most significant improvement for SQLite concurrency, allowing one writer and multiple readers to operate simultaneously.
    *   **Verification:** After server startup, connect to the SQLite database (e.g., using the `sqlite3` CLI tool or a database browser) and execute the command `PRAGMA journal_mode;`. The expected result should be `wal`. Alternatively, this check can be added to a startup log message in the Go application.
*   **B. Set/Increase `busy_timeout` (High Priority):**
    *   **Action:** In `InitDB()` in `internal/db/db.go`, add or adjust `PRAGMA busy_timeout = 5000;` (to wait up to 5 seconds). The value can be tuned.
    *   **Rationale:** This gives SQLite operations a grace period to wait for a lock to be released if the database is temporarily busy, rather than failing immediately.
*   **C. Optimize Transaction Scopes and Duration (Medium Priority):**
    *   **Action:** Based on findings in Investigation Step 2.B, refactor relevant code sections:
        *   Ensure database transactions are as short-lived as possible, encompassing only the operations that *must* be atomic.
        *   Move read operations (if they don't need to be part of the atomic write) outside and before the transaction begins.
        *   Critically, ensure that slow external operations (like API calls to the LLM service) are **not** performed while a database transaction is being held open. The pattern should be:
            1.  Start transaction (if needed for pre-reads).
            2.  Read necessary data.
            3.  Commit/rollback transaction.
            4.  Perform external API call.
            5.  Start new transaction.
            6.  Write results to the database.
            7.  Commit/rollback transaction.
    *   **Rationale:** Shorter transactions reduce the window during which locks are held, minimizing contention.
*   **D. Review and Refine Database Access Patterns (Medium Priority):**
    *   **Action:** For operations identified as highly contentious, evaluate if the database interaction can be restructured. For example, batching updates if applicable, or using more granular updates.
    *   **Rationale:** Optimizing how data is read and written can reduce the load and locking.
*   **E. Application-Level Retry Mechanisms (Lower Priority / If Above Are Insufficient):**
    *   **Action:** For critical operations that might still occasionally face `SQLITE_BUSY` errors despite other optimizations, consider implementing a retry loop with an exponential backoff strategy directly in the Go application code.
    *   **Rationale:** This adds resilience at the application layer but also increases complexity. It should be a targeted solution for specific known contention points if other measures are not fully effective.
*   **F. Sequentialize Conflicting Postman Tests (Workaround / Last Resort):**
    *   **Action:** If backend changes prove insufficient or too complex for the immediate scope, and specific Postman tests are identified as the primary triggers due to their parallel nature, investigate options within Newman or the Postman collection to enforce sequential execution for those specific tests or introduce small delays.
    *   **Rationale:** This is a workaround to allow tests to pass by mitigating the trigger, not a fundamental fix for the backend's concurrency handling.

## 4. Current Status & Findings

**✅ COMPLETED:**
*   **WAL mode enabled** - Successfully implemented and verified with log message "WAL mode enabled successfully"
*   **`busy_timeout` already set** - Found existing `PRAGMA busy_timeout = 5000` configuration

**🔍 CURRENT ISSUE ANALYSIS:**
After implementing WAL mode, test results show:
*   **SQLITE_BUSY errors still occur** during article insertion: `database is locked (5) (SQLITE_BUSY)`
*   **Variable substitution failures** in Postman tests ({{sseArticleId}}, {{confidenceArticleId}} not being replaced)
*   **Server errors (500)** due to SQLITE_BUSY during concurrent article creation
*   **API validation errors (400)** due to missing environment variables

**🎯 NEXT ACTIONS NEEDED:**
1.  Investigate transaction scopes in article insertion workflow
2.  Implement application-level retry mechanisms for SQLITE_BUSY
3.  Fix Postman variable substitution issues

## 5. Testing and Validation Strategy

A rigorous testing approach will be followed after each significant change:

*   **Incremental Testing:**
    1.  After implementing a proposed solution (e.g., enabling WAL mode, adjusting `busy_timeout`, refactoring a specific transaction block), restart the backend server (`make run`).
    2.  Execute the full Postman test suite: `npx newman run .\postman\unified_backend_tests.json -e .\postman\local_environment.json`
    3.  Thoroughly examine the Newman output for any `SQLITE_BUSY` errors or changes in test failure patterns.
    4.  Concurrently, monitor the server logs for `SQLITE_BUSY` messages, stack traces, or other database-related errors.
*   **Regression Testing:**
    *   Ensure that all backend unit tests continue to pass: `make unit`
    *   Ensure that all backend Go integration tests continue to pass: `make integration`
*   **Iterative Refinement:**
    *   If `SQLITE_BUSY` errors persist, analyze the context in which they occur and proceed to the next proposed solution in the plan, or revisit and refine previous implementations.
    *   The `busy_timeout` value may require tuning based on observed behavior.

## 5. Deliverable

*   This documented plan, saved as `d:\Dev\newbalancer_go_fe3 - Copy\docs\PR\sqlite_busy_resolution_plan.md`.
*   Modified Go source files (primarily in `internal/db/db.go`, `internal/api/`, and `internal/llm/`) that implement the chosen solutions.
*   Confirmation that all Postman tests pass without `SQLITE_BUSY` errors.

This structured approach aims to methodically diagnose and resolve the SQLite concurrency issues, ensuring the backend API remains robust and reliable under test conditions and, by extension, under concurrent user load.
