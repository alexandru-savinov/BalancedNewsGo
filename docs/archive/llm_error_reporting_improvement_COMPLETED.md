# LLM Error Reporting Improvement Plan

## Rationale

Currently, when the `/api/llm/reanalyze/{id}` endpoint encounters an LLM error (such as authentication failure, rate limit, or service unavailability), the API often returns a generic error response (e.g., 503 Service Unavailable, code: `llm_service_error`). This obscures the true cause of the failure from both users and operators, making debugging and user support more difficult. The error details are logged, but not surfaced in the API response.

**Goal:**
- Ensure that all LLM-related errors (auth, rate limit, streaming, etc.) are reported to the client with the correct HTTP status, error code, and actionable message, including LLM error details.
- Improve operational visibility into LLM service issues.
- Enable clients to implement smarter retry strategies based on error types.

---

## Problems Identified

1. **Loss of Error Specificity:**
   - The `reanalyzeHandler` logs the real LLM error but returns a generic `ErrLLMUnavailable` to the client.
   - Clients receive only a 503 and a generic message, not the root cause (e.g., 401 Unauthorized for auth failure).

2. **Inconsistent Error Mapping:**
   - The `RespondError` function in `internal/api/response.go` is designed to map `LLMAPIError` to specific HTTP statuses and error codes, but this is bypassed if the handler does not propagate the real error.

3. **Lack of Test Coverage:**
   - There are no integration tests verifying that LLM errors are correctly propagated to the API response.

4. **Inefficient Troubleshooting:**
   - Support teams must access server logs to diagnose issues that could be identified directly from API responses.
   - No standardized error taxonomy for different LLM providers.

5. **Suboptimal Client Behavior:**
   - Clients cannot implement intelligent retry or fallback strategies without detailed error information.

---

## Solution Plan

### 1. Propagate Real LLM Errors in Handlers
- **Change:** In `reanalyzeHandler` (and any similar handlers), when an LLM error occurs (e.g., during health check or scoring), pass the actual error (e.g., `healthErr`) to `RespondError` instead of a generic error.
- **Example:**
  ```go
  if workingModel == "" {
      log.Printf("[reanalyzeHandler %d] No working models found (health check failed or skipped with no models): %v", articleID, healthErr)
      RespondError(c, healthErr) // <-- propagate the real error
      return
  }
  ```

### 2. Review and Update All LLM-Related Error Paths
- **Audit** all API handlers that interact with LLM services (reanalyze, scoring, feedback, etc.) to ensure they propagate real LLM errors to `RespondError`.
- **Update** any that use generic errors for LLM failures.
- **Key modules to check:**
  - `internal/api/api.go` - All LLM-related handlers
  - `internal/llm/llm.go` - LLM client error generation
  - `internal/llm/ensemble.go` - Error handling in ensemble scoring
  - `cmd/score_articles/main.go` - Batch scoring error handling

### 3. Standardize LLM Error Classification
- Implement a unified error taxonomy for different LLM providers (OpenAI, Anthropic, OpenRouter, etc.)
- Map provider-specific errors to our standard error types:
  ```go
  // Example mapping table
  var errorTypeMapping = map[string]string{
      "insufficient_quota": "rate_limit",
      "token_limit_exceeded": "content_filter",
      "context_length_exceeded": "input_too_long",
      // Add mappings for all providers
  }
  ```

### 4. Enhance Error Logging and Monitoring
- Add structured logging for all LLM errors with standardized fields.
- Implement Prometheus metrics for LLM error counts by type and provider.
- Set up alerting thresholds for unusual error rates.
- Example metrics:
  ```go
  llmErrorCounter := prometheus.NewCounterVec(
      prometheus.CounterOpts{
          Name: "llm_api_errors_total",
          Help: "Total number of LLM API errors by type and provider",
      },
      []string{"error_type", "provider", "model"},
  )
  ```

### 5. Update Error Response Structure
- Extend the error response format to include:
  - Human-readable message
  - Machine-readable error code
  - Provider-specific error details (safely sanitized)
  - Suggested client action (retry, configuration change, contact support)
  - Correlation ID for log tracing

### 6. Update API Documentation
- Update OpenAPI/Swagger docs to reflect the new error status codes and response schemas for LLM errors.
- Document all possible error codes, their meanings, and recommended client actions.
- Include examples of each error type for client implementation reference.

### 7. Add/Update Integration Tests
- Add tests that:
  - Simulate LLM authentication failure, rate limit, and streaming errors.
  - Assert that the API returns the correct HTTP status, error code, and error details in the response body.
- Update existing tests to expect the new, more specific error responses.
- Test error handling for each supported LLM provider.
- Add performance tests to ensure error handling doesn't introduce latency.

### 8. Implement Graceful Degradation
- When specific LLM providers fail, attempt to use alternatives before failing completely.
- Document the fallback behavior in the API response.
- Add configuration options for fallback policies.

### 9. Security Review
- Perform security review to ensure error details don't leak sensitive information.
- Implement sanitization of provider error messages to remove API keys or sensitive data.
- Add rate limiting for error responses to prevent information disclosure via timing attacks.

### 10. Rollout Strategy
- **Phase 1:** Implement changes in development/staging environments
- **Phase 2:** Enable detailed logging in production without changing responses
- **Phase 3:** Roll out new error responses with feature flag
- **Phase 4:** Monitor error rates and client behavior
- **Phase 5:** Full deployment and client notification

---

## Example: Improved Error Response

**Before:**
```json
{
  "success": false,
  "error": {
    "code": "llm_service_error",
    "message": "LLM service unavailable"
  }
}
```

**After (for LLM auth failure):**
```json
{
  "success": false,
  "error": {
    "code": "llm_authentication_error",
    "message": "LLM service authentication failed",
    "details": {
      "provider": "openai",
      "model": "gpt-4",
      "llm_status_code": 401,
      "llm_message": "Invalid API key provided",
      "error_type": "authentication",
      "correlation_id": "req_7PxEi3a4XkHQZm"
    },
    "recommended_action": "Contact administrator to update API credentials",
    "retry_after_seconds": null
  }
}
```

**After (for rate limiting):**
```json
{
  "success": false,
  "error": {
    "code": "llm_rate_limit_error",
    "message": "LLM service rate limit exceeded",
    "details": {
      "provider": "anthropic",
      "model": "claude-3-opus",
      "llm_status_code": 429,
      "llm_message": "Rate limit exceeded",
      "error_type": "rate_limit",
      "correlation_id": "req_9TxFj5c8ZlBQCn"
    },
    "recommended_action": "Retry after backoff period",
    "retry_after_seconds": 30
  }
}
```

---

## Implementation Steps

1. [ ] Refactor `reanalyzeHandler` to propagate real LLM errors to `RespondError`.
2. [ ] Audit and update other handlers for similar error propagation.
3. [ ] Implement standardized error taxonomy for all LLM providers.
4. [ ] Update error response structure in `internal/api/response.go`.
5. [ ] Add metrics collection for LLM errors in `internal/metrics/prom.go`.
6. [ ] Implement security sanitization for error details.
7. [ ] Update API documentation (Swagger/OpenAPI) for new error responses.
8. [ ] Add/expand integration tests for LLM error propagation.
9. [ ] Implement feature flag for gradual rollout.
10. [ ] Update client documentation and communicate the change.
11. [ ] Set up monitoring dashboards for error metrics.
12. [ ] Conduct post-implementation review to verify effectiveness.

---

## Success Criteria

1. **Technical Validation:**
   - All integration tests pass with the new error responses.
   - No sensitive information is exposed in error details.
   - Error response time remains under 100ms, even with detailed responses.

2. **Operational Validation:**
   - Support tickets related to ambiguous LLM errors decrease by 50%.
   - Time to diagnose LLM issues decreases by 60%.
   - Dashboards show clear visibility into error rates by type.

3. **User Experience:**
   - Client applications can implement intelligent retry strategies.
   - User-facing error messages are more actionable and less frustrating.

---

## References
- `internal/api/api.go` (reanalyzeHandler)
- `internal/api/response.go` (RespondError)
- `internal/llm/llm.go` (LLMAPIError)
- `internal/metrics/prom.go` (metrics collection)
- `docs/testing.md`
