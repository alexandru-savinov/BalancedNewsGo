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