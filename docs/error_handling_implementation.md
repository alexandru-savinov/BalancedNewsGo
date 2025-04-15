# Error Handling Implementation

This document outlines the implementation of enhanced error handling in the News Filter application.

## Backend Implementation

### 1. Error Response Structure

We've updated the error response structure to follow industry best practices:

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

### 2. Middleware

We've implemented several middleware components:

- **ErrorHandlingMiddleware**: Catches panics and provides consistent error responses
- **RequestLoggingMiddleware**: Logs request details with sanitized sensitive information
- **RateLimitMiddleware**: Implements rate limiting with appropriate headers
- **CORSMiddleware**: Handles cross-origin resource sharing

### 3. Helper Functions

We've added specialized error response helpers:

- `RespondError`: Basic error response
- `RespondErrorWithMetadata`: Error with additional context
- `RespondValidationError`: Validation-specific errors with field details
- `RespondProviderError`: LLM provider errors with provider details
- `RespondRateLimitError`: Rate limit errors with rate limit headers
- `RespondModerationError`: Content moderation errors with flagged content details

### 4. Logging Enhancements

- Added request IDs for correlation
- Structured logging with context
- Performance tracking
- Warning level for non-critical issues
- Stack traces in development mode

## Frontend Implementation

### 1. Error Handler Class

We've created a JavaScript `ErrorHandler` class that:

- Handles API responses consistently
- Processes different error types based on status codes
- Displays user-friendly notifications
- Provides detailed error information for debugging
- Handles warnings separately from errors

### 2. Fetch Wrapper

The `fetchWithErrorHandling` method wraps the standard fetch API to:

- Parse API responses
- Extract and handle errors
- Process warnings
- Provide consistent error handling

### 3. Notification System

We've implemented a notification system that:

- Displays different types of notifications (error, warning, info)
- Shows appropriate icons and colors based on error type
- Provides detailed information when available
- Auto-dismisses warnings but keeps errors until manually dismissed
- Is responsive across device sizes

## Usage Examples

### Backend (Go)

```go
// Basic error
if err != nil {
    api.LogError("GetArticles", err)
    api.RespondError(c, api.StatusInternalServerError, "Failed to retrieve articles")
    return
}

// Validation error
if len(title) == 0 {
    validationErrors := []api.ValidationError{
        {Field: "title", Error: "Title cannot be empty"},
    }
    api.RespondValidationError(c, "Validation failed", validationErrors)
    return
}

// LLM provider error
if providerErr != nil {
    api.RespondProviderError(c, api.StatusBadGateway, 
        "Error communicating with LLM provider", 
        "openai", rawError)
    return
}
```

### Frontend (JavaScript)

```javascript
// Using the error handler
async function loadArticles() {
    try {
        const articles = await errorHandler.fetchWithErrorHandling('/api/articles');
        renderArticles(articles);
    } catch (error) {
        // Error already handled by errorHandler
        renderEmptyState();
    }
}

// Manual error handling
function validateForm() {
    const title = document.getElementById('title').value;
    if (!title) {
        errorHandler.showNotification({
            type: 'validation',
            message: 'Please enter a title',
            code: 'VALIDATION_ERROR'
        });
        return false;
    }
    return true;
}
```

## Benefits

1. **Consistency**: Standardized error handling across the application
2. **Detailed Information**: More context for debugging and user feedback
3. **Graceful Degradation**: Better handling of partial failures
4. **Improved UX**: User-friendly error messages and notifications
5. **Better Debugging**: Correlated logs and detailed error information
6. **Rate Limiting**: Protection against abuse with clear feedback
7. **Security**: Sanitized error messages that don't expose sensitive details