## Technical Recommendation: Explicit Error Handling for Complete LLM Analysis Failure

**Date:** 2023-10-27
**Author:** AI Assistant (Gemini)
**Status:** Proposed

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

**6. Testing**

*   Add/update unit tests for `calculateCompositeScore` to assert that `ErrAllPerspectivesInvalid` is returned when `numberOfValidPerspectives` is 0.
*   Add/update unit tests for `ComputeCompositeScoreWithConfidenceFixed` to ensure it propagates the error correctly.
*   Add/update unit tests for `ScoreManager.UpdateArticleScore` to verify:
    *   It correctly identifies `ErrAllPerspectivesInvalid` using `errors.Is`.
    *   It does *not* call `db.UpdateArticleScoreLLM` when this error occurs.
    *   It *does* call `sm.SetProgress` with an appropriate error state.
    *   It returns the `ErrAllPerspectivesInvalid` error.
*   Integration tests should cover API scenarios that trigger scoring, ensuring the API handles the propagated error gracefully (e.g., returns a 5xx status code, SSE reports error). 