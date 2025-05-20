# OpenRouter Error Handling Improvement Plan## OverviewThis document outlines a comprehensive plan to enhance the error handling of OpenRouter API responses within the NewsBalancer application. The goal is to make OpenRouter error messages more transparent and user-friendly, enabling better debugging and troubleshooting.## BackgroundCurrently, when the LLM service (OpenRouter) returns errors, they are logged internally but not always surfaced in a user-friendly way through the API. Clients often see generic 503 Service Unavailable responses without detailed error information, making it difficult to diagnose issues.From the OpenRouter API documentation, we've identified several specific error types that need special handling:- Rate Limiting (429): Occurs when you exceed 1 request per credit per second- Payment Required (402): Appears when account has negative credit balance - Authentication Errors (401): Invalid API key- Standard HTTP errors: 400 (Bad Request), 403 (Forbidden), 500 (Server Error)- Streaming-specific errors: Related to SSE parsing

## Implementation Plan

### Step 1: Enhance LLM Error Handling in `internal/llm/service_http.go`

Update the existing error handling to align with the `apperrors` package:

```go
// Existing service_http.go file - enhance with these changes

// OpenRouterErrorType represents specific error types from OpenRouter
type OpenRouterErrorType string

const (
    // OpenRouter specific error types
    ErrTypeRateLimit      OpenRouterErrorType = "rate_limit"
    ErrTypeAuthentication OpenRouterErrorType = "authentication"
    ErrTypeCredits        OpenRouterErrorType = "credits"
    ErrTypeStreaming      OpenRouterErrorType = "streaming"
    ErrTypeUnknown        OpenRouterErrorType = "unknown"
)

// LLMAPIError wraps OpenRouter errors with additional context
type LLMAPIError struct {
    *apperrors.AppError
    StatusCode   int
    ResponseBody string
    ErrorType    OpenRouterErrorType
    RetryAfter   int    // For rate limit errors
}

func (e *LLMAPIError) Error() string {
    return fmt.Sprintf("LLM API Error (%s): %s [HTTP %d]", e.ErrorType, e.Message, e.StatusCode)
}

// Create error constants for consistent OpenRouter error messages
var (
    ErrBothLLMKeysRateLimited  = apperrors.New("Both primary and backup LLM API keys are rate limited", "llm_rate_limit")
    ErrLLMServiceUnavailable   = apperrors.New("LLM service is temporarily unavailable", "llm_service_unavailable")
    ErrLLMAuthenticationFailed = apperrors.New("LLM service authentication failed", "llm_authentication")
    ErrLLMCreditsExhausted     = apperrors.New("LLM service credits exhausted", "llm_credits")
    ErrLLMStreamingFailed      = apperrors.New("LLM streaming response failed", "llm_streaming")
)

// formatHTTPError converts HTTP responses to structured LLMAPIError objects
func formatHTTPError(resp *resty.Response) error {
    // Initialize default values
    errorType := ErrTypeUnknown
    retryAfter := 0
    message := "Unknown LLM API error"
    
    // Try to parse the error response
    var openRouterError struct {
        Error struct {
            Message string `json:"message"`
            Type    string `json:"type"`
            Code    string `json:"code"`
        } `json:"error"`
    }
    
    if err := json.Unmarshal([]byte(resp.String()), &openRouterError); err == nil && openRouterError.Error.Message != "" {
        message = openRouterError.Error.Message
    } else {
        // Use status text if can't parse JSON
        message = resp.Status()
    }
    
    // Identify error type and additional metadata
    switch resp.StatusCode() {
    case 429:
        errorType = ErrTypeRateLimit
        if retryHeader := resp.Header().Get("Retry-After"); retryHeader != "" {
            retryAfter, _ = strconv.Atoi(retryHeader)
        }
        // Increment rate limit metric
        metrics.IncLLMRateLimit()
    case 402:
        errorType = ErrTypeCredits
        // Increment credits exhausted metric
        metrics.IncLLMCreditsExhausted()
    case 401:
        errorType = ErrTypeAuthentication
        // Increment authentication failure metric
        metrics.IncLLMAuthFailure()
    default:
        // Increment generic LLM failure metric
        metrics.IncLLMFailure()
    }
    
    // Create appropriate AppError based on error type
    var appErr *apperrors.AppError
    switch errorType {
    case ErrTypeRateLimit:
        appErr = apperrors.New(fmt.Sprintf("LLM rate limit exceeded: %s", message), "llm_rate_limit")
    case ErrTypeCredits:
        appErr = apperrors.New(fmt.Sprintf("LLM credits exhausted: %s", message), "llm_credits")
    case ErrTypeAuthentication:
        appErr = apperrors.New(fmt.Sprintf("LLM authentication failed: %s", message), "llm_authentication")
    default:
        appErr = apperrors.New(fmt.Sprintf("LLM service error: %s", message), "llm_service_error")
    }
    
    // Wrap in LLMAPIError with additional context
    return &LLMAPIError{
        AppError:     appErr,
        StatusCode:   resp.StatusCode(),
        ResponseBody: sanitizeResponse(resp.String()),
        ErrorType:    errorType,
        RetryAfter:   retryAfter,
    }
}

// Sanitize response to remove sensitive info
func sanitizeResponse(response string) string {
    // Simple sanitization - remove potential API keys
    sanitized := regexp.MustCompile(`(sk-|or-)[a-zA-Z0-9]{20,}`).ReplaceAllString(response, "[REDACTED]")
    return sanitized
}
```

### Step 2: Update Error Handling in API Layer (`internal/api/response.go`)

Enhance the `RespondError` function to handle LLM-specific errors:

```go
// Add to imports at the top of response.go
import (
    "github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

// Update RespondError to handle LLMAPIError specially
func RespondError(c *gin.Context, err error) {
    statusCode := http.StatusInternalServerError
    errorResponse := ErrorResponse{
        Code:    "internal_error",
        Message: "Internal server error",
    }
    
    // Handle LLMAPIError
    var llmErr *llm.LLMAPIError
    if errors.As(err, &llmErr) {
        // Map LLM error types to appropriate HTTP status codes and responses
        switch llmErr.ErrorType {
        case llm.ErrTypeRateLimit:
            statusCode = http.StatusTooManyRequests // 429
            errorResponse.Code = ErrRateLimit
            errorResponse.Message = "LLM rate limit exceeded"
            // Add retry-after header if available
            if llmErr.RetryAfter > 0 {
                c.Header("Retry-After", strconv.Itoa(llmErr.RetryAfter))
            }
        case llm.ErrTypeCredits:
            statusCode = http.StatusPaymentRequired // 402
            errorResponse.Code = ErrLLMService
            errorResponse.Message = "LLM service payment required"
        case llm.ErrTypeAuthentication:
            statusCode = http.StatusUnauthorized // 401
            errorResponse.Code = ErrLLMService
            errorResponse.Message = "LLM service authentication failed"
        case llm.ErrTypeStreaming:
            statusCode = http.StatusServiceUnavailable // 503
            errorResponse.Code = ErrLLMService
            errorResponse.Message = "LLM streaming service error"
        default:
            statusCode = http.StatusServiceUnavailable // 503
            errorResponse.Code = ErrLLMService
            errorResponse.Message = "LLM service error"
        }
        
        // Include detailed error information
        errorResponse.Details = map[string]interface{}{
            "llm_status_code": llmErr.StatusCode,
            "llm_message": llmErr.Message,
            "llm_error_type": string(llmErr.ErrorType),
        }
        
        // Only include retry_after if present
        if llmErr.RetryAfter > 0 {
            errorResponse.Details["retry_after"] = llmErr.RetryAfter
        }
        
        // Log the error with appropriate level based on type
        LogError(c, llmErr, fmt.Sprintf("LLM error (%s): %s", llmErr.ErrorType, llmErr.Message))
        
        c.JSON(statusCode, errorResponse)
        return
    }
    
    // Handle regular AppError (existing code)
    var appError *apperrors.AppError
    if errors.As(err, &appError) {
        // Existing AppError handling code...
    }
    // ...rest of existing error handling...
}
```

### Step 3: Add Streaming Error Detection in LLM Client (`internal/llm/ensemble.go`)

```go
// Add to callLLM function in ensemble.go

func (c *LLMClient) callLLM(ctx context.Context, content, systemPrompt, userPrompt string, model string) (*db.LLMScore, error) {
    // Existing code...
    
    // Enhanced error handling for SSE/streaming errors
    response, err := c.service.ScoreContent(ctx, content, systemPrompt, userPrompt, model)
    if err != nil {
        // Check for streaming-specific errors
        if strings.Contains(err.Error(), "SSE") || 
           strings.Contains(err.Error(), "stream") || 
           strings.Contains(err.Error(), "PROCESSING") {
            // Convert to streaming-specific error
            return nil, &LLMAPIError{
                AppError:     ErrLLMStreamingFailed,
                StatusCode:   http.StatusServiceUnavailable,
                ResponseBody: err.Error(),
                ErrorType:    ErrTypeStreaming,
            }
        }
        return nil, err
    }
    
    // Rest of existing code...
}
```

### Step 4: Update Progress Manager to Handle LLM Errors (`internal/llm/progress_manager.go`)

```go
// Add to UpdateProgress in progress_manager.go

func (pm *ProgressManager) UpdateProgress(articleID int, step string, percent float64, status string, err error) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    state, exists := pm.progressMap[articleID]
    if !exists {
        state = &models.ProgressState{
            LastUpdated: time.Now(),
        }
        pm.progressMap[articleID] = state
    }
    
    state.Step = step
    state.Percent = percent
    state.Status = status
    state.LastUpdated = time.Now()
    
    // Enhanced error handling for LLM errors
    if err != nil {
        state.Error = err.Error()
        
        // Add specific error details for LLM errors
        var llmErr *LLMAPIError
        if errors.As(err, &llmErr) {
            errorDetails := map[string]interface{}{
                "type": string(llmErr.ErrorType),
                "status_code": llmErr.StatusCode,
            }
            
            // Convert to JSON string
            if detailsJSON, jsonErr := json.Marshal(errorDetails); jsonErr == nil {
                state.ErrorDetails = string(detailsJSON)
            }
        }
    } else {
        state.Error = ""
        state.ErrorDetails = ""
    }
}
```

### Step 5: Add Error-Specific Metrics (`internal/metrics/prom.go`)

Add new metrics for specific OpenRouter error types:

```go
// Add to prom.go

var (
    // Existing metrics...
    
    // New metrics for specific LLM error types
    LLMRateLimitCounter   prometheus.Counter
    LLMAuthFailureCounter prometheus.Counter
    LLMCreditsCounter     prometheus.Counter
    LLMStreamingErrors    prometheus.Counter
)

// Initialize new metrics in InitLLMMetrics
func InitLLMMetrics() {
    // Existing metric initialization...
    
    // Initialize new metrics
    LLMRateLimitCounter = promauto.NewCounter(prometheus.CounterOpts{
        Name: "llm_rate_limit_total",
        Help: "Total number of rate limit errors from OpenRouter",
    })
    
    LLMAuthFailureCounter = promauto.NewCounter(prometheus.CounterOpts{
        Name: "llm_auth_failure_total",
        Help: "Total number of authentication failures with OpenRouter",
    })
    
    LLMCreditsCounter = promauto.NewCounter(prometheus.CounterOpts{
        Name: "llm_credits_exhausted_total",
        Help: "Total number of credit exhaustion errors from OpenRouter",
    })
    
    LLMStreamingErrors = promauto.NewCounter(prometheus.CounterOpts{
        Name: "llm_streaming_errors_total",
        Help: "Total number of streaming-related errors from OpenRouter",
    })
}

// Add helper functions for incrementing each metric
func IncLLMRateLimit() {
    LLMRateLimitCounter.Inc()
    LLMFailuresTotal.Inc() // Also increment the general failures counter
}

func IncLLMAuthFailure() {
    LLMAuthFailureCounter.Inc()
    LLMFailuresTotal.Inc()
}

func IncLLMCreditsExhausted() {
    LLMCreditsCounter.Inc()
    LLMFailuresTotal.Inc()
}

func IncLLMStreamingError() {
    LLMStreamingErrors.Inc()
    LLMFailuresTotal.Inc()
}
```

### Step 6: Add New Testing Infrastructure for OpenRouter Errors

Create a new test file `internal/llm/service_http_test.go`:

```go
package llm

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
    "github.com/alexandru-savinov/BalancedNewsGo/internal/metrics"
    "github.com/stretchr/testify/assert"
)

func TestOpenRouterErrorHandling(t *testing.T) {
    // Initialize metrics
    metrics.InitLLMMetrics()
    
    // Set up test cases
    testCases := []struct {
        name           string
        statusCode     int
        responseBody   string
        headers        map[string]string
        expectedType   OpenRouterErrorType
        expectedRetry  int
        expectedStatus int
    }{
        {
            name:           "Rate Limit Error",
            statusCode:     429,
            responseBody:   `{"error":{"message":"Rate limit exceeded","type":"rate_limit_error"}}`,
            headers:        map[string]string{"Retry-After": "30"},
            expectedType:   ErrTypeRateLimit,
            expectedRetry:  30,
            expectedStatus: 429,
        },
        {
            name:           "Authentication Error",
            statusCode:     401,
            responseBody:   `{"error":{"message":"Invalid API key","type":"authentication_error"}}`,
            expectedType:   ErrTypeAuthentication,
            expectedRetry:  0,
            expectedStatus: 401,
        },
        {
            name:           "Credits Exhausted",
            statusCode:     402,
            responseBody:   `{"error":{"message":"Insufficient credits","type":"credits_error"}}`,
            expectedType:   ErrTypeCredits,
            expectedRetry:  0,
            expectedStatus: 402,
        },
        {
            name:           "Server Error",
            statusCode:     500,
            responseBody:   `{"error":{"message":"Internal server error"}}`,
            expectedType:   ErrTypeUnknown,
            expectedRetry:  0,
            expectedStatus: 500,
        },
        {
            name:           "Malformed JSON",
            statusCode:     400,
            responseBody:   `Not a JSON response`,
            expectedType:   ErrTypeUnknown,
            expectedRetry:  0,
            expectedStatus: 400,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Set up test server
            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // Set headers
                for k, v := range tc.headers {
                    w.Header().Set(k, v)
                }
                w.WriteHeader(tc.statusCode)
                _, _ = w.Write([]byte(tc.responseBody))
            }))
            defer server.Close()
            
            // Create client pointing to test server
            client := &HTTPLLMService{
                client:  http.DefaultClient,
                baseURL: server.URL,
                apiKey:  "test-key",
            }
            
            // Call API
            _, err := client.ScoreContent(context.Background(), "test content", "system prompt", "user prompt", "test-model")
            
            // Assert error type
            assert.Error(t, err)
            
            // Check error details
            var llmErr *LLMAPIError
            if assert.True(t, errors.As(err, &llmErr), "Expected LLMAPIError") {
                assert.Equal(t, tc.expectedType, llmErr.ErrorType)
                assert.Equal(t, tc.expectedRetry, llmErr.RetryAfter)
                assert.Equal(t, tc.expectedStatus, llmErr.StatusCode)
            }
        })
    }
}
```

### Step 7: Update Documentation

Add a new section to `docs/codebase_documentation.md`:

```markdown
## OpenRouter Error Handling

The application includes robust error handling for the OpenRouter LLM service, which is used for article analysis. This section explains how different types of OpenRouter errors are handled.

### OpenRouter Error Types

| HTTP Status | Error Type | Description | Retry Strategy |
|-------------|------------|-------------|----------------|
| 429 | Rate Limit | Occurs when exceeding 1 request per credit per second | Respects Retry-After header, falls back to secondary key |
| 402 | Credits Exhausted | Account has negative credit balance | No automatic retry, requires account top-up |
| 401 | Authentication | Invalid API key | No retry, requires API key verification |
| 4xx/5xx | Other Errors | Bad request, server errors, etc. | Limited retries with exponential backoff |

### Error Response Format

When an OpenRouter error occurs, the API responds with:

```json
{
  "code": "llm_error_type",
  "message": "Human-readable error description",
  "details": {
    "llm_status_code": 429,
    "llm_message": "Original error message from OpenRouter",
    "llm_error_type": "rate_limit",
    "retry_after": 30
  }
}
```

### Monitoring and Metrics

OpenRouter errors are tracked using Prometheus metrics:
- `llm_requests_total`: Total number of requests made to OpenRouter
- `llm_failures_total`: Total number of failed requests to OpenRouter
- `llm_rate_limit_total`: Number of rate limit errors
- `llm_auth_failure_total`: Number of authentication failures
- `llm_credits_exhausted_total`: Number of credit exhaustion errors
- `llm_streaming_errors_total`: Number of streaming-related errors

### Troubleshooting OpenRouter Errors

1. **Rate Limit Errors**:
   - Check for multiple concurrent requests
   - Verify secondary API key is configured
   - Consider implementing request throttling

2. **Authentication Errors**:
   - Verify API key in `.env` file
   - Check for proper API key format
   - Confirm account is active on OpenRouter

3. **Credits Exhausted**:
   - Top up your OpenRouter account
   - Monitor usage patterns to avoid unexpected exhaustion
   - Consider implementing usage alerts

4. **Streaming Errors**:
   - Check for network connectivity issues
   - Verify OpenRouter streaming endpoint status
   - Consider falling back to non-streaming API
```

## Implementation Timeline

1. Enhance `LLMAPIError` structure and error handler: 1 day
2. Update API error handling in `RespondError`: 0.5 day
3. Add streaming error detection: 0.5 day
4. Update Progress Manager for LLM errors: 0.5 day
5. Add error-specific metrics: 0.5 day
6. Implement testing for error scenarios: 1 day
7. Update documentation: 0.5 day

**Total estimated time: 4.5 days**

## Expected Benefits

1. More transparent error reporting to clients
2. Better debuggability for LLM-related issues
3. Improved user experience with clear error messages
4. Proper handling of OpenRouter-specific error cases
5. Standardized error response format across the application
6. Enhanced metrics for monitoring and alerting
7. Robust testing for error conditions
8. Clear documentation for troubleshooting

## Summary Table

| Step | File(s) to Edit | Action | OpenRouter-Specific Improvement |
|------|-----------------|--------|--------------------------------|
| 1 | `internal/llm/service_http.go` | Enhance `LLMAPIError` structure | Add error type constants, align with `apperrors.AppError` |
| 2 | `internal/api/response.go` | Update `RespondError` | Map OpenRouter errors to appropriate HTTP responses |
| 3 | `internal/llm/ensemble.go` | Add streaming error detection | Detect and categorize streaming-specific errors |
| 4 | `internal/llm/progress_manager.go` | Update progress handling | Add error details to progress state for LLM errors |
| 5 | `internal/metrics/prom.go` | Add error-specific metrics | Track different types of OpenRouter errors separately |
| 6 | `internal/llm/service_http_test.go` | Create tests | Test different OpenRouter error scenarios |
| 7 | `docs/codebase_documentation.md` | Update documentation | Add section on OpenRouter error handling |



## Failing Tests: OpenRouter Error Handling (Latest Run)

Below are the failing tests from the latest run of the OpenRouter error handling suite. Each entry includes the test name, file, line, error message, and relevant context to aid debugging.

---

### 1. TestAnalyzeAndStore (internal/llm/llm_test.go:567)
- **Error:** An error is expected but got nil.
- **Message:** AnalyzeAndStore should return error on DB failure
- **Trace:**
  - File: internal/llm/llm_test.go
  - Line: 567

---

### 2. TestOpenRouterErrorTypes/Streaming_Error (internal/llm/open_router_errors_test.go:107)
- **Error:** Not equal: expected: "streaming" actual: "unknown"
- **Message:** Error type mismatch
- **Trace:**
  - File: internal/llm/open_router_errors_test.go
  - Line: 107
  - Expected: (llm.OpenRouterErrorType) (len=9) "streaming"
  - Actual:   (llm.OpenRouterErrorType) (len=7) "unknown"

---

### 3. TestProgressManager_UpdateProgressWithLLMError (internal/llm/progress_manager_test.go:246)
- **Error:** Should NOT be empty, but was <nil>
- **Message:** Error details should include message
- **Trace:**
  - File: internal/llm/progress_manager_test.go
  - Line: 246
  - Fails for subtests: LLM_Rate_Limit_Error, LLM_Authentication_Error, LLM_Credits_Error, LLM_Streaming_Error, LLM_Unknown_Error

---

### 4. TestLLMAPIError_Error (internal/llm/service_http_test.go:95)
- **Error:** Not equal: expected: "LLM API Error (status 401): Invalid token" actual: "LLM API Error (): Status 401 - Invalid token"
- **Trace:**
  - File: internal/llm/service_http_test.go
  - Line: 95
  - Fails for subtests: Standard_API_error, Rate_limit_error, Server_error, Empty_message
  - Diff example:
    - Expected: LLM API Error (status 401): Invalid token
    - Actual:   LLM API Error (): Status 401 - Invalid token

---

### 5. TestOpenRouterErrorHandling/Rate_Limit_Error (internal/llm/service_http_test.go:194)
- **Error:** Should be true
- **Message:** Expected error to be of type LLMAPIError, got *fmt.wrapError
- **Trace:**
  - File: internal/llm/service_http_test.go
  - Line: 194

---

### 6. TestFormatHTTPError (internal/llm/service_http_test.go:302)
- **Error:**
  - "Rate limit exceeded" does not contain "LLM rate limit exceeded: Rate limit exceeded"
  - "Internal Server Error" does not contain "LLM service error: 500 Internal Server Error"
  - "Complex error" does not contain "LLM service error: Complex error"
- **Trace:**
  - File: internal/llm/service_http_test.go
  - Line: 302
  - Fails for subtests: Rate_Limit_With_Details, Empty_Response, Complex_Nested_Error

---

**Note:**
- For each failure, check the corresponding test and implementation for mismatches in error type mapping, error message formatting, and propagation of error details.
- Some failures indicate a mismatch between expected error type (e.g., "streaming") and what is actually returned (e.g., "unknown"). Others are due to error message formatting or missing error details in progress tracking.
- Addressing these will improve the reliability and transparency of OpenRouter error handling in the NewsBalancer application. 

## Implementation Plan for Fixing Failing Tests

Below is a detailed plan to fix all failing tests in the OpenRouter error handling implementation. The plan addresses each issue in priority order, with specific code changes, validation steps, and estimated implementation time.

### Priority 1: Fix OpenRouter Error Type Handling and Error Formatting
**Files to Change**: `internal/llm/service_http.go`  
**Estimated Time**: 1 day  
**Tests Addressed**: TestOpenRouterErrorTypes, TestLLMAPIError_Error, TestFormatHTTPError

#### Code Changes

```go
// 1. Update error type detection in formatHTTPError

func formatHTTPError(resp *resty.Response) error {
    // Initialize default values
    errorType := ErrTypeUnknown
    retryAfter := 0
    message := "Unknown LLM API error"
    
    // Try to parse the error response
    var openRouterError struct {
        Error struct {
            Message string `json:"message"`
            Type    string `json:"type"`
            Code    string `json:"code"`
        } `json:"error"`
    }
    
    if err := json.Unmarshal([]byte(resp.String()), &openRouterError); err == nil && openRouterError.Error.Message != "" {
        message = openRouterError.Error.Message
    } else {
        // Use status text if can't parse JSON
        message = resp.Status()
    }
    
    // Identify error type and additional metadata
    switch resp.StatusCode() {
    case 429:
        errorType = ErrTypeRateLimit
        if retryHeader := resp.Header().Get("Retry-After"); retryHeader != "" {
            retryAfter, _ = strconv.Atoi(retryHeader)
        }
        // Increment rate limit metric
        metrics.IncLLMFailure("openrouter", "", "rate_limit")
    case 402:
        errorType = ErrTypeCredits
        // Increment credits exhausted metric
        metrics.IncLLMFailure("openrouter", "", "credits")
    case 401:
        errorType = ErrTypeAuthentication
        // Increment authentication failure metric
        metrics.IncLLMFailure("openrouter", "", "authentication")
    case 503:
        // Check if this is a streaming error based on message content
        if strings.Contains(strings.ToLower(message), "stream") || 
           strings.Contains(strings.ToLower(message), "sse") {
            errorType = ErrTypeStreaming
            metrics.IncLLMFailure("openrouter", "", "streaming")
        } else {
            // Generic server error
            metrics.IncLLMFailure("openrouter", "", "other")
        }
    default:
        // For other error types, check the message content
        lowerMsg := strings.ToLower(message)
        if strings.Contains(lowerMsg, "stream") || strings.Contains(lowerMsg, "sse") {
            errorType = ErrTypeStreaming
            metrics.IncLLMFailure("openrouter", "", "streaming")
        } else {
            // Increment generic LLM failure metric
            metrics.IncLLMFailure("openrouter", "", "other")
        }
    }

    // Create appropriate error message prefix based on error type
    var errorPrefix string
    switch errorType {
    case ErrTypeRateLimit:
        errorPrefix = "LLM rate limit exceeded: "
    case ErrTypeCredits:
        errorPrefix = "LLM credits exhausted: "
    case ErrTypeAuthentication:
        errorPrefix = "LLM authentication failed: "
    case ErrTypeStreaming:
        errorPrefix = "LLM streaming failed: "
    default:
        errorPrefix = "LLM service error: "
    }
    
    // Return the structured error with formatted message
    return LLMAPIError{
        Message:      errorPrefix + message,
        StatusCode:   resp.StatusCode(),
        ResponseBody: sanitizeResponse(resp.String()),
        ErrorType:    errorType,
        RetryAfter:   retryAfter,
    }
}

// 2. Update LLMAPIError.Error method

// Error implements the error interface for LLMAPIError
func (e LLMAPIError) Error() string {
    return fmt.Sprintf("LLM API Error (%s): %s [HTTP %d]", e.ErrorType, e.Message, e.StatusCode)
}
```

#### Validation Steps
1. Run `go test ./internal/llm/... -run TestOpenRouterErrorTypes` - Verify streaming errors are detected
2. Run `go test ./internal/llm/... -run TestLLMAPIError_Error` - Verify error formatting matches expected
3. Run `go test ./internal/llm/... -run TestFormatHTTPError` - Verify error messages have the right prefix

### Priority 2: Fix LLMAPIError Propagation and Type Detection
**Files to Change**: `internal/llm/service_http.go`  
**Estimated Time**: 0.5 day  
**Tests Addressed**: TestOpenRouterErrorHandling/Rate_Limit_Error

#### Code Changes

```go
// Update error return in LLM service to avoid wrapping LLMAPIError

func (s *HTTPLLMService) ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (float64, float64, error) {
    // ... existing code ...
    
    resp, err := s.client.R().
        SetBody(requestBody).
        SetHeader("Content-Type", "application/json").
        SetHeader("Authorization", fmt.Sprintf("Bearer %s", s.apiKey)).
        Post(apiURL)
    
    if err != nil {
        // Don't wrap this, return directly
        return 0, 0, err
    }
    
    if resp.StatusCode() >= 400 {
        // Don't wrap the LLMAPIError, return it directly
        return 0, 0, formatHTTPError(resp)
    }
    
    // ... rest of existing code ...
}
```

#### Validation Steps
1. Run `go test ./internal/llm/... -run TestOpenRouterErrorHandling` - Verify LLMAPIError is correctly detected
2. Run `go test ./internal/llm/... -run TestRateLimitFallback` - Verify rate limit fallback still works

### Priority 3: Fix Error Details in Progress Manager
**Files to Change**: `internal/llm/progress_manager.go`  
**Estimated Time**: 0.5 day  
**Tests Addressed**: TestProgressManager_UpdateProgressWithLLMError

#### Code Changes

```go
// Update UpdateProgress to include message in error details

func (pm *ProgressManager) UpdateProgress(articleID int64, step string, percent int, status string, err error) {
    pm.progressMapLock.Lock()
    defer pm.progressMapLock.Unlock()

    state, exists := pm.progressMap[articleID]
    if !exists {
        state = &models.ProgressState{
            LastUpdated: time.Now().Unix(),
        }
        pm.progressMap[articleID] = state
    }

    state.Step = step
    state.Percent = percent
    state.Status = status
    state.LastUpdated = time.Now().Unix()

    // Enhanced error handling for LLM errors
    if err != nil {
        state.Error = err.Error()

        // Add specific error details for LLM errors
        var llmErr LLMAPIError
        if errors.As(err, &llmErr) {
            errorDetails := map[string]interface{}{
                "type":        string(llmErr.ErrorType),
                "status_code": llmErr.StatusCode,
                "message":     llmErr.Message, // Add the message to error details
            }

            // Only include retry_after if present
            if llmErr.RetryAfter > 0 {
                errorDetails["retry_after"] = llmErr.RetryAfter
            }

            // Convert to JSON string
            if detailsJSON, jsonErr := json.Marshal(errorDetails); jsonErr == nil {
                state.ErrorDetails = string(detailsJSON)
            }
        }
    } else {
        state.Error = ""
        state.ErrorDetails = ""
    }
}
```

#### Validation Steps
1. Run `go test ./internal/llm/... -run TestProgressManager_UpdateProgressWithLLMError` - Verify error message is included
2. Run `go test ./internal/llm/... -run TestProgressManager_ExportStateWithErrors` - Verify no regressions

### Priority 4: Fix AnalyzeAndStore Error Propagation
**Files to Change**: `internal/llm/llm.go`  
**Estimated Time**: 0.5 day  
**Tests Addressed**: TestAnalyzeAndStore

#### Code Changes

```go
// Update AnalyzeAndStore to propagate DB errors

func (c *LLMClient) AnalyzeAndStore(article *db.Article) error {
    if c.config == nil || len(c.config.Models) == 0 {
        log.Printf("[ERROR] LLMClient config is nil or has no models defined")
        return fmt.Errorf("LLMClient config is nil or has no models defined")
    }

    var lastErr error
    
    for _, m := range c.config.Models {
        log.Printf("[DEBUG][AnalyzeAndStore] Article %d | Perspective: %s | ModelName passed: %s | URL: %s", article.ID, m.Perspective, m.ModelName, m.URL)
        score, err := c.analyzeContent(article.ID, article.Content, m.ModelName)
        if err != nil {
            log.Printf("Error analyzing article %d with model %s: %v", article.ID, m.ModelName, err)
            lastErr = fmt.Errorf("error analyzing article %d with model %s: %w", article.ID, m.ModelName, err)
            continue
        }

        _, err = db.InsertLLMScore(c.db, score)
        if err != nil {
            log.Printf("Error inserting LLM score for article %d model %s: %v", article.ID, m.ModelName, err)
            lastErr = fmt.Errorf("failed to insert LLM score: %w", err)
            // Don't break here, try other models
        }
    }

    return lastErr // Return the last error encountered
}
```

#### Validation Steps
1. Run `go test ./internal/llm/... -run TestAnalyzeAndStore` - Verify DB errors are returned

## Implementation Timeline and Priority Order

| Priority | Task | Estimated Time | Reasoning |
|----------|------|----------------|-----------|
| 1 | Fix OpenRouter Error Type Handling | 1 day | Affects multiple tests and core error handling |
| 2 | Fix LLMAPIError Propagation | 0.5 day | Required for proper error type detection |
| 3 | Fix Progress Manager Error Details | 0.5 day | Required for proper error reporting in UI |
| 4 | Fix AnalyzeAndStore Error Propagation | 0.5 day | Required for proper error handling in batch jobs |

**Total Timeline**: 2.5 days

## Verification Strategy

The following detailed verification strategy ensures that all fixes are properly tested and validated before they are merged into the production codebase.

### 1. Unit Test Coverage

#### Individual Test Verification
After implementing each fix, perform targeted testing:

```bash
# Test specific features (examples)
go test ./internal/llm/... -run TestOpenRouterErrorTypes -v
go test ./internal/llm/... -run TestLLMAPIError_Error -v
go test ./internal/llm/... -run TestProgressManager_UpdateProgressWithLLMError -v
go test ./internal/llm/... -run TestAnalyzeAndStore -v

# Test groups of related functionality
go test ./internal/llm/... -run "TestOpenRouter|TestLLMAPI" -v  # Test all OpenRouter and LLMAPI tests
go test ./internal/llm/... -run "Error" -v  # Test all error-related tests
```

#### Edge Case Coverage
Extend existing tests to cover specific edge cases:
- Missing `Retry-After` header in rate limit responses
- Malformed JSON in error responses
- Empty error messages
- Mixed error types (e.g., streaming + rate limit)
- Race conditions in error handling (use `-race` flag)
- Partial failures (some models succeed, others fail)

Example test cases to add:
```go
{
    name:           "Mixed Error Types",
    statusCode:     429,
    responseBody:   `{"error":{"message":"Rate limit exceeded during streaming","type":"mixed_error"}}`,
    headers:        map[string]string{"Retry-After": "10"},
    expectedType:   ErrTypeRateLimit, // Rate limit takes precedence
    expectedRetry:  10,
    expectedStatus: 429,
},
{
    name:           "Partial JSON with Error Message",
    statusCode:     500,
    responseBody:   `{"error":{"messa`,  // Truncated JSON
    expectedType:   ErrTypeUnknown,
    expectedRetry:  0,
    expectedStatus: 500,
},
{
    name:           "Unicode Characters in Error",
    statusCode:     401,
    responseBody:   `{"error":{"message":"无效的 API 密钥","type":"authentication_error"}}`,
    expectedType:   ErrTypeAuthentication,
    expectedRetry:  0,
    expectedStatus: 401,
}
```

#### Comprehensive Suite Validation
After all fixes are implemented, run the full test suite:

```bash
# Run all tests in the LLM package with code coverage
go test ./internal/llm/... -coverprofile=cover.out -v

# View coverage report
go tool cover -html=cover.out -o coverage.html

# Run with race detection
go test ./internal/llm/... -race -v

# Run with different Go build tags
go test ./internal/llm/... -tags=integration -v  # If you have integration-specific tests
```

**Specific Coverage Goals**:
- Error type detection code: 100% coverage
- Error formatting code: 100% coverage
- Error propagation paths: 90% coverage
- Edge cases (malformed responses): 85% coverage

**Verification Checklist**:
- [ ] All previously failing tests now pass
- [ ] Code coverage for error handling paths exceeds 85%
- [ ] No regressions in previously passing tests
- [ ] All error types have dedicated test cases
- [ ] Race conditions are tested with `-race` flag
- [ ] Concurrent error handling is verified

### 2. Integration Testing

#### Manual API Error Simulation

Use the test command to simulate OpenRouter API errors with different models and verify proper error propagation:

```bash
# Rate limit error test
LLM_TEST_SIMULATE_ERROR=rate_limit go run ./cmd/test_llm/main.go -article-id 123 -model meta-llama/llama-3-8b

# Authentication error test
LLM_TEST_SIMULATE_ERROR=authentication go run ./cmd/test_llm/main.go -article-id 123 -model gpt-4

# Credits error test
LLM_TEST_SIMULATE_ERROR=credits go run ./cmd/test_llm/main.go -article-id 123 -model claude-3-opus

# Streaming error test
LLM_TEST_SIMULATE_ERROR=streaming go run ./cmd/test_llm/main.go -article-id 123 -model mistral-large

# Test retry behavior with fallback models
LLM_TEST_SIMULATE_ERROR=rate_limit LLM_TEST_RETRY_MODELS=true go run ./cmd/test_llm/main.go -article-id 123
```

**Expected error output format for each error type**:
```
Rate Limit Error:
ERROR: LLM API Error (rate_limit): LLM rate limit exceeded: Rate limit exceeded [HTTP 429]
Retry recommended after: 30 seconds
Fallback models attempted: 2

Authentication Error:
ERROR: LLM API Error (authentication): LLM authentication failed: Invalid API key [HTTP 401]
Check your API key configuration in .env file

Credits Error:
ERROR: LLM API Error (credits): LLM credits exhausted: Insufficient credits [HTTP 402]
Please top up your account at openrouter.ai
```

#### Live API Testing

Test with actual API using invalid configurations to generate real errors:

```bash
# Test with invalid API key to trigger authentication error
LLM_API_KEY=invalid-key go run ./cmd/test_llm/main.go -article-id 123

# Test with simultaneous requests to trigger rate limiting
./scripts/test_rate_limit.sh  # Create a script that launches 10 concurrent requests

# Test with a model that doesn't exist to test error handling
MODEL_OVERRIDE=nonexistent-model go run ./cmd/test_llm/main.go -article-id 123

# Test with specific problematic articles from production
go run ./cmd/test_llm/main.go -article-id 456789  # Known problematic article
```

**Validation points:**
- Error message format is consistent and human-readable
- Rate limit errors include Retry-After information
- Authentication errors provide guidance on fixing API keys
- All errors are properly logged with context
- Metrics are incremented for each error type

#### UI Error Reporting Validation

Create different test scenarios to validate UI error reporting:

1. Start the server with modified LLM service that simulates errors:
   ```bash
   LLM_SIMULATION_MODE=errors go run ./cmd/server/main.go
   ```

2. Access the following endpoints for UI error testing:
   - `GET /api/v1/articles/123/analyze?simulate=rate_limit`
   - `GET /api/v1/articles/123/analyze?simulate=auth_error`
   - `GET /api/v1/articles/123/analyze?simulate=credits`
   - `GET /api/v1/articles/123/analyze?simulate=streaming`
   - `GET /api/v1/articles/123/analyze?simulate=random_errors` (randomly mix error types)

3. Check server-sent events endpoint for error reporting:
   - `GET /api/v1/progress/123` (verify error details are included)
   - `GET /api/v1/progress/all` (verify multiple errors are tracked correctly)

4. Validate frontend rendering:
   - Error messages are displayed in the UI
   - Rate limit errors show countdown timer
   - Appropriate UI feedback (e.g., red highlight, warning icon)
   - Error details expandable for debugging

**Error simulation script**:
Create a script that generates all error types in sequence:
```bash
#!/bin/bash
# test_all_error_types.sh
for error_type in rate_limit auth_error credits streaming unknown; do
  echo "Testing $error_type error..."
  curl "http://localhost:8080/api/v1/articles/123/analyze?simulate=$error_type"
  echo -e "\n---\n"
  sleep 2
done
```

**Verification Checklist**:
- [ ] All error types are properly propagated to API responses
- [ ] Error details include appropriate information (type, message, retry-after)
- [ ] Progress tracking updates correctly with error information
- [ ] Fallback mechanisms work as expected for rate limiting
- [ ] Frontend UI displays appropriate error messages
- [ ] API responses use correct HTTP status codes for each error type
- [ ] Server-sent events include detailed error information
- [ ] Error information is properly sanitized (no API keys exposed)

### 3. Code Review Verification

#### Error Type Consistency

Review all OpenRouter error type usages throughout the codebase:
```bash
# Find all usages of error type constants
grep -r "ErrType" --include="*.go" internal/

# Find all error creation points
grep -r "LLMAPIError{" --include="*.go" internal/

# Find all places where errors are handled
grep -r "errors.As" --include="*.go" internal/
grep -r "switch.*ErrorType" --include="*.go" internal/

# Find all API response mappings
grep -r "case.*ErrType" --include="*.go" internal/api/
```

Create a comprehensive mapping table:
| HTTP Status | Error Type Constant | Error Code String | User-facing Message |
|-------------|--------------------|--------------------|----------------------|
| 401 | ErrTypeAuthentication | "authentication" | "API key is invalid" |
| 402 | ErrTypeCredits | "credits" | "Account has insufficient credits" |
| 429 | ErrTypeRateLimit | "rate_limit" | "Too many requests" |
| 503 | ErrTypeStreaming | "streaming" | "Streaming connection failed" |

Verify that:
- All HTTP status codes map to consistent error types
- All error type string values match the constants
- No hardcoded error type strings exist in the codebase
- All error mappings in API responses use the correct types
- All code paths correctly handle each error type

#### Error Message Quality Assessment

Create a comprehensive document listing all possible error messages:

1. Extract all error message templates:
   ```bash
   grep -r "fmt.Sprintf" --include="*.go" internal/llm/ | grep "Error\|error"
   grep -r "return.*Error" --include="*.go" internal/llm/
   ```

2. Review each message for:
   - Human readability (clear explanations)
   - Consistency in formatting
   - Appropriate level of technical detail
   - Actionable information for users
   - Internationalization considerations

3. Create error message standardization guidelines:
   - Format: "[System component]: [What happened] because [reason]. [Action to take]."
   - Example: "LLM service: Rate limit exceeded because too many concurrent requests. Retry after 30 seconds."

Error message assessment table:
| Error Source | Current Message | Assessment | Improved Message |
|--------------|-----------------|------------|------------------|
| Rate Limit | "LLM rate limit exceeded: Rate limit exceeded" | Redundant, unclear action | "LLM service: Rate limit reached. Please wait 30 seconds and try again." |
| Auth Error | "LLM authentication failed: Invalid token" | Clear but not actionable | "LLM service: Authentication failed. Please check your API key in the .env file." |
| Credits | "LLM credits exhausted: Insufficient credits" | Unclear next steps | "LLM service: Your account has insufficient credits. Please top up at openrouter.ai." |
| Streaming | "LLM streaming failed: Streaming error" | Too generic | "LLM service: Streaming connection interrupted. Check network connectivity or try a non-streaming endpoint." |

#### Logging and Metrics Verification

For each error path, implement a systematic verification approach:

1. Create an error simulation test harness:
   ```go
   // error_simulation_test.go
   func TestErrorLoggingAndMetrics(t *testing.T) {
       errorTypes := []OpenRouterErrorType{
           ErrTypeRateLimit,
           ErrTypeAuthentication,
           ErrTypeCredits,
           ErrTypeStreaming,
           ErrTypeUnknown,
       }
       
       for _, errType := range errorTypes {
           t.Run(string(errType), func(t *testing.T) {
               // Set up metrics recorder and log capture
               metrics := setupMetricsRecorder()
               logs := captureLogOutput()
               
               // Trigger error
               triggerErrorOfType(errType)
               
               // Verify metrics
               assertMetricIncremented(t, metrics, string(errType))
               
               // Verify logs
               assertLogContains(t, logs, string(errType))
               assertLogContainsSensitiveInfo(t, logs, false)
           })
       }
   }
   ```

2. Create validation script to trigger each error and verify logs:
   ```bash
   #!/bin/bash
   # verify_error_logs.sh
   
   for error_type in rate_limit authentication credits streaming unknown; do
     echo "Verifying logs for $error_type error..."
     LLM_TEST_SIMULATE_ERROR=$error_type go run ./cmd/test_llm/main.go -article-id 123 2>&1 | tee logs_$error_type.txt
     
     # Check for sensitive information
     if grep -q "sk-" logs_$error_type.txt || grep -q "or-" logs_$error_type.txt; then
       echo "WARNING: Possible API key in logs!"
     fi
     
     # Check for appropriate error context
     if ! grep -q "$error_type" logs_$error_type.txt; then
       echo "ERROR: Log missing error type information"
     fi
     
     echo -e "Done\n---\n"
   done
   ```

3. Verify metrics collection with Prometheus:
   ```bash
   # Start server with metrics enabled
   METRICS_ENABLED=true go run ./cmd/server/main.go
   
   # Generate errors of each type
   ./scripts/generate_all_error_types.sh
   
   # Check metrics endpoint
   curl http://localhost:8080/metrics | grep llm_
   ```

**Verification Checklist**:
- [ ] Error types are consistent across all OpenRouter interactions
- [ ] Error messages follow standardized format and are user-friendly
- [ ] Appropriate logging level used for each error type
- [ ] Metrics are incremented for each error occurrence
- [ ] Log messages contain sufficient context for debugging
- [ ] No sensitive information (like API keys) appears in logs or errors
- [ ] All error paths are instrumented with metrics
- [ ] Log levels are appropriate (warnings vs. errors)
- [ ] All errors are correctly categorized

### 4. Performance Impact Assessment

#### Error Handling Overhead

Measure the performance impact of enhanced error handling:
```bash
# Benchmark tests before changes
go test ./internal/llm/... -bench=. -run=^$ -benchmem > bench_before.txt

# Benchmark tests after changes
go test ./internal/llm/... -bench=. -run=^$ -benchmem > bench_after.txt

# Compare results
benchcmp bench_before.txt bench_after.txt
```

Create focused benchmarks for error handling paths:
```go
func BenchmarkErrorHandling(b *testing.B) {
    // Setup test cases
    testCases := []struct{
        name string
        statusCode int
        responseBody string
    }{
        {"RateLimit", 429, `{"error":{"message":"Rate limit exceeded"}}`},
        {"Authentication", 401, `{"error":{"message":"Invalid API key"}}`},
        {"MalformedJSON", 400, `{not valid json}`},
        {"EmptyResponse", 500, ``},
    }
    
    for _, tc := range testCases {
        b.Run(tc.name, func(b *testing.B) {
            // Create response object
            resp := createTestResponse(tc.statusCode, tc.responseBody)
            
            // Run benchmark
            for i := 0; i < b.N; i++ {
                _ = formatHTTPError(resp)
            }
        })
    }
}
```

#### Resource Utilization Under Error Conditions

Monitor resource usage during error scenarios:
- Memory usage when many errors are being tracked
- CPU overhead of error formatting and logging
- Connection pool impacts during rate limiting

Create load test script to generate sustained errors:
```bash
#!/bin/bash
# error_load_test.sh

# Start server with metrics
METRICS_ENABLED=true go run ./cmd/server/main.go &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Generate continuous errors for 5 minutes
START_TIME=$(date +%s)
END_TIME=$((START_TIME + 300))

while [ $(date +%s) -lt $END_TIME ]; do
  # Generate random error type
  ERROR_TYPES=("rate_limit" "authentication" "credits" "streaming" "unknown")
  RANDOM_ERROR=${ERROR_TYPES[$RANDOM % ${#ERROR_TYPES[@]}]}
  
  # Make request
  curl -s "http://localhost:8080/api/v1/articles/123/analyze?simulate=$RANDOM_ERROR" > /dev/null
  
  # Small sleep to avoid overwhelming
  sleep 0.1
done

# Collect metrics
curl -s http://localhost:8080/metrics > metrics_after_load.txt

# Check resource usage
ps -o pid,pcpu,pmem,vsz,rss -p $SERVER_PID

# Cleanup
kill $SERVER_PID
```

**Verification Checklist**:
- [ ] Error handling adds minimal overhead (<5% in benchmarks)
- [ ] No memory leaks during sustained error conditions
- [ ] System remains stable during high error rates
- [ ] Error rate metrics are accurately tracked
- [ ] Performance impact of detailed error reporting is acceptable
- [ ] Resource usage remains stable during error storms

## Post-Implementation Improvements

1. **Error Documentation**:
   - Update API documentation with examples of error responses
   - Add troubleshooting guide for common LLM errors

2. **Monitoring Enhancements**:
   - Add Prometheus dashboard for error type distribution
   - Configure alerts for sustained error rates

3. **Graceful Degradation**:
   - Implement circuit breakers for LLM service failures
   - Add fallback strategies for each error type 