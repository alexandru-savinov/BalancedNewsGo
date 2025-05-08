## Technical Recommendation: Explicit Error Handling for Complete LLM Analysis Failure

**Date:** 2023-10-27 (Updated: 2025-05-08)
**Author:** AI Assistant (Gemini)
**Status:** Integration Testing in Progress

**0. Current Implementation Status**

As of the latest update, the core code changes outlined in section 3 (Define Specific Error, Modify Score Calculation Logic, Update Caller Logic in ScoreManager) have been implemented. Unit tests for the `internal/llm` package, including `llm_test.go`, `composite_score_test.go`, `model_name_handling_test.go`, and `score_manager_test.go`, have been successfully updated and are passing. These tests verify the correct propagation and handling of `ErrAllPerspectivesInvalid` and the related `ErrAllScoresZeroConfidence` at the unit level.

Integration testing is underway following the plan in Section 7.

*   **Scenario 1 (API - `ErrAllPerspectivesInvalid`):**
    *   **Status:** Completed.
    *   **Debugging:** Initial tests failed due to:
        *   `math.NaN()` insertion errors (resolved by using `math.Inf(1)` for seeding).
        *   DB changes not visible between processes (resolved by removing `DROP TABLE` from `InitDB`).
        *   API validation error (resolved by adding content-type and empty body to POST request).
        *   SSE stream verification was inconclusive due to tool limitations.
    *   **Outcome:** Database verification confirmed that triggering `ErrAllPerspectivesInvalid` correctly prevents the `composite_score` from being updated to `0.0` in the `articles` table.

*   **Scenario 2 (API - `ErrAllScoresZeroConfidence`):**
    *   **Status:** Pending.

*   **Scenario 3 (Batch Scoring - `cmd/score_articles`):**
    *   **Status:** Pending.

*   **Scenario 4 (Validation - `cmd/validate_labels`):**
    *   **Status:** Pending.

**1. Context & Problem Statement**

The current system calculates a composite political bias score for news articles by aggregating results from multiple LLM analyses, grouped by political perspective (left, center, right), based on the configuration in `configs/composite_score_config.json`.

The configuration includes a `handle_invalid` setting. When set to `"ignore"`, the system disregards invalid scores (e.g., NaN, Infinity) returned by LLMs for a specific perspective. The final composite score is then calculated based only on the perspectives that provided *valid* scores.

A problematic edge case arises when *all* perspectives configured for analysis return invalid scores (or fail in a way that results in no valid score being assigned to any perspective). In this scenario, with `handle_invalid: "ignore"`, the `calculateCompositeScore` function (in `internal/llm/composite_score_fix.go`) currently returns a numerical score of `0.0`.

This behavior leads to ambiguity:

*   **Misleading Score:** A `0.0` score typically implies a "neutral" political bias. However, in this failure case, it signifies a *complete inability to analyze the article*, not neutrality.
*   **Data Corruption:** Storing this ambiguous `0.0` score in the `articles` table (`composite_score` column) via `ScoreManager.UpdateArticleScore` corrupts the data, making it impossible to distinguish between genuinely neutral articles and articles where analysis failed completely.
*   **Insufficient Signaling:** While the associated `confidence` score might be low in this scenario, it's not a direct or guaranteed signal of *total* analysis failure across all perspectives.

**2. Proposed Solution**

To address this ambiguity and improve system robustness, we propose modifying the system to explicitly signal a complete analysis failure using Go's standard error handling mechanism.

Instead of returning `0.0` when no valid perspective scores are available, the score calculation logic will return a specific, designated error: `ErrAllPerspectivesInvalid`. The calling component (`ScoreManager`) will detect this specific error and prevent the ambiguous `0.0` score from being persisted to the database, while still updating the task progress state to reflect the error.

**3. Implementation Details**

The implementation involves changes in three key areas:

**3.1. Define the Specific Error:**

A new sentinel error will be defined in the `internal/llm` package.

*   **File:** `internal/llm/errors.go`
*   **Change:** Add the following error definition:
    ```go
    package llm
    import "errors"

    // ... existing errors ...
    var ErrRateLimited = ErrBothLLMKeysRateLimited

    // ErrAllPerspectivesInvalid indicates that despite attempting analysis across
    // configured perspectives, no valid score could be obtained from any of them.
    var ErrAllPerspectivesInvalid = errors.New("failed to get valid scores from any LLM perspective")
    ```

**3.2. Modify Score Calculation Logic:**

The functions responsible for calculating the composite score need to return the new error when appropriate.

*   **File:** `internal/llm/composite_score_fix.go`
*   **Function:** `calculateCompositeScore(...)` (Internal helper)
*   **Change:** Modify the logic where the score is calculated based on `numberOfValidPerspectives`.
    ```go
    // Inside calculateCompositeScore function:
    // ... calculation of sumOfScores, numberOfValidPerspectives ...

    if numberOfValidPerspectives == 0 {
        // Previously returned 0.0 here. Now return the specific error.
        // The float return value is ignored when error is non-nil.
        return 0.0, ErrAllPerspectivesInvalid
    }

    // ... rest of the calculation logic (average/weighted) ...
    ```
*   **Function:** `ComputeCompositeScoreWithConfidenceFixed(...)` (Main function)
*   **Change:** Ensure this function correctly propagates `ErrAllPerspectivesInvalid` if received from `calculateCompositeScore`. When this specific error occurs, confidence calculation should return 0.0 alongside the error.
    *(Revised flow based on self-correction):*
    ```go
    // Inside ComputeCompositeScoreWithConfidenceFixed function:
    // ... setup ...
    compositeScore, calcErr := calculateCompositeScore(...)
    if calcErr != nil {
        // Return 0.0 for score and confidence, plus the specific error
        // This handles ErrAllPerspectivesInvalid and any other potential errors from calculation
        return 0.0, 0.0, calcErr
    }

    // Only calculate confidence if composite score calculation succeeded
    confidence, confErr := calculateConfidence(...)
    if confErr != nil {
       log.Printf("Warning: failed to calculate confidence: %v", confErr)
       confidence = 0.0 // Assign default confidence on calculation error
    }

    // Success case
    return compositeScore, confidence, nil
    ```

**3.3. Update Caller Logic (`ScoreManager`):**

The `ScoreManager` needs to explicitly check for and handle the new error.

*   **File:** `internal/llm/score_manager.go`
*   **Function:** `UpdateArticleScore(...)`
*   **Change:** Add error checking after the call to `sm.calculator.CalculateScore`.
    ```go
    // Inside UpdateArticleScore function:
    // ... (previous steps, including checkForAllZeroResponses) ...

    compositeScore, confidence, err := sm.calculator.CalculateScore(scores, cfg)
    if err != nil {
        // Check for the specific "all invalid" error
        if errors.Is(err, ErrAllPerspectivesInvalid) {
            log.Printf("[ERROR] ScoreManager: ArticleID %d: %v. Score will not be updated.", articleID, err)
            // Update progress to reflect the error state
            errorState := models.ProgressState{ /* ... fields indicating error ... */
                 Step:        ProgressStepError,
                 Message:     err.Error(),
                 Status:      ProgressStatusError,
                 Percent:     100, // Or appropriate percentage indicating completion with error
                 LastUpdated: time.Now().Unix(),
            }
            sm.SetProgress(articleID, &errorState)
            // IMPORTANT: Do NOT proceed to update the DB score. Return the error.
            return err
        } else {
            // Handle other, unexpected errors from CalculateScore
            log.Printf("[ERROR] ScoreManager: ArticleID %d: Unexpected error calculating score: %v. Score will not be updated.", articleID, err)
            // Update progress similarly
            errorState := models.ProgressState{ /* ... fields indicating generic error ... */
                 Step:        ProgressStepError,
                 Message:     fmt.Sprintf("Internal error calculating score: %v", err),
                 Status:      ProgressStatusError,
                 Percent:     100,
                 LastUpdated: time.Now().Unix(),
            }
            sm.SetProgress(articleID, &errorState)
            return err
        }
    }

    // --- If err is nil (Success case) ---
    // Proceed with database update as before
    log.Printf("[INFO] ScoreManager: ArticleID %d: Calculated score=%.4f, confidence=%.4f", articleID, compositeScore, confidence)
    if err := db.UpdateArticleScoreLLM(sm.db, articleID, compositeScore, confidence); err != nil {
        log.Printf("[ERROR] ScoreManager: ArticleID %d: Failed to update score in DB: %v", articleID, err)
        // Update progress with DB error
        dbErrorState := models.ProgressState{ /* ... */ }
        sm.SetProgress(articleID, &dbErrorState)
        return err // Return the DB error
    }

    // Invalidate cache and set success progress
    sm.InvalidateScoreCache(articleID)
    successState := models.ProgressState{ /* ... fields indicating success ... */
        Step:        ProgressStepComplete,
        Message:     "Analysis complete.",
        Status:      ProgressStatusSuccess,
        Percent:     100,
        FinalScore:  &compositeScore,
        LastUpdated: time.Now().Unix(),
    }
    sm.SetProgress(articleID, &successState)
    log.Printf("[INFO] ScoreManager: ArticleID %d: Score updated successfully.", articleID)

    return nil // Success
    ```

**4. Benefits**

*   **Eliminates Ambiguity:** Clearly distinguishes between a calculated neutral score (`0.0`) and a complete analysis failure.
*   **Improves Data Integrity:** Prevents misleading `0.0` scores from being persisted in the database when analysis fails.
*   **Increases Robustness:** Leverages Go's error handling for explicit failure management, forcing callers to handle this state.
*   **Enhances Clarity:** Makes the code's intent clearer – failure is signaled via error, not a magic number.
*   **Enables Better Feedback:** Allows the API layer to detect this specific failure and provide more informative error responses/status updates to clients.

**5. Potential Impact & Considerations**

*   **Database State:** Articles where analysis fails completely due to this condition will *not* have their `composite_score` and `confidence` fields updated in the `articles` table. They will retain their previous values (or `NULL` if never successfully scored). This is the desired outcome.
*   **API Behavior:**
    *   API endpoints triggering analysis (e.g., `/api/llm/reanalyze/:id`) should ideally check for this specific error returned by `ScoreManager` and potentially map it to a specific HTTP status (e.g., 503 Service Unavailable, 500 Internal Server Error with details).
    *   The SSE progress endpoint (`/api/llm/score-progress/:id`) will now correctly report an error status and message when this failure occurs, providing better real-time feedback.
*   **Monitoring:** Existing monitoring that might interpret `score == 0.0` needs review. Monitoring should focus on the explicit error logs/metrics or the progress status updates.

**5.1. Persisting Final Article Status in Database (NEW FINDING & PLAN)**

**Observation during Integration Testing:**
While `ScoreManager` updates an in-memory `ProgressState` (used for SSEs and logging), the final outcome of an article's scoring attempt (e.g., successful, failed due to `ErrAllPerspectivesInvalid`, failed due to `ErrAllScoresZeroConfidence`) is not currently persisted directly into the `articles.status` column in the database. The `articles.status` column (schema default 'pending') does not reflect these terminal states post-analysis. This means that while the `composite_score` is correctly not updated on specific errors, the reason for the lack of score (or the success state) isn't easily queryable from the primary `articles` table.

**Proposed Enhancement:**
To make the final processing state of each article explicitly and persistently available in the database, the following changes are proposed:

1.  **Define Article Status Constants:**
    *   Introduce standardized string constants for article processing statuses. These will clearly define the possible values for the `articles.status` column.
    *   Location: A new file (e.g., `internal/models/article_status.go`) or an existing relevant model definitions file.
    *   Example Constants:
        *   `ArticleStatusPending` ("pending")
        *   `ArticleStatusProcessing` ("processing")
        *   `ArticleStatusScored` ("scored")
        *   `ArticleStatusFailedAllInvalid` ("failed_all_invalid")
        *   `ArticleStatusFailedZeroConfidence` ("failed_zero_confidence")
        *   `ArticleStatusFailedError` ("failed_error") // For other generic scoring errors

2.  **Database Utility for Status Update:**
    *   Implement a new function in the `internal/db` package, e.g., `UpdateArticleStatus(dbOrTx sqlx.ExtContext, articleID int64, status string) error`.
    *   This function will execute an SQL statement: `UPDATE articles SET status = ? WHERE id = ?`.
    *   It should be added to the `DBOperations` interface if appropriate for wider use, or used directly by `ScoreManager`.

3.  **Modify `ScoreManager.UpdateArticleScore`:**
    *   In addition to setting the in-memory `ProgressState` and deciding whether to update `composite_score`, `ScoreManager` will now also call the new `db.UpdateArticleStatus` function to persist the final status to the `articles.status` column.
    *   Examples:
        *   Upon successful score calculation and database update of `composite_score`: set `status` to `ArticleStatusScored`.
        *   When `ErrAllPerspectivesInvalid` is handled: set `status` to `ArticleStatusFailedAllInvalid`.
        *   When `ErrAllScoresZeroConfidence` is handled: set `status` to `ArticleStatusFailedZeroConfidence`.
        *   For other errors (e.g., failure in `sm.calculator.CalculateScore` or failure in `db.UpdateArticleScoreLLM`): set `status` to `ArticleStatusFailedError`.

**Benefits of this Enhancement:**
*   **Persistent State:** The final outcome of scoring (success or specific failure type) becomes a persistent attribute of the article in the database.
*   **Improved Diagnosability:** Easier to query and understand why an article has a NULL score or what its last processing state was.
*   **Enhanced Batch Processing:** Batch jobs like `cmd/score_articles` can more reliably use the `articles.status` column to select articles that need processing (e.g., `WHERE status = 'pending'`) or re-processing.
*   **Clearer System Behavior:** Aligns the persistent database state more closely with the actual events occurring during analysis.

**6. Testing**

*   Add/update unit tests for `calculateCompositeScore` to assert that `ErrAllPerspectivesInvalid` is returned when `numberOfValidPerspectives` is 0.
*   Add/update unit tests for `ComputeCompositeScoreWithConfidenceFixed` to ensure it propagates the error correctly.
*   Add/update unit tests for `ScoreManager.UpdateArticleScore` to verify:
    *   It correctly identifies `ErrAllPerspectivesInvalid` using `errors.Is`.
    *   It does *not* call `db.UpdateArticleScoreLLM` when this error occurs.
    *   It *does* call `sm.SetProgress` with an appropriate error state.
    *   It returns the `ErrAllPerspectivesInvalid` error.
*   Integration tests should cover API scenarios that trigger scoring, ensuring the API handles the propagated error gracefully (e.g., returns a 5xx status code, SSE reports error).

**7. Integration Test Plan**

This section outlines the plan for integration testing the changes related to explicit error handling for complete LLM analysis failure.

**Overall Test Strategy:**

1.  **Isolate Dependencies:**
    *   **Database:** Utilize a dedicated test database or ensure a clean state before each test run (e.g., by deleting and recreating specific test articles and their scores).
    *   **LLM Responses:** Instead of live LLM calls, pre-seed the `llm_scores` table with data that specifically triggers `ErrAllPerspectivesInvalid` and `ErrAllScoresZeroConfidence`.
2.  **Configuration:** Ensure `configs/composite_score_config.json` is suitable (e.g., `handle_invalid: "ignore"` to reliably trigger `ErrAllPerspectivesInvalid` with `NaN` scores).
3.  **Verification:** Check API HTTP status codes/bodies, SSE progress messages, application logs, database state (scores not updated incorrectly), and output files from batch commands.

**Detailed Plan Per Scenario:**

**Phase 1: Setup Common Test Utilities & Data**

*   **Action 1.1: Database Reset Script/Function:**
    *   Create a script (SQL or Go function) to delete specific test articles (e.g., IDs 9001-9010), their associated `llm_scores`, `labels`, and relevant progress entries.
    *   *Purpose:* Ensure a clean slate for tests.
*   **Action 1.2: Test Article Insertion Function:**
    *   Create a Go helper to insert a basic article into `articles` and return its ID.
    *   *Purpose:* Easily create fresh articles.
*   **Action 1.3: `llm_scores` Seeding Functions:**
    *   **Function A (`seed_for_ErrAllPerspectivesInvalid`):** Takes `articleID`, inserts `db.LLMScore` records that are all "invalid" (e.g., `NaN` scores) to make `numberOfValidPerspectives == 0` (requires `handle_invalid: "ignore"` in config).
    *   **Function B (`seed_for_ErrAllScoresZeroConfidence`):** Takes `articleID`, inserts `db.LLMScore` records for non-ensemble models, each with metadata indicating zero confidence (e.g., `{"confidence": 0.0}`).
    *   **Function C (`seed_for_successful_score`):** Takes `articleID`, inserts `db.LLMScore` records leading to a normal, successful score calculation.
    *   *Purpose:* Precisely control input conditions.

**Scenario 1: API - `ErrAllPerspectivesInvalid`**

*   **Step 1.1 (Setup):** DB reset, insert test article (ID 9001), call `seed_for_ErrAllPerspectivesInvalid(9001)`. Ensure `handle_invalid: "ignore"` in config.
*   **Step 1.2 (Execution):** Start server. `POST /api/llm/reanalyze/9001`. Listen to `GET /api/llm/score-progress/9001`.
*   **Step 1.3 (Verification):**
    *   **API Response (`/reanalyze`):** Expected HTTP Status `500` (or mapped error). Body indicates failure.
    *   **Database (`articles` table):** `composite_score` for ID 9001 should be original value or `NULL`, not `0.0`.
    *   **SSE Progress (`/score-progress`):** Final message for ID 9001: `Status: "Error"`, `Message: "failed to get valid scores from any LLM perspective"`.
    *   **Server Logs:** Check for errors related to ID 9001 and `ErrAllPerspectivesInvalid`.
*   **Step 1.4 (Teardown):** Stop server. DB reset.

**Scenario 2: API - `ErrAllScoresZeroConfidence`**

*   **Step 2.1 (Setup):** DB reset, insert test article (ID 9002), call `seed_for_ErrAllScoresZeroConfidence(9002)`.
*   **Step 2.2 (Execution):** Start server. `POST /api/llm/reanalyze/9002`. Listen to `GET /api/llm/score-progress/9002`.
*   **Step 2.3 (Verification):**
    *   **API Response (`/reanalyze`):** Expected HTTP Status `500` (or mapped error). Body indicates failure.
    *   **Database (`articles` table):** `composite_score` for ID 9002 should be original or `NULL`, not `0.0`.
    *   **SSE Progress (`/score-progress`):** Final message for ID 9002: `Status: "Error"`, `Message: "all LLMs returned empty or zero-confidence responses"`.
    *   **Server Logs:** Check for errors related to ID 9002 and `ErrAllScoresZeroConfidence`.
*   **Step 2.4 (Teardown):** Stop server. DB reset.

**Scenario 3: Batch Scoring (`cmd/score_articles`)**

*   **Step 3.1 (Setup):** DB reset.
    *   Article A (ID 9003): Insert, `seed_for_successful_score(9003)`.
    *   Article B (ID 9004): Insert (initial score `NULL` or `-99.0`), `seed_for_ErrAllPerspectivesInvalid(9004)`.
    *   Article C (ID 9005): Insert (initial score `NULL` or `-99.0`), `seed_for_ErrAllScoresZeroConfidence(9005)`.
    *   Ensure `handle_invalid: "ignore"` in config.
*   **Step 3.2 (Execution):** `go run cmd/score_articles/main.go` (may need modification to target specific test IDs).
*   **Step 3.3 (Verification):**
    *   **Console Logs (`cmd/score_articles`):** Success for 9003; `ErrAllPerspectivesInvalid` for 9004; `ErrAllScoresZeroConfidence` for 9005.
    *   **Database (`articles` table):** 9003 updated; 9004 & 9005 original score (not `0.0`).
*   **Step 3.4 (Teardown):** DB reset.

**Scenario 4: Validation Against Labeled Data (`cmd/validate_labels`)**

*   **Step 4.1 (Setup):** DB reset.
    *   Article/Label D (ID 9006): Insert article & label. `seed_for_successful_score(9006)`.
    *   Article/Label E (ID 9007): Insert article & label. `seed_for_ErrAllPerspectivesInvalid(9007)`.
    *   Article/Label F (ID 9008): Insert article & label. `seed_for_ErrAllScoresZeroConfidence(9008)`.
*   **Step 4.2 (Execution):** `go run cmd/validate_labels/main.go`.
*   **Step 4.3 (Verification):**
    *   **Console Logs (`cmd/validate_labels`):** Normal for 9006; error for 9007 (`ErrAllPerspectivesInvalid`); error for 9008 (`ErrAllScoresZeroConfidence`).
    *   **Output JSON:** Entries for 9007 & 9008 should reflect analysis error, not `predicted_score: 0.0`.
    *   **Overall Metrics:** Unanalyzable cases (E,F) should not distort metrics.
    *   *Consideration:* `validate_labels` may need modification to distinctly report analysis errors from `EnsembleAnalyze`.
*   **Step 4.4 (Teardown):** DB reset.

This plan aims for reliable and insightful integration tests by controlling test conditions, primarily through pre-seeding `llm_scores`. 