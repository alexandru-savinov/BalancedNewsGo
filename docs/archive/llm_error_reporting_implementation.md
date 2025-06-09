# LLM Error Reporting Implementation

## Overview

This PR implements the [LLM Error Reporting Improvement Plan](llm_error_reporting_improvement.md) to enhance the error handling and reporting for LLM service errors in the NewsBalancer API. The implementation ensures that specific LLM errors are properly propagated to API clients with detailed information, correct HTTP status codes, and actionable guidance.

## Changes Made

### 1. Error Propagation in Handlers

- Updated `reanalyzeHandler` to propagate the actual LLM error (`healthErr`) to `RespondError` instead of using the generic `ErrLLMUnavailable` error.
- Improved error context for configuration loading failures.

### 2. Enhanced Error Response Structure

- Added provider information to LLM error responses.
- Added a correlation ID field to help with debugging.
- Included user-friendly recommended actions based on error type.
- Standardized error type field naming from `llm_error_type` to `error_type`.

### 3. Detailed Metrics Tracking

- Added a new Prometheus metric `llm_api_errors_total` with dimensions for:
  - Provider (e.g., openrouter)
  - Model
  - Error type
  - Status code
- Updated the `RespondError` function to track these metrics for all LLM errors.

### 4. OpenAPI Documentation Updates

- Updated Swagger documentation for the `/api/llm/reanalyze/{id}` endpoint to include specific status codes:
  - 401 for authentication errors
  - 402 for payment/credits errors
  - 429 for rate limiting
  - 503 for service unavailability and streaming errors
- Added detailed descriptions for each error type.

### 5. Integration Testing

- Added a new test case `TestReanalyzeEndpointLLMErrorPropagation` to verify that:
  - LLM authentication errors are properly propagated with status code 401
  - Response contains detailed error information including provider and error type
  - Response includes the recommended action

## Example Error Response

Before:
```json
{
  "success": false,
  "error": {
    "code": "llm_service_error",
    "message": "LLM service unavailable"
  }
}
```

After:
```json
{
  "success": false,
  "error": {
    "code": "llm_service_error",
    "message": "LLM service authentication failed",
    "details": {
      "provider": "openrouter",
      "model": "gpt-4",
      "llm_status_code": 401,
      "llm_message": "Invalid API key",
      "error_type": "authentication",
      "retry_after": null,
      "correlation_id": "req_123456"
    },
    "recommended_action": "Contact administrator to update API credentials"
  }
}
```

## Benefits

1. **Better Debugging**: Operators can quickly identify the root cause of LLM errors from the API response.
2. **Improved Client Experience**: Frontend applications can provide more informative error messages to users.
3. **Actionable Responses**: Recommended actions guide clients on how to resolve or handle the error.
4. **Enhanced Monitoring**: Detailed metrics allow for better tracking and alerting on specific error types.

## Testing Notes

The changes have been verified with new integration tests that specifically check LLM error propagation through the API layer.
