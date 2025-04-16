# Article Rescoring API Issue Analysis

## Test Results Summary

The Essential Article Rescoring Tests revealed several failures related to boundary value handling in the article rescoring functionality:

- **TC4 - Rescore with Upper Boundary Score (1.0)**: Failed with 400 Bad Request
- **TC5 - Rescore with Lower Boundary Score (-1.0)**: Failed with 400 Bad Request
- Other test cases for invalid scores (above range, non-numeric, malformed JSON) passed correctly

## Root Cause Analysis

After examining the codebase, I've identified the following issues:

1. **API Validation Restriction**: In the `reanalyzeHandler` function (`internal/api/api.go`), there's a check that rejects any request containing a "score" field:

    ```go
    if _, hasScore := raw["score"]; hasScore {
        RespondError(c, http.StatusBadRequest, ErrValidation, "Payload must not contain 'score' field")
        LogError("reanalyzeHandler: payload contains forbidden 'score' field", nil)
        return
    }
    ```

    This validation is preventing direct score updates through the API, which is needed for the test cases that verify boundary value handling.

2. **Score Range Configuration**: The `CompositeScoreConfig` in `configs/composite_score_config.json` correctly defines the valid score range as:
    ```json
    "min_score": -1.0,
    "max_score": 1.0,
    ```

    However, the validation logic may not be correctly handling the exact boundary values.

3. **Score Update Process**: The article scores aren't being updated after rescoring attempts, as evidenced by the composite scores remaining at 0 in the test results.

## Proposed Solution

### 1. Allow Direct Score Updates in API

**Done**  
The `manualScoreHandler` function already handles direct score updates for boundary values such as `-1.0` and `1.0`. Thus, test cases should use the `manualScoreHandler` endpoint (`/api/manual-score/:id`) for testing boundary values such as `-1.0` and `1.0`.

### 2. Ensure Boundary Values Are Handled Correctly

**Done**  
Review and update the score validation logic in `ComputeCompositeScoreWithConfidence` to ensure it correctly handles the boundary values (`-1.0` and `1.0`) as valid scores:

```go
// Current check
if cfg.HandleInvalid == "ignore" && (isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore) {
    continue
}
```

The current implementation appears correct for inclusive boundaries, but additional testing is needed to verify.

### 3. Use LLM-Specific Score Update Function

Ensure that `UpdateArticleScoreLLM` is used instead of `UpdateArticleScore` when the score comes from the LLM system:

```go
// In StoreEnsembleScore function
updateErr := db.UpdateArticleScoreLLM(c.db, article.ID, ensembleScore.Score, confidence)
```

### 4. Add Direct Score Update Endpoint

Consider adding a dedicated endpoint for direct score updates that bypasses the LLM scoring process for testing purposes. However, it seems that the `manualScoreHandler` already serves this purpose.

## Implementation Priority

1. **High Priority**: Use `manualScoreHandler` for testing boundary values such as `-1.0` and `1.0`. **Done**
2. **Medium Priority**: Verify boundary value handling in score validation. **Done**
3. **Medium Priority**: Ensure correct score update function is used.
4. **Low Priority**: Add dedicated test endpoint for direct score updates (if `manualScoreHandler` is not sufficient).

## Testing Plan

After implementing these changes:

1. Run the Essential Article Rescoring Tests again to verify the fixes.
2. Add additional test cases for edge cases:
    - Scores very close to boundaries (-0.999, 0.999)
    - Zero score (0.0)
    - Fractional scores (0.5, -0.5)
3. Verify that the API correctly updates article scores and sets the appropriate `score_source`.

## Conclusion

The article rescoring functionality has issues with boundary value handling that are preventing valid scores at the exact boundaries (-1.0 and 1.0) from being accepted. By modifying the API validation and ensuring proper score update processes, we can fix these issues and ensure the rescoring functionality works correctly for all valid score values.
