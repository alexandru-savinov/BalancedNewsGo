# PR Proposal 1: Detailed Implementation Plan for API Endpoint Discrepancies (Enhanced Testing & Schema Considerations)

## Overview

This implementation plan addresses three critical discrepancies between the frontend JavaScript code and backend Go API endpoints. These issues cause certain frontend features to fail silently when attempting to communicate with non-existent or incorrectly named API endpoints. This revised plan includes more comprehensive testing strategies and incorporates database schema considerations identified during code review.

## 0. Prerequisites: Database Schema Modifications (Recommended)

Before or alongside the API changes, the following database schema modifications are recommended to ensure optimal performance and data consistency for the new endpoints. These changes should be added to `schema.sql` and applied to the database.

1.  **Index for Feedback Query:**
    *   **Rationale:** To optimize `FetchFeedbackByArticleID` which queries by `article_id` and orders by `created_at`.
    *   **SQL:** `CREATE INDEX idx_feedback_article_id_created_at ON feedback(article_id, created_at);`

2.  **Index for Distinct Sources Query:**
    *   **Rationale:** To optimize `FetchDistinctSources` which queries for distinct `source` values from the `articles` table.
    *   **SQL:** `CREATE INDEX idx_articles_source ON articles(source);`

3.  **Case-Insensitive Source Handling (Consideration):**
    *   **Rationale:** The `articles.source` column is currently case-sensitive. This means "SourceA" and "sourcea" would be treated as different sources. For a more user-friendly filter, consider making sources case-insensitive.
    *   **Option A (Schema Change - Recommended for consistency):** Modify `schema.sql` for the `articles` table:
        ```sql
        source TEXT NOT NULL COLLATE NOCASE,
        ```
    *   **Option B (Query Change):** Modify the `FetchDistinctSources` query if schema change is not immediately feasible:
        ```sql
        // query := `SELECT DISTINCT source COLLATE NOCASE FROM articles WHERE source IS NOT NULL AND source != '' ORDER BY source COLLATE NOCASE ASC`
        ```
    *   **Option C (Application-level Normalization):** Ensure sources are normalized to a consistent case (e.g., lowercase) during data ingestion. (Requires checking/modifying ingestion logic).
    *   **Decision for this PR:** This proposal will proceed assuming the current case-sensitive behavior for sources unless a decision is made to change it as part of this scope. The `FetchDistinctSources` query will remain as `SELECT DISTINCT source ...`. The impact on user experience should be noted if sources remain case-sensitive.

## 1. Standardize Ensemble Details Endpoint Path

### Current State

* **Frontend (`web/js/article.js`):** Makes calls to `/api/articles/${articleId}/ensemble-details`
* **Backend (`internal/api/api.go`):** Registers route as `/api/articles/:id/ensemble`
* **Swagger Definition:** Documents path as `/api/articles/{id}/ensemble`

### Implementation Steps

1. **Modify Backend Route Registration**
   * **File:** `internal/api/api.go` (line ~174)
   * **Current Code:**
     ```go
     // @Router /api/articles/{id}/ensemble [get]
     // @ID getArticleEnsemble
     router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(dbConn)))
     ```
   * **New Code:**
     ```go
     // @Router /api/articles/{id}/ensemble-details [get]
     // @ID getArticleEnsembleDetails // Suggestion: Update ID for clarity
     router.GET("/api/articles/:id/ensemble-details", SafeHandler(ensembleDetailsHandler(dbConn)))
     ```

2. **Update Swagger Documentation**
   * **File:** `internal/api/docs/swagger.json` (and `swagger.yaml`)
   * **Change:** Update the path from `"/api/articles/{id}/ensemble"` to `"/api/articles/{id}/ensemble-details"`. Also update the operationId if changed in step 1.
   * **Implementation Method:** After changing the code annotation above, run the Swagger generation command:
     ```bash
     go install github.com/swaggo/swag/cmd/swag@latest
     swag init -g cmd/server/main.go
     ```

3. **Verify Handler Logic**
   * No changes needed to the handler function `ensembleDetailsHandler` as it uses the `:id` parameter correctly.

4. **Enhanced Testing Plan**
   * **Unit Test (`internal/api/api_test.go` or specific `ensemble_test.go`):**
     * **Existing Tests:** Verify no regressions in existing unit tests for `ensembleDetailsHandler` due to path change (though handler logic is untouched, route registration change is key).
     * **Test Cases for Handler (if not already exhaustive):**
       * Test with a valid article ID that has ensemble details. Verify correct data and 200 OK.
       * Test with a valid article ID that does *not* have ensemble details (e.g., not yet processed). Verify appropriate response (e.g., 404 Not Found or empty data with 200 OK, depending on desired behavior).
       * Test with an invalid article ID format (e.g., non-numeric). Verify 400 Bad Request.
       * Test with a non-existent article ID. Verify 404 Not Found.
       * Mock database errors during fetching article or ensemble data and verify 500 Internal Server Error.
   * **Integration Test (e.g., Postman/Newman collection):**
     * **Setup:** Ensure the test database contains articles with and without ensemble details.
     * **Test Case 1 (Happy Path):**
       * Make a GET request to `/api/articles/{valid_id_with_details}/ensemble-details`.
       * Verify HTTP status 200 OK.
       * Verify `Content-Type` is `application/json`.
       * Verify the response body structure and data integrity (e.g., expected fields are present, data types are correct).
     * **Test Case 2 (Article without Ensemble Data):**
       * Make a GET request to `/api/articles/{valid_id_without_details}/ensemble-details`.
       * Verify the expected response (e.g., 404 or 200 with empty/null data).
     * **Test Case 3 (Non-Existent Article ID):**
       * Make a GET request to `/api/articles/99999999/ensemble-details` (an ID known not to exist).
       * Verify HTTP status 404 Not Found.
     * **Test Case 4 (Invalid Article ID Format):**
       * Make a GET request to `/api/articles/abc/ensemble-details`.
       * Verify HTTP status 400 Bad Request.
     * **Test Case 5 (Old Endpoint):**
       * Make a GET request to the old path `/api/articles/{valid_id_with_details}/ensemble`.
       * Verify HTTP status 404 Not Found (unless rollback plan for temporary dual support is active).
   * **Browser/Manual Test:**
     * Start the server: `NO_AUTO_ANALYZE=true go run cmd/server/main.go`
     * Navigate to an article detail page in the browser.
     * Use browser Developer Tools (Network tab) to confirm the request to `/api/articles/[id]/ensemble-details` is made.
     * Verify the request succeeds with a 200 status and the ensemble details section of the page is populated correctly.
     * Test with an article that might not have ensemble details yet to see how the UI handles it.
     * Check server logs for any errors related to this endpoint.

## 2. Implement GET Endpoint for Article Feedback

### Current State

* **Frontend (`web/js/article.js`):** Makes calls to `/api/articles/${articleId}/feedback`
* **Backend:** Only implements `POST /api/feedback` for submitting new feedback, but no endpoint to retrieve existing feedback for a *specific article* via GET.

### Implementation Steps

**0. Prerequisite: Ensure Database Index (see Section 0.1)**
   * Verify that `CREATE INDEX idx_feedback_article_id_created_at ON feedback(article_id, created_at);` has been applied.

1. **Add Database Query Function**
   * **File:** `internal/db/db.go`
   * **New Function:**
     ```go
     // FetchFeedbackByArticleID retrieves all feedback entries for a specific article
     func FetchFeedbackByArticleID(db *sqlx.DB, articleID int64) ([]Feedback, error) {
         var feedbackEntries []Feedback // Renamed for clarity if `Feedback` is also a struct name
         query := `SELECT id, article_id, user_id, feedback_text, category, ensemble_output_id, source, created_at
                  FROM feedback
                  WHERE article_id = ?
                  ORDER BY created_at DESC`

         err := db.Select(&feedbackEntries, query, articleID)
         if err != nil {
             // It's common for db.Select to return sql.ErrNoRows if no results are found.
             // This is not an application error, so handle it gracefully.
             if errors.Is(err, sql.ErrNoRows) {
                 return []Feedback{}, nil // Return empty slice, not an error
             }
             return nil, handleError(err, "error fetching feedback by article ID")
         }

         return feedbackEntries, nil
     }
     ```

2. **Create New API Handler**
   * **File:** `internal/api/api.go`
   * **New Handler Function:**
     ```go
     // @Summary Get article feedback
     // @Description Get all feedback submitted for a specific article
     // @Tags Feedback
     // @Accept json
     // @Produce json
     // @Param article_id query integer true "Article ID" mininum(1)
     // @Success 200 {object} StandardResponse{data=[]db.Feedback} "Successfully retrieved feedback"
     // @Failure 400 {object} ErrorResponse "Invalid or missing article ID"
     // @Failure 404 {object} ErrorResponse "Article not found"
     // @Failure 500 {object} ErrorResponse "Server error retrieving feedback"
     // @Router /api/feedback [get] // Path chosen to be distinct from POST /api/feedback by HTTP method
     // @ID getArticleFeedback
     func getFeedbackHandler(dbConn *sqlx.DB) gin.HandlerFunc {
         return func(c *gin.Context) {
             start := time.Now()
             articleIDStr := c.Query("article_id")
             if articleIDStr == "" {
                 RespondError(c, NewAppError(ErrValidation, "Missing required query parameter: article_id"))
                 return
             }

             articleID, err := strconv.ParseInt(articleIDStr, 10, 64)
             if err != nil || articleID < 1 {
                 RespondError(c, NewAppError(ErrValidation, "Invalid article ID format or value"))
                 return
             }

             // Check if article exists to give a 404 if not
             // Assuming db.FetchArticleByID exists and returns db.ErrArticleNotFound
             _, err = db.FetchArticleByID(dbConn, articleID)
             if err != nil {
                 if errors.Is(err, db.ErrArticleNotFound) { // Use your project's specific error for not found
                     RespondError(c, NewAppError(ErrNotFound, "Article not found"))
                     return
                 }
                 RespondError(c, WrapError(err, ErrInternal, "Failed to verify article existence"))
                 return
             }

             feedbackItems, err := db.FetchFeedbackByArticleID(dbConn, articleID)
             if err != nil {
                 // The DB function now handles sql.ErrNoRows, so this should be an actual error
                 RespondError(c, WrapError(err, ErrInternal, "Failed to fetch feedback"))
                 return
             }

             // db.FetchFeedbackByArticleID now returns an empty slice if no feedback,
             // so this explicit check might no longer be needed if feedbackItems is initialized.
             // if feedbackItems == nil {
             //     feedbackItems = []db.Feedback{}
             // }

             RespondSuccess(c, feedbackItems)
             LogPerformance("getFeedbackHandler", start)
         }
     }
     ```

3. **Register the New Route**
   * **File:** `internal/api/api.go` (in the `RegisterRoutes` function)
   * **Add (e.g., near other feedback-related routes or group by entity):**
     ```go
     // @Summary Get article feedback
     // @Description Get all feedback submitted for a specific article
     // @Tags Feedback
     // @Accept json
     // @Produce json
     // @Param article_id query integer true "Article ID"
     // @Success 200 {object} StandardResponse{data=[]db.Feedback}
     // @Failure 400 {object} ErrorResponse "Invalid article ID"
     // @Failure 404 {object} ErrorResponse "Article not found"
     // @Failure 500 {object} ErrorResponse "Server error"
     // @Router /api/feedback [get]
     // @ID getArticleFeedback
     router.GET("/api/feedback", SafeHandler(getFeedbackHandler(dbConn)))
     ```

4. **Update Frontend Code**
   * **File:** `web/js/article.js`
   * **Current Code (in `loadFeedbackStatus` function or similar):**
     ```javascript
     const response = await fetch(`/api/articles/${articleId}/feedback`);
     ```
   * **New Code:**
     ```javascript
     const response = await fetch(`/api/feedback?article_id=${articleId}`);
     ```

5. **Enhanced Testing Plan**
   * **Unit Test (`internal/api/feedback_test.go`):**
     * **Setup:** Mock the `*sqlx.DB` connection and `db.FetchFeedbackByArticleID` and `db.FetchArticleByID` functions.
     * **Test Case 1 (Happy Path - Feedback Exists):**
       * Mock `db.FetchArticleByID` to return a dummy article.
       * Mock `db.FetchFeedbackByArticleID` to return a slice of `db.Feedback` objects.
       * Call the handler with a valid `article_id`.
       * Verify 200 OK, correct `Content-Type`, and the response body matches the mocked feedback.
     * **Test Case 2 (Happy Path - No Feedback):**
       * Mock `db.FetchArticleByID` to return a dummy article.
       * Mock `db.FetchFeedbackByArticleID` to return an empty slice (`[]db.Feedback{}`).
       * Call the handler with a valid `article_id`.
       * Verify 200 OK and the response body is an empty JSON array (`[]`).
     * **Test Case 3 (Missing `article_id` Query Parameter):**
       * Call handler without `article_id`. Verify 400 Bad Request and appropriate error message.
     * **Test Case 4 (Invalid `article_id` Format - e.g., "abc"):**
       * Call handler with `article_id=abc`. Verify 400 Bad Request and error message.
     * **Test Case 5 (Invalid `article_id` Value - e.g., "0" or "-1"):**
       * Call handler with `article_id=0`. Verify 400 Bad Request and error message.
     * **Test Case 6 (Article Not Found):**
       * Mock `db.FetchArticleByID` to return `db.ErrArticleNotFound` (or your project's equivalent).
       * Call handler with a valid `article_id`. Verify 404 Not Found.
     * **Test Case 7 (DB Error on Fetching Article):**
       * Mock `db.FetchArticleByID` to return a generic error.
       * Call handler. Verify 500 Internal Server Error.
     * **Test Case 8 (DB Error on Fetching Feedback):**
       * Mock `db.FetchArticleByID` to return a dummy article.
       * Mock `db.FetchFeedbackByArticleID` to return an error.
       * Call handler. Verify 500 Internal Server Error.
   * **Integration Test (Postman/Newman):**
     * **Setup:** Ensure test DB has articles: one with multiple feedback items, one with one feedback item, one with no feedback, and ensure some article IDs don't exist.
     * **Test Case 1 (Article with Multiple Feedback):**
       * `GET /api/feedback?article_id={id_with_multiple_feedback}`.
       * Verify 200 OK, `Content-Type`, response is an array of feedback objects, ordered by `created_at DESC`.
     * **Test Case 2 (Article with Single Feedback):**
       * `GET /api/feedback?article_id={id_with_one_feedback}`.
       * Verify 200 OK, response is an array with one feedback object.
     * **Test Case 3 (Article with No Feedback):**
       * `GET /api/feedback?article_id={id_with_no_feedback}`.
       * Verify 200 OK, response is an empty array `[]`.
     * **Test Case 4 (Non-Existent Article ID):**
       * `GET /api/feedback?article_id=9999999`. Verify 404 Not Found.
     * **Test Case 5 (Invalid `article_id` - text):**
       * `GET /api/feedback?article_id=sometext`. Verify 400 Bad Request.
     * **Test Case 6 (Missing `article_id`):**
       * `GET /api/feedback`. Verify 400 Bad Request.
     * **Test Case 7 (Invalid `article_id` - zero):**
       * `GET /api/feedback?article_id=0`. Verify 400 Bad Request.
   * **Browser/Manual Test:**
     * Start the server: `NO_AUTO_ANALYZE=true go run cmd/server/main.go`
     * Manually POST some feedback for a test article using an API tool or existing UI if POST works.
     * Navigate to the article detail page for that article.
     * Use Developer Tools to verify the `GET /api/feedback?article_id={id}` call.
     * Confirm the feedback is displayed correctly on the page.
     * Test with an article known to have no feedback; verify UI handles empty state gracefully.
     * Check server logs for errors.

## 3. Implement GET Endpoint for News Sources

### Current State

* **Frontend (`web/js/list.js`):** Makes calls to `/api/sources` to populate source filter dropdown.
* **Backend:** No endpoint exists for retrieving a list of unique news sources.

### Implementation Steps

**0. Prerequisite: Ensure Database Index (see Section 0.2) and Consider Source Case Sensitivity (Section 0.3)**
   * Verify that `CREATE INDEX idx_articles_source ON articles(source);` has been applied.
   * Acknowledge the decision on handling case sensitivity for sources. If sources remain case-sensitive (default), the frontend filter will show "CNN" and "cnn" as separate items if both exist in the database.

1. **Add Database Query Function**
   * **File:** `internal/db/db.go`
   * **New Function:**
     ```go
     // FetchDistinctSources retrieves all unique, non-empty sources from the articles table, ordered alphabetically.
     // Note: This query is case-sensitive by default in SQLite. See Section 0.3 for considerations.
     func FetchDistinctSources(db *sqlx.DB) ([]string, error) {
         var sources []string
         // Ensure filtering out NULL or empty strings, and order for consistent output
         query := `SELECT DISTINCT source FROM articles WHERE source IS NOT NULL AND source != '' ORDER BY source ASC`

         err := db.Select(&sources, query)
         if err != nil {
             if errors.Is(err, sql.ErrNoRows) { // If no articles or no sources found
                 return []string{}, nil // Return empty slice
             }
             return nil, handleError(err, "error fetching distinct sources")
         }

         return sources, nil
     }
     ```

2. **Create New API Handler**
   * **File:** `internal/api/api.go`
   * **New Handler Function:**
     ```go
     // @Summary Get news sources
     // @Description Get a list of all unique news sources from articles in the system
     // @Tags Articles
     // @Accept json
     // @Produce json
     // @Success 200 {object} StandardResponse{data=[]string} "Successfully retrieved news sources"
     // @Failure 500 {object} ErrorResponse "Server error retrieving sources"
     // @Router /api/sources [get]
     // @ID getNewsSources
     func getSourcesHandler(dbConn *sqlx.DB) gin.HandlerFunc {
         return func(c *gin.Context) {
             start := time.Now()

             cacheKey := "news_sources_list" // Make cache key specific
             // Assuming articlesCache and articlesCacheLock are defined globally/accessibly in api.go
             articlesCacheLock.RLock()
             if cachedData, found := articlesCache.Get(cacheKey); found {
                 articlesCacheLock.RUnlock()
                 if sources, ok := cachedData.([]string); ok {
                     RespondSuccess(c, sources)
                     LogPerformance("getSourcesHandler (cache hit)", start)
                     return
                 }
                 // If type assertion fails, something is wrong with cache, proceed to fetch
                 LogError(c, errors.New("cached news_sources data has unexpected type"), "getSourcesHandler cache type error")
             }
             articlesCacheLock.RUnlock() // Ensure RUnlock is called if not returned early

             dbSources, err := db.FetchDistinctSources(dbConn)
             if err != nil {
                 RespondError(c, WrapError(err, ErrInternal, "Failed to fetch news sources from database"))
                 // LogError already part of WrapError or RespondError typically
                 return
             }

             // db.FetchDistinctSources now returns empty slice if none found
             // if dbSources == nil {
             //     dbSources = []string{}
             // }

             articlesCacheLock.Lock()
             articlesCache.Set(cacheKey, dbSources, 5*time.Minute) // Configurable cache duration
             articlesCacheLock.Unlock()

             RespondSuccess(c, dbSources)
             LogPerformance("getSourcesHandler (db fetch)", start)
         }
     }
     ```

3. **Register the New Route**
   * **File:** `internal/api/api.go` (in the `RegisterRoutes` function)
   * **Add (e.g., grouped with other article-related or general utility routes):**
     ```go
     // @Summary Get news sources
     // @Description Get a list of all unique news sources in the system
     // @Tags Articles
     // @Accept json
     // @Produce json
     // @Success 200 {object} StandardResponse{data=[]string}
     // @Failure 500 {object} ErrorResponse "Server error"
     // @Router /api/sources [get]
     // @ID getNewsSources
     router.GET("/api/sources", SafeHandler(getSourcesHandler(dbConn)))
     ```

4. **Enhanced Testing Plan**
   * **Unit Test (`internal/api/sources_test.go` or `api_test.go`):**
     * **Setup:** Mock `*sqlx.DB` and `db.FetchDistinctSources`. Mock cache interactions if possible/needed or test cache logic separately.
     * **Test Case 1 (Happy Path - Sources Exist):**
       * Mock `db.FetchDistinctSources` to return `[]string{"Source A", "Source B"}`.
       * Call handler. Verify 200 OK, `Content-Type`, and response body `["Source A", "Source B"]`.
       * Verify cache is populated with the result.
     * **Test Case 2 (Happy Path - No Sources):**
       * Mock `db.FetchDistinctSources` to return `[]string{}`.
       * Call handler. Verify 200 OK and response body `[]`.
       * Verify cache is populated with an empty list.
     * **Test Case 3 (DB Error):**
       * Mock `db.FetchDistinctSources` to return an error.
       * Call handler. Verify 500 Internal Server Error.
       * Verify cache is not populated with error.
     * **Test Case 4 (Cache Hit):**
       * Pre-populate cache with `[]string{"Cached Source"}`.
       * Mock `db.FetchDistinctSources` to return a *different* list (to ensure DB isn't called).
       * Call handler. Verify 200 OK and response `["Cached Source"]`.
       * Ensure `db.FetchDistinctSources` was not called.
     * **Test Case 5 (Cache Miss then Populate):**
       * Ensure cache is empty for the key.
       * Mock `db.FetchDistinctSources` to return `[]string{"DB Source"}`.
       * Call handler. Verify 200 OK and response `["DB Source"]`.
       * Call handler again. Verify `db.FetchDistinctSources` not called on the second time (cache hit).
   * **Integration Test (Postman/Newman):**
     * **Setup:** Ensure test DB has articles with a variety of sources, including some articles with NULL or empty string sources, and some duplicate sources.
     * **Test Case 1 (Sources Exist):**
       * `GET /api/sources`. Verify 200 OK, `Content-Type`.
       * Response body is a JSON array of unique, non-empty strings, sorted alphabetically.
       * Example: if DB has "CNN", "BBC", "cnn", "", NULL, "Fox News", "BBC", expect `["BBC", "CNN", "Fox News"]` (assuming case sensitivity in DB query or Go processing, adjust if source normalization happens, or if `COLLATE NOCASE` is used. Without schema/query changes, expect `["BBC", "CNN", "Fox News", "cnn"]` if "cnn" also exists and ordering places it there.).
   * **Browser/Manual Test:**
     * Start the server: `NO_AUTO_ANALYZE=true go run cmd/server/main.go`
     * Navigate to the article list page (`/web/list.html` or similar).
     * Check if the "Filter by source" dropdown is populated.
     * Use Developer Tools (Network tab) to confirm the `GET /api/sources` call, status, and response data.
     * Verify the sources in the dropdown match unique sources from the articles table.
     * If possible, add an article with a new source via an admin tool or script, then after cache expiry (or server restart if cache is in-memory and not persisted), refresh the list page and see if the new source appears in the filter.
     * Check server logs for errors.

## Overall Verification and Validation (Enhanced)

After implementing all three fixes:

1. **Apply Database Schema Changes (if not already done as prerequisites):**
    *   Ensure the new indexes (`idx_feedback_article_id_created_at`, `idx_articles_source`) are created.
    *   Confirm the decision and implementation regarding `articles.source` case sensitivity.

2. **Run All Go Unit Tests:**
   * Ensure comprehensive coverage for new and modified handlers and DB functions.
   * Command: `$env:NO_AUTO_ANALYZE='true'; go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out`
   * Review coverage report, aiming for high coverage on new/changed logic.

3. **Run All API Integration Tests (Postman/Newman):**
   * Execute the full test suite.
   * Command: `scripts/test.cmd api` (or equivalent)
   * Verify all tests pass, including new ones for these endpoints and existing ones to catch regressions.
   * Check test run reports for any failures or warnings.

4. **Comprehensive Frontend Integration Testing:**
   * Start the server: `NO_AUTO_ANALYZE=true go run cmd/server/main.go`
   * **Scenario 1 (Ensemble Details):**
     * Open multiple articles: one with full ensemble data, one without.
     * Verify ensemble details load correctly (or UI shows appropriate message for missing data) via `/api/articles/{id}/ensemble-details`.
     * Check browser console and network tab for errors.
   * **Scenario 2 (Feedback):**
     * For an article, submit new feedback using the UI.
     * Refresh the page/section. Verify the `GET /api/feedback?article_id={id}` call happens and displays the newly submitted feedback along with any existing feedback, sorted correctly.
     * Test on an article with no prior feedback.
     * Check browser console and network tab for errors.
   * **Scenario 3 (News Sources Filter):**
     * Go to the article list page. Verify the source filter dropdown is populated correctly by `/api/sources`.
     * Select a source and verify the article list filters as expected (this tests frontend logic using the data).
     * Check browser console and network tab for errors.
   * **Scenario 4 (General UI Stability):**
     * Navigate through various parts of the application to ensure no unrelated areas have been impacted.

5. **Verify Swagger Documentation:**
   * Generate updated Swagger docs: `swag init -g cmd/server/main.go`
   * Access Swagger UI at `http://localhost:8080/swagger/index.html`.
   * Verify:
     * `/api/articles/{id}/ensemble-details` path is correct, operationId matches, and schema is accurate.
     * `GET /api/feedback` (with `article_id` query param) is documented with correct parameters, responses, and schemas.
     * `GET /api/sources` is documented with correct responses and schemas.
     * All descriptions and summaries are clear.

6. **Code Review and Static Analysis:**
   * Run `golangci-lint run` (or your project's linter) and address any issues.
   * Conduct a peer code review focusing on:
     * Correctness of logic in new handlers and DB functions.
     * Proper error handling and use of `apperrors`.
     * Adherence to coding standards.
     * Efficiency of DB queries (especially considering the new indexes).
     * Correctness of cache implementation (keying, locking, expiry).
     * Clarity and completeness of Swagger annotations.

7. **Check Server Logs:**
   * Throughout all testing phases, monitor server logs for any unexpected errors, warnings, or excessive logging.

8. **Update Related Documentation:**
    *   Ensure `docs/plans/potential_improvements.md` is updated to reflect the correct path for the ensemble details endpoint if it was listed there.

## Rollback Plan

If issues arise after deployment:

1. **Ensemble Endpoint**
   * Temporarily support both endpoints until frontend is updated:
     ```go
     // Support both old and new paths
     router.GET("/api/articles/:id/ensemble", SafeHandler(ensembleDetailsHandler(dbConn))) // Old
     router.GET("/api/articles/:id/ensemble-details", SafeHandler(ensembleDetailsHandler(dbConn))) // New
     ```
   * The frontend will continue to use the new path; the old path is for graceful transition if an older client exists or quick server-side revert is needed.

2. **Feedback and Sources Endpoints**
   * These are new endpoints. Rollback involves:
     * Commenting out or removing the route registration in `internal/api/api.go`.
     * Reverting the frontend JavaScript changes in `web/js/article.js` and `web/js/list.js`.
     * The new handler functions and DB functions can remain (they won't be called) or be reverted if desired for a cleaner state.

## Impact Assessment

**Code Changes:**
* `internal/api/api.go`: ~45-50 lines added/modified (new handlers, route registrations)
* `internal/db/db.go`: ~35-40 lines added (new DB functions, minor logic adjustments)
* `web/js/article.js`: ~1-3 lines modified
* `internal/api/docs/swagger.json` & `swagger.yaml`: Generated files, changes based on annotations.

**Test Changes:**
* New unit test files/additions: e.g., `internal/api/feedback_test.go`, `internal/api/sources_test.go`, or additions to `api_test.go`. Estimated ~150-250 lines for comprehensive unit tests.
* Modified/New integration tests (Postman/Newman): ~5-10 new test cases added/modified across the collections.

**Risk Assessment:**
*   **Low-Medium Risk**: Changes are additive for two endpoints and a path correction for one. Core functionalities are largely untouched directly. Database performance for new endpoints is a consideration, mitigated by adding recommended indexes.
*   **Mitigation**: Implementation of recommended schema changes (indexes), enhanced comprehensive testing (unit, integration, frontend E2E), clear rollback plan, and careful code review. Consideration of source data normalization/case handling for user experience.

**Schema Changes:**
*   `schema.sql`: Addition of 2 new indexes. Potential modification to `articles.source` column for `COLLATE NOCASE`.

</rewritten_file>
