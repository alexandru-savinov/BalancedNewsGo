# Debugging Plan for Test Failures - 2025-05-10

## 1. Observed Issues

*   Initial test runs (`scripts/test.cmd all`) failed with `connect ECONNREFUSED 127.0.0.1:8080` (server not running).
*   After manual server start (`go run cmd/server/main.go`), `scripts/test.cmd all` still failed with 7 assertion failures (500/503 errors in `test-results\all_tests_run_10.05.20251229.log`).
*   `README.md` mandates `NO_AUTO_ANALYZE=true` for Newman tests to prevent SQLite concurrency issues.
*   Server logs (without `NO_AUTO_ANALYZE=true`) showed `SQLITE_BUSY` errors.
*   **Recent Server Log Analysis (with server running, during a period similar to test execution based on ArticleID 4066 processing):**
    *   `[ERROR] Error in calculateCompositeScore: failed to get valid scores from any LLM perspective. actualValidCount=3`. This occurs even when all LLMs return 0.0 (neutral). Suggests all-zero scores are treated as "no valid score for calculation" but results in a default of 0 being stored. Logging could be clearer if this is expected.
    *   `[ERROR] Error inserting/updating ensemble score post-commit: SQL logic error: ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint (1)`. This is a **critical database error**, indicating a mismatch between the Go application's SQL query (likely for `ensemble_scores` or `articles` table update) and the actual table schema's constraints.

## 2. Hypothesis

There are likely **two primary issues** contributing to test failures:

1.  **Missing `NO_AUTO_ANALYZE=true` for Server Process:** When the server is started manually with `go run cmd/server/main.go` without this environment variable, background LLM analysis (involving DB writes via `internal/llm/score_manager.go` and `internal/db/`) causes SQLite database lock contention (`SQLITE_BUSY`). This leads to API request failures (500/503) and test assertion failures.
2.  **SQL `ON CONFLICT` Clause Mismatch:** Independent of the concurrency issue, there's an underlying bug where an `INSERT` or `UPDATE` statement for saving scores (likely ensemble scores) uses an `ON CONFLICT` clause that refers to a column (or columns) not actually covered by a `PRIMARY KEY` or `UNIQUE` constraint in the relevant database table. This SQL error will cause relevant API operations to fail, leading to test failures even if the concurrency issue is resolved.

The `test.cmd` script likely sets `NO_AUTO_ANALYZE=true` for its Newman execution environment, but this doesn't affect a separately launched server unless the variable is also passed to that server's environment.

## 3. Debugging and Verification Steps

1.  **Ensure a Clean Environment and Correct Server Launch (Addressing Hypothesis 1):**
    *   Terminate any lingering `go.exe` or `newsbalancer_server.exe` processes.
    *   Clean previous test results: `scripts/test.cmd clean`.
    *   Verify `.env` and `.env.example` for any `NO_AUTO_ANALYZE` mentions. (Likely needs explicit setting for `go run`).
    *   Start the server in the background with `NO_AUTO_ANALYZE=true` explicitly set:
        ```powershell
        $env:NO_AUTO_ANALYZE='true'; go run cmd/server/main.go
        ```
    *   Wait ~10-15 seconds. **Crucially, check server startup logs** for `NO_AUTO_ANALYZE` recognition and absence of immediate DB errors.

2.  **Targeted Test Execution (Isolating Hypothesis 1 impact):**
    *   Run the `essential` test suite: `scripts/test.cmd essential`.

3.  **Analyze Results from Step 2:**
    *   **If `essential` tests pass:** This indicates `NO_AUTO_ANALYZE=true` was the main blocker for this suite. Proceed to run `scripts/test.cmd all`. If `all` also passes, both primary issues might have been resolved by this or were not hit by `essential`.
    *   **If `essential` tests still fail:**
        *   Examine the new test log (`test-results/`) and server logs.
        *   **Prioritize looking for the `SQL logic error: ON CONFLICT clause does not match...` error in server logs.** Its presence confirms Hypothesis 2 is now the likely dominant issue.
        *   If `SQLITE_BUSY` errors persist, the `NO_AUTO_ANALYZE=true` variable is still not effective for the `go run` process. Re-check PowerShell environment variable scoping for background processes.

4.  **Root Cause Analysis of SQL Issue (New):**
    *   Review version control history for recent changes to:
        *   `internal/db/db.go` - Look for schema changes, especially to tables used for article scoring
        *   `internal/llm/score_manager.go` - Look for altered SQL queries or score persistence logic
        *   Other files that might manipulate ensemble scores
    *   Use `git blame` on the relevant files to identify when these components were last modified and by whom.
    *   Check for any parallel PRs or branches that might have changed the schema or SQL queries without coordinating.
    *   Examine if this is a regression or a long-standing issue that wasn't previously exposed (e.g., due to test conditions not triggering this particular code path).

5.  **Investigate and Fix SQL `ON CONFLICT` Mismatch (Addressing Hypothesis 2):**
    *   Based on server logs (or by stepping through code if necessary), identify the exact Go function and SQL query causing the `ON CONFLICT` error. This is likely in `internal/db/db.go` (e.g., `UpdateArticleScoreLLM` or a similar function for ensemble scores) or `internal/llm/score_manager.go`.
    *   Review the `createSchema` function in `internal/db/db.go` to find the DDL for the target table.
    *   Compare the columns specified in the `ON CONFLICT` clause of the problematic query against the `PRIMARY KEY` and `UNIQUE` constraints defined for that table.
    *   **Resolution Options:**
        *   **Option A - Modify SQL Query:** Adjust the Go code to use an existing valid constraint in the `ON CONFLICT` clause. This is preferable if the schema is correct and the query is wrong.
        *   **Option B - Modify Schema:** Update the table schema in `createSchema` to add the necessary `UNIQUE` or `PRIMARY KEY` constraint. This is appropriate if the schema is missing constraints assumed by the queries.

6.  **Data Migration Strategy (New):**
    *   If Option B from step 5 (schema change) is required:
        *   **Test Database:** Create a backup copy of the current database to test migration steps.
        *   **Migration Script:** Develop a small migration script using `cmd/migrate_db/main.go` or similar to:
            *   Add the necessary constraints without data loss
            *   Handle any data conflicts that might arise from the new constraint
        *   **Verification Queries:** Write queries to verify data integrity before and after the migration
        *   **Rollback Plan:** Include a rollback strategy in case the migration fails
        *   **Schema Version Tracking:** Consider adding schema version tracking if not already present

7.  **Isolated Test Case Development (New):**
    *   For persistent issues, create minimal test cases that target specific endpoints:
        *   **Article Scoring Test:** Directly test the `/api/llm/reanalyze/{id}` endpoint in isolation.
        *   **Manual Newman Request:** Extract the specific failing request from `unified_backend_tests.json` and run it individually with Newman.
        *   **Go Test Case:** Write a focused Go test that reproduces the SQL error condition in isolation from other API tests.
    *   Use server logs during these isolated tests to gather more specific error details.

8.  **Retest after SQL Fix:**
    *   Ensure the server is running with `NO_AUTO_ANALYZE=true`.
    *   Run `scripts/test.cmd essential` and then `scripts/test.cmd all`.

9.  **Documentation Updates (New):**
    *   **Environment Variable Documentation:** If `NO_AUTO_ANALYZE` is found to be critical:
        *   Update `README.md` to make its purpose clearer
        *   Add to `.env.example` with proper comments
        *   Consider documenting in `docs/codebase_documentation.md` under environment variables
    *   **SQL Schema Changes:** If schema is modified:
        *   Document the changes in `CHANGELOG.md`
        *   Update any data model diagrams or docs
        *   Add comments to the source code explaining constraints and their purpose
    *   **Testing Guide Updates:** Clarify in `docs/testing.md` if server must be started differently when running specific test suites

10.  **Further Investigation (if issues *still* persist):**
     *   Re-evaluate the `calculateCompositeScore` error: If the `ON CONFLICT` was the main cause of 500s, this calculation path (all zeros) might be acceptable, but logging should be changed from `ERROR` to `INFO` or `DEBUG` if it's an expected outcome for neutral articles.
     *   Verify `.env` file contents again (e.g., `OPENAI_API_KEY`).
     *   Simplify by running individual Newman requests manually against the server.

## 4. Success Criteria

*   The `essential` and `all` test suites pass consistently.
*   Server logs show no `SQLITE_BUSY` errors during test execution (confirming `NO_AUTO_ANALYZE=true` is effective for the server).
*   Server logs show no `SQL logic error: ON CONFLICT clause does not match...` errors (confirming SQL fix is effective).
*   Any schema changes are properly migrated without data loss.
*   Documentation is updated to reflect findings and changes.

## 5. Next Immediate Actions (Planned Execution Steps)

1.  Terminate `newsbalancer_server.exe` and any `go.exe` server processes.
2.  Run `scripts/test.cmd clean`.
3.  Verify `.env` and `.env.example` for `NO_AUTO_ANALYZE`.
4.  Start server: `$env:NO_AUTO_ANALYZE='true'; go run cmd/server/main.go` (in background).
5.  Wait ~15 seconds and check server startup logs for `NO_AUTO_ANALYZE` processing and absence of immediate errors.
6.  Run `scripts/test.cmd essential`.
7.  Analyze results (test logs and server logs), paying close attention to whether `SQLITE_BUSY` is gone and if the `ON CONFLICT` SQL error appears.
8.  Proceed to appropriate next steps based on results (SQL fix, documentation updates, etc.) 