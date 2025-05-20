package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRespondError_LLMErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name                  string
		err                   error
		expectedStatusCode    int
		expectedErrorCode     string
		shouldHaveRetryHeader bool
		expectedRetryValue    string
		checkErrorDetails     bool
	}{
		{
			name: "Rate Limit Error",
			err: llm.LLMAPIError{
				Message:      "Rate limit exceeded",
				StatusCode:   429,
				ResponseBody: "limit exceeded",
				ErrorType:    llm.ErrTypeRateLimit,
				RetryAfter:   30,
			},
			expectedStatusCode:    http.StatusTooManyRequests,
			expectedErrorCode:     ErrRateLimit,
			shouldHaveRetryHeader: true,
			expectedRetryValue:    "30",
			checkErrorDetails:     true,
		},
		{
			name: "Authentication Error",
			err: llm.LLMAPIError{
				Message:      "Invalid API key",
				StatusCode:   401,
				ResponseBody: "auth failed",
				ErrorType:    llm.ErrTypeAuthentication,
			},
			expectedStatusCode:    http.StatusUnauthorized,
			expectedErrorCode:     ErrLLMService,
			shouldHaveRetryHeader: false,
			checkErrorDetails:     true,
		},
		{
			name: "Credits Exhausted Error",
			err: llm.LLMAPIError{
				Message:      "Insufficient credits",
				StatusCode:   402,
				ResponseBody: "payment required",
				ErrorType:    llm.ErrTypeCredits,
			},
			expectedStatusCode:    http.StatusPaymentRequired,
			expectedErrorCode:     ErrLLMService,
			shouldHaveRetryHeader: false,
			checkErrorDetails:     true,
		},
		{
			name: "Streaming Error",
			err: llm.LLMAPIError{
				Message:      "Streaming failed",
				StatusCode:   503,
				ResponseBody: "streaming error",
				ErrorType:    llm.ErrTypeStreaming,
			},
			expectedStatusCode:    http.StatusServiceUnavailable,
			expectedErrorCode:     ErrLLMService,
			shouldHaveRetryHeader: false,
			checkErrorDetails:     true,
		},
		{
			name: "Generic LLM Error",
			err: llm.LLMAPIError{
				Message:      "Unknown error",
				StatusCode:   500,
				ResponseBody: "unknown error",
				ErrorType:    llm.ErrTypeUnknown,
			},
			expectedStatusCode:    http.StatusServiceUnavailable,
			expectedErrorCode:     ErrLLMService,
			shouldHaveRetryHeader: false,
			checkErrorDetails:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/?model=test", nil)

			// Call the function we're testing
			RespondError(c, tc.err)

			// Check status code
			assert.Equal(t, tc.expectedStatusCode, w.Code, "Status code mismatch")

			// Verify JSON response has expected structure
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json", "Content-Type should be application/json")

			// Check retry header if applicable
			if tc.shouldHaveRetryHeader {
				retryValue := w.Header().Get("Retry-After")
				assert.NotEmpty(t, retryValue, "Retry-After header should be present")
				if tc.expectedRetryValue != "" {
					assert.Equal(t, tc.expectedRetryValue, retryValue, "Retry-After value mismatch")
				}
			} else {
				assert.Empty(t, w.Header().Get("Retry-After"), "Retry-After header should not be present")
			}

			// Parse the JSON response and verify error code
			var respBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &respBody)
			assert.NoError(t, err, "Response should be valid JSON")

			errorData, ok := respBody["error"].(map[string]interface{})
			assert.True(t, ok, "Response should have error object")

			assert.Equal(t, tc.expectedErrorCode, errorData["code"], "Error code mismatch")
			assert.NotEmpty(t, errorData["message"], "Error message should not be empty")

			// For LLM API errors, check error details
			if tc.checkErrorDetails {
				assert.Contains(t, errorData["message"], "LLM", "LLM error message should mention LLM")
			}
		})
	}
}

func TestRespondError_AppErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name               string
		err                error
		expectedStatusCode int
	}{
		{
			name: "Validation Error",
			err: &apperrors.AppError{
				Code:    ErrValidation,
				Message: "Invalid input",
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "Not Found Error",
			err: &apperrors.AppError{
				Code:    ErrNotFound,
				Message: "Resource not found",
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "Generic Error",
			err:                errors.New("generic error"),
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			RespondError(c, tc.err)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		})
	}
}

func TestLogError(t *testing.T) {
	// This is more of a smoke test since we can't easily test logging output
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Test with LLM error
	llmErr := llm.LLMAPIError{
		Message:    "Test LLM error",
		StatusCode: 429,
		ErrorType:  llm.ErrTypeRateLimit,
	}
	LogError(c, llmErr, "TestOperation")

	// Test with app error
	appErr := &apperrors.AppError{
		Code:    "test_code",
		Message: "Test app error",
	}
	LogError(c, appErr, "TestOperation")

	// Test with generic error
	genericErr := errors.New("generic error")
	LogError(c, genericErr, "TestOperation")

	// Test with nil error
	LogError(c, nil, "TestOperation")
}
