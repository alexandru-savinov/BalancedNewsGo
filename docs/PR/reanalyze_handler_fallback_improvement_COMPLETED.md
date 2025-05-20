# Improving Fallback Mechanism in `reanalyzeHandler`

**Date:** 2025-05-17
**Author:** AI Assistant
**Status:** Proposed

## 1. Summary

The current `reanalyzeHandler` in `internal/api/api.go` performs a health check for LLM models before queueing a full article re-analysis. This health check is too strict: if a model fails for any reason other than a specific rate limit error (`llm.ErrBothLLMKeysRateLimited`), the handler immediately stops checking other configured models and reports an error. This prevents the system from attempting re-analysis even if other models in the ensemble are healthy, undermining the robustness of the ensemble approach.

This document proposes a modification to make the health check in `reanalyzeHandler` more resilient by allowing it to attempt all configured models before concluding that no working model is available.

## 2. In-Depth Analysis & Root Cause

### 2.1. Current Workflow

1.  **User Action & Initial Log:** A user triggers re-analysis via `POST /api/llm/reanalyze/:id`.
2.  **`reanalyzeHandler` Health Check (`internal/api/api.go`):**
    *   Iterates models from `composite_score_config.json`.
    *   For each model, calls `llmClient.ScoreWithModel()` with a short timeout (2 seconds).
    *   **Critical Flaw:** If `ScoreWithModel()` returns an error that is *not* `llm.ErrBothLLMKeysRateLimited` (e.g., `context deadline exceeded`), the health check loop **breaks immediately**.
    *   If no `workingModel` is found, `reanalyzeHandler` returns an error, and the more comprehensive `llmClient.ReanalyzeArticle()` is **never called**.

3.  **`llmClient.ScoreWithModel()` & `HTTPLLMService.ScoreContent()`:**
    *   `ScoreWithModel` calls `llmService.ScoreContent()`.
    *   `HTTPLLMService.ScoreContent()` (`internal/llm/service_http.go`) has its own fallbacks (primary/backup keys, trying *other models if current model is rate-limited on both keys*).
    *   If a non-rate-limit error (like a timeout) occurs for a model, `ScoreContent` propagates this error up.

4.  **The Disconnect:**
    *   When a model (e.g., `meta-llama/llama-4-maverick`) times out during the `reanalyzeHandler`'s health check:
        *   `ScoreContent` returns the timeout error.
        *   `reanalyzeHandler` receives this timeout. Since it's not `ErrBothLLMKeysRateLimited`, its health check loop breaks.
    *   **Root Cause:** The `reanalyzeHandler`'s health check is too quick to abandon finding a working model if initial attempts fail with errors like timeouts. It doesn't try all available models in such cases.

### 2.2. Intended vs. Actual Behavior

*   **Intended:** The system should leverage its ensemble configuration to provide resilience. If one LLM model is down, others should be used.
*   **Actual (for re-analysis initiation):** The initial health check in `reanalyzeHandler` can prematurely halt the process if the first-tried models have issues other than specific rate limit conditions. The more robust fallback logic in `llmClient.ReanalyzeArticle()` (which tries all models sequentially) is not reached.

## 3. Proposed Fix

Modify the health check loop in `reanalyzeHandler` (`internal/api/api.go`) to try all configured models before giving up. It should only determine "no working models found" if every model in the configuration fails the health check.

### 3.1. Conceptual Code Change (Illustrative)

```go
// Located in internal/api/api.go, within reanalyzeHandler

// ... (previous code in handler) ...

        } else { // This 'else' corresponds to not having NO_AUTO_ANALYZE=true
            originalTimeout := llmClient.GetHTTPLLMTimeout() // Requires GetHTTPLLMTimeout() to be added to LLMClient (see section 3.3)
            healthCheckTimeout := 2 * time.Second           // Keep short timeout for individual health checks
            if os.Getenv("HEALTH_CHECK_TIMEOUT_SECONDS") != "" { // Allow override for testing
                if s, err := strconv.Atoi(os.Getenv("HEALTH_CHECK_TIMEOUT_SECONDS")); err == nil && s > 0 {
                    healthCheckTimeout = time.Duration(s) * time.Second
                }
            }
            llmClient.SetHTTPLLMTimeout(healthCheckTimeout) 

            var lastHealthCheckError error 

            for _, modelConfig := range cfg.Models {
                log.Printf("[reanalyzeHandler %d] Health checking model: %s", articleID, modelConfig.ModelName)
                // 'article' object is already fetched in reanalyzeHandler; ScoreWithModel expects this object.
                _, healthCheckErr := llmClient.ScoreWithModel(article, modelConfig.ModelName) 
                
                if healthCheckErr == nil {
                    workingModel = modelConfig.ModelName
                    lastHealthCheckError = nil 
                    log.Printf("[reanalyzeHandler %d] Health check PASSED for model: %s", articleID, workingModel)
                    break 
                }
                
                log.Printf("[reanalyzeHandler %d] Health check FAILED for model %s: %v. Trying next model.", articleID, modelConfig.ModelName, healthCheckErr)
                lastHealthCheckError = healthCheckErr 
            }
            llmClient.SetHTTPLLMTimeout(originalTimeout) // Restore original timeout

            if workingModel == "" { // All models failed health check
                healthErr = lastHealthCheckError 
                if healthErr == nil { 
                    healthErr = apperrors.New("All models failed health check", "llm_service_unavailable")
                }
            }
        }

        if workingModel == "" {
            log.Printf("[reanalyzeHandler %d] No working models found after checking all configured models. Last error: %v", articleID, healthErr)
            // Ensure healthErr is an AppError or wrapped appropriately before RespondError
            if healthErr != nil {
                 // Ensure it's an AppError; if not, wrap it.
                appErr, ok := healthErr.(*apperrors.AppError)
                if !ok {
                    // Determine appropriate app error type, e.g., ErrLLMService or ErrInternal
                    // For a timeout, ErrLLMService or a specific timeout error code might be best.
                    // Example: Using ErrLLMService.WithDetail(...)
                    respondErr := apperrors.NewAppError(apperrors.ErrLLMService, fmt.Sprintf("All models failed health check. Last error: %v", healthErr.Error()))
                    RespondError(c, respondErr)
                } else {
                    RespondError(c, appErr)
                }
            } else {
                 RespondError(c, apperrors.NewAppError(apperrors.ErrLLMService, "No working models found and no specific error recorded."))
            }
            return
        }
// ... (rest of the function) ...
```

### 3.2. Explanation of Changes

1.  **`lastHealthCheckError`:** Introduced to store the error from the last model attempted during the health check.
2.  **Loop Modification:** The condition `if !errors.Is(healthErr, llm.ErrBothLLMKeysRateLimited) { break }` is removed. The loop now continues to the next model if an error occurs, only breaking if a model passes the health check (`healthCheckErr == nil`).
3.  **Error Reporting:** If `workingModel` is still empty after checking all models, `healthErr` (which holds the last error encountered or a default one if no specific model error occurred) is used for the `RespondError` call. This provides more specific feedback.
4.  **Timeout Restoration:** Ensures the HTTP client's timeout is restored to its original value (obtained via `GetHTTPLLMTimeout`) after the health check loop.
5.  **Error Wrapping for `RespondError`**: Added a check to ensure that the error passed to `RespondError` is an `AppError`, wrapping it if necessary. This maintains consistency with the error handling framework and provides clearer, structured errors to the client.
6.  **Passing `article` Object**: The conceptual code correctly uses `llmClient.ScoreWithModel(article, ...)` as `ScoreWithModel` expects the full article object, which is already fetched earlier in `reanalyzeHandler`.
7.  **Health Check Timeout Override:** Added an environment variable `HEALTH_CHECK_TIMEOUT_SECONDS` to allow overriding the default 2-second health check timeout per model, primarily for testing purposes.

### 3.3. Prerequisite: `GetHTTPLLMTimeout()` Method

To enable robust restoration of the LLM client's original timeout, the `GetHTTPLLMTimeout` method needs to be added to the `LLMClient` type in `internal/llm/llm.go`. This method will retrieve the current timeout value from the underlying HTTP service client.

**Conceptual Code for `GetHTTPLLMTimeout()`:**

```go
// Located in internal/llm/llm.go, within LLMClient

// GetHTTPLLMTimeout returns the current HTTP timeout for the LLM service.
// It defaults to defaultLLMTimeout if the specific service or client is not configured as expected.
func (c *LLMClient) GetHTTPLLMTimeout() time.Duration {
    httpService, ok := c.llmService.(*HTTPLLMService)
    // Ensure all parts of the chain are non-nil before dereferencing
    if ok && httpService != nil && httpService.client != nil && httpService.client.GetClient() != nil {
        return httpService.client.GetClient().Timeout
    }
    // Fallback to the package-level default LLM timeout if not specifically set or accessible
    log.Printf("[GetHTTPLLMTimeout] Warning: Could not retrieve specific timeout from HTTPLLMService, returning default: %v", defaultLLMTimeout)
    return defaultLLMTimeout 
}

```

## 4. Benefits

*   **Increased Resilience:** The system will attempt to use any available model in the ensemble for re-analysis initiation, even if some models (including those tried first) are experiencing timeouts or other non-rate-limiting failures.
*   **Better Ensemble Utilization:** Increases the likelihood that the more comprehensive `ReanalyzeArticle` function (which has its own robust model iteration) is invoked.
*   **Improved Logging:** Logs will detail attempts and failures for each model during the health check, aiding diagnostics.

## 5. Potential Considerations

*   **Health Check Duration:** If many models are configured and all are slow (but not dead), the health check phase could take longer (e.g., N models * 2s timeout). This is likely acceptable for initiating a background task.
*   **Error Propagation:** The `lastHealthCheckError` provides better context than a generic failure if all models fail the health check. 

## 6. Testing Strategy

Thorough testing is crucial to ensure the improved fallback mechanism in `reanalyzeHandler` behaves as expected and doesn't introduce regressions. This will involve both unit and integration tests.

### 6.1. Unit Tests (for `internal/api/api.go` - `reanalyzeHandler`)

While `reanalyzeHandler` is a http handler, the core logic for the health check loop can be tested with more focused tests if the LLM client interactions are appropriately mocked. However, given the existing integration test setup, most detailed behavioral tests will likely be integration tests.

Key areas for focused tests, potentially by refactoring parts of the health check into a testable helper function or through careful mocking at the handler level:
*   **`GetHTTPLLMTimeout` interaction:**
    *   Verify `llmClient.GetHTTPLLMTimeout()` is called.
    *   Verify `llmClient.SetHTTPLLMTimeout()` is called with the `healthCheckTimeout`.
    *   Verify `llmClient.SetHTTPLLMTimeout()` is called again with the `originalTimeout`.
*   **Environment Variable for Timeout:**
    *   Test that `HEALTH_CHECK_TIMEOUT_SECONDS` correctly overrides the default `healthCheckTimeout`.

### 6.2. Integration Tests (for `reanalyzeHandler` in `internal/api/api_integration_test.go`)

Leverage the existing `setupIntegrationTestServer` and `IntegrationMockLLMClient` from `internal/api/api_integration_test.go`. The `IntegrationMockLLMClient` should be configured to mock the `ScoreWithModel` method for different scenarios.

**Test Scenarios:**

Assume a configuration with at least three models (e.g., ModelA, ModelB, ModelC) for comprehensive testing.

1.  **Scenario: First Model Healthy**
    *   **Setup:**
        *   Mock `llmClient.ScoreWithModel` for ModelA to return `(0.0, nil)`.
    *   **Expected:**
        *   `reanalyzeHandler` uses ModelA.
        *   Background `llmClient.ReanalyzeArticle` is initiated (if `NO_AUTO_ANALYZE` is not true).
        *   API returns 202 "reanalyze queued".

2.  **Scenario: Second Model Healthy, First Fails (Non-RateLimit Error)**
    *   **Setup:**
        *   Mock `llmClient.ScoreWithModel` for ModelA to return `(0.0, errors.New("generic error"))`.
        *   Mock `llmClient.ScoreWithModel` for ModelB to return `(0.0, nil)`.
    *   **Expected:**
        *   Health check attempts ModelA (fails), then ModelB (passes).
        *   `reanalyzeHandler` uses ModelB.
        *   Background `llmClient.ReanalyzeArticle` is initiated.
        *   API returns 202 "reanalyze queued".

3.  **Scenario: Second Model Healthy, First Fails (RateLimit Error - special but should still continue)**
    *   **Setup:**
        *   Mock `llmClient.ScoreWithModel` for ModelA to return `(0.0, llm.ErrBothLLMKeysRateLimited)`.
        *   Mock `llmClient.ScoreWithModel` for ModelB to return `(0.0, nil)`.
    *   **Expected:**
        *   Health check attempts ModelA (fails with rate limit), then ModelB (passes).
        *   `reanalyzeHandler` uses ModelB.
        *   API returns 202 "reanalyze queued".

4.  **Scenario: All Models Fail Health Check (Different Errors)**
    *   **Setup:**
        *   Mock `llmClient.ScoreWithModel` for ModelA to return `(0.0, errors.New("timeout error A"))` (simulating timeout).
        *   Mock `llmClient.ScoreWithModel` for ModelB to return `(0.0, llm.LLMAPIError{Message: "auth error B", StatusCode: 401, ErrorType: llm.ErrTypeAuthentication})`.
        *   Mock `llmClient.ScoreWithModel` for ModelC to return `(0.0, errors.New("generic error C"))`.
    *   **Expected:**
        *   Health check attempts ModelA, ModelB, and ModelC; all fail.
        *   API returns an appropriate error status (e.g., 503 Service Unavailable).
        *   The error response should be an `apperrors.AppError` wrapping the last error encountered (e.g., "generic error C").
        *   `llmClient.ReanalyzeArticle` is NOT called.

5.  **Scenario: All Models Fail Health Check (All Timeouts)**
    *   **Setup:**
        *   Use `HEALTH_CHECK_TIMEOUT_SECONDS=1` (or a very short duration).
        *   Mock `llmClient.ScoreWithModel` for all models to simulate a delay longer than the health check timeout (e.g., by having the mock function sleep briefly before returning `nil`, or by having it return `context.DeadlineExceeded`).
    *   **Expected:**
        *   All health checks time out.
        *   API returns an appropriate error, likely related to the timeout (e.g., an `apperrors.AppError` wrapping `context.DeadlineExceeded` or a custom timeout error from `ScoreWithModel`).
        *   `llmClient.ReanalyzeArticle` is NOT called.

6.  **Scenario: `NO_AUTO_ANALYZE=true`**
    *   **Setup:**
        *   Set `os.Setenv("NO_AUTO_ANALYZE", "true")`.
        *   Ensure `llmClient.ScoreWithModel` is NOT expected to be called (or if it is for some passthrough, it doesn't affect the outcome related to health check).
    *   **Expected:**
        *   Health check loop is skipped.
        *   `reanalyzeHandler` assumes the first configured model if available.
        *   Background `llmClient.ReanalyzeArticle` is NOT initiated.
        *   API returns 202 "reanalyze queued" (or appropriate status for this mode).
    *   **Cleanup:** `os.Unsetenv("NO_AUTO_ANALYZE")`.

7.  **Scenario: Empty LLM Model Configuration**
    *   **Setup:**
        *   Mock `llm.LoadCompositeScoreConfig()` to return an empty `cfg.Models` list.
    *   **Expected:**
        *   `reanalyzeHandler` returns an error early (e.g., "No LLM models configured").
        *   This is existing behavior but should be re-verified.

8.  **Scenario: Error Wrapping Verification**
    *   **Setup:**
        *   Similar to Scenario 4, ensure the last error from `ScoreWithModel` is *not* already an `*apperrors.AppError`.
    *   **Expected:**
        *   The `RespondError` function receives a correctly wrapped `*apperrors.AppError` with code `apperrors.ErrLLMService` and details from the original error.
        *   Verify based on `TestReanalyzeEndpointLLMErrorPropagation` for structure.

### 6.3. Tests for `internal/llm/llm.go`

*   **Unit Test for `GetHTTPLLMTimeout()`:**
    *   Test with `llmService` as `*HTTPLLMService` and a configured `resty.Client` with a specific timeout. Verify the correct timeout is returned.
    *   Test with `llmService` as `*HTTPLLMService` but `httpService.client` or `httpService.client.GetClient()` is nil. Verify `defaultLLMTimeout` is returned and a warning is logged.
    *   Test with `llmService` not being an `*HTTPLLMService`. Verify `defaultLLMTimeout` is returned and a warning is logged.

**General Test Practices:**

*   Ensure all new tests have appropriate logging and use `t.Parallel()` where suitable.
*   Clean up any environment variables set during tests (e.g., `HEALTH_CHECK_TIMEOUT_SECONDS`, `NO_AUTO_ANALYZE`).
*   Extend existing test files (`internal/api/api_integration_test.go` and `internal/llm/llm_test.go`) rather than creating new ones unless a new testing paradigm is needed. 