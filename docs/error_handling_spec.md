# News Filter API Error Handling Specification

## Overview

This document outlines the standardized error handling approach for the News Filter API. The goal is to provide consistent, informative, and actionable error responses across all API endpoints.

## Error Response Structure

All API errors follow this JSON structure:

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "A human-readable error message",
    "metadata": {
      // Optional additional context about the error
    }
  }
}
```

### Fields

- **success**: Always `false` for error responses
- **error**: Object containing error details
  - **code**: Numeric HTTP status code (matches the HTTP response status)
  - **message**: Human-readable description of the error
  - **metadata**: (Optional) Additional context-specific information about the error

## HTTP Status Codes

The API uses standard HTTP status codes:

- **400 Bad Request**: Invalid input, missing required fields, or malformed request
- **401 Unauthorized**: Authentication failure or missing authentication
- **403 Forbidden**: The request is understood but forbidden due to permissions
- **404 Not Found**: The requested resource does not exist
- **409 Conflict**: The request conflicts with the current state (e.g., duplicate resource)
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Unexpected server error
- **502 Bad Gateway**: Error communicating with an upstream service
- **503 Service Unavailable**: Service temporarily unavailable

## Error Types and Metadata

### Validation Errors

For validation errors, the `metadata` field includes details about specific validation failures:

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "Validation failed",
    "metadata": {
      "validation_errors": [
        {
          "field": "title",
          "error": "Title cannot be empty"
        },
        {
          "field": "pub_date",
          "error": "Invalid date format"
        }
      ]
    }
  }
}
```

### LLM Provider Errors

When errors occur with LLM providers, additional context is provided:

```json
{
  "success": false,
  "error": {
    "code": 502,
    "message": "Error communicating with LLM provider",
    "metadata": {
      "provider_name": "openai",
      "raw": {
        "error": {
          "message": "Rate limit exceeded",
          "type": "rate_limit_error",
          "code": "rate_limit_exceeded"
        }
      }
    }
  }
}
```

### Content Moderation Errors

When content is flagged by moderation:

```json
{
  "success": false,
  "error": {
    "code": 403,
    "message": "Content flagged by moderation",
    "metadata": {
      "reasons": ["prohibited_content"],
      "flagged_input": "The flagged portion of text...",
      "provider_name": "content-filter"
    }
  }
}
```

### Rate Limiting Errors

When rate limits are exceeded:

```json
{
  "success": false,
  "error": {
    "code": 429,
    "message": "Rate limit exceeded",
    "metadata": {
      "headers": {
        "X-RateLimit-Limit": "100",
        "X-RateLimit-Remaining": "0",
        "X-RateLimit-Reset": "1619284800000"
      }
    }
  }
}
```

## Partial Success with Warnings

For operations that succeed but with issues, the API may return a success response with warnings:

```json
{
  "success": true,
  "data": {
    // Operation result data
  },
  "warnings": [
    {
      "code": "PARTIAL_DATA",
      "message": "Some article metadata could not be retrieved"
    }
  ]
}
```

## Fallback Mechanisms

When primary operations fail but fallbacks are available:

```json
{
  "success": true,
  "data": {
    // Fallback data
    "fallback_used": true,
    "fallback_reason": "Primary scoring service unavailable"
  }
}
```

## Client Implementation Guidelines

### JavaScript Example

```javascript
async function fetchArticles() {
  try {
    const response = await fetch('/api/articles');
    const data = await response.json();
    
    if (!data.success) {
      // Handle error
      console.error(`Error ${data.error.code}: ${data.error.message}`);
      
      // Handle specific error types
      if (data.error.code === 429) {
        // Handle rate limiting
        const resetTime = data.error.metadata?.headers?.['X-RateLimit-Reset'];
        if (resetTime) {
          console.log(`Rate limit will reset at: ${new Date(parseInt(resetTime))}`);
        }
      }
      
      return null;
    }
    
    // Handle warnings if present
    if (data.warnings && data.warnings.length > 0) {
      console.warn("API warnings:", data.warnings);
    }
    
    return data.data;
  } catch (error) {
    console.error("Network or parsing error:", error);
    return null;
  }
}
```

### Go Client Example

```go
func FetchArticle(id string) (*Article, error) {
    resp, err := http.Get(fmt.Sprintf("/api/articles/%s", id))
    if err != nil {
        return nil, fmt.Errorf("network error: %w", err)
    }
    defer resp.Body.Close()
    
    var response struct {
        Success bool            `json:"success"`
        Data    *Article        `json:"data,omitempty"`
        Error   *ErrorResponse  `json:"error,omitempty"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    
    if !response.Success {
        if response.Error != nil {
            return nil, fmt.Errorf("API error %d: %s", 
                response.Error.Code, response.Error.Message)
        }
        return nil, errors.New("unknown API error")
    }
    
    return response.Data, nil
}
```

## Implementation Notes for API Developers

1. Always use the `RespondError` and `RespondSuccess` helper functions
2. Include specific error codes from the constants defined in the codebase
3. Add detailed metadata when available to help clients debug issues
4. Log all errors with appropriate context for server-side debugging
5. Use structured logging with request IDs to correlate client errors with server logs
6. For unexpected errors, provide generic messages to clients but log detailed information server-side