package llm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenRouterErrorTypes(t *testing.T) {
	// Set up test cases for different error types
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
			name:           "Streaming Error",
			statusCode:     503,
			responseBody:   `{"error":{"message":"Streaming failed","type":"streaming_error"}}`,
			expectedType:   ErrTypeStreaming,
			expectedRetry:  0,
			expectedStatus: 503,
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

			// Create a resty client that points to test server
			client := resty.New()

			// Test with the formatHTTPError function directly
			// Make actual request to server to get a real response
			resp, err := client.R().Get(server.URL)
			require.NoError(t, err, "Request to test server should succeed")

			// Process the error with formatHTTPError
			llmErr := formatHTTPError(resp)

			// Verify error details
			require.IsType(t, LLMAPIError{}, llmErr, "Expected error to be of type LLMAPIError")

			// Type assertion
			typedErr, ok := llmErr.(LLMAPIError)
			if !ok {
				t.Fatalf("Expected LLMAPIError, got %T", llmErr)
			}

			// Verify expected fields
			assert.Equal(t, tc.expectedType, typedErr.ErrorType, "Error type mismatch")
			assert.Equal(t, tc.expectedRetry, typedErr.RetryAfter, "Retry-After value mismatch")
			assert.Equal(t, tc.expectedStatus, typedErr.StatusCode, "Status code mismatch")

			// Verify error message includes the type
			assert.Contains(t, typedErr.Error(), string(typedErr.ErrorType), "Error message should include error type")
		})
	}
}

func TestSanitizeResponseFunction(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "OpenRouter API Key",
			input:    `{"error": "Invalid API key: or-abc123xyz456789defghi"}`,
			expected: `{"error": "Invalid API key: [REDACTED]"}`,
		},
		{
			name:     "OpenAI API Key",
			input:    `{"error": "Invalid API key: sk-opq987uvw654321rstxyz"}`,
			expected: `{"error": "Invalid API key: [REDACTED]"}`,
		},
		{
			name:     "Multiple API Keys",
			input:    `{"keys": ["or-abc123xyz456789defghi", "sk-opq987uvw654321rstxyz"]}`,
			expected: `{"keys": ["[REDACTED]", "[REDACTED]"]}`,
		},
		{
			name:     "No API Keys",
			input:    `{"error": "Invalid request format"}`,
			expected: `{"error": "Invalid request format"}`,
		},
		{
			name:     "Error with API Key in Message",
			input:    `{"error":{"message":"Invalid authorization header: Bearer or-123456789abcdefghijk"}}`,
			expected: `{"error":{"message":"Invalid authorization header: Bearer [REDACTED]"}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeResponse(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestErrorTypes(t *testing.T) {
	// Test the error type constants
	assert.Equal(t, OpenRouterErrorType("rate_limit"), ErrTypeRateLimit)
	assert.Equal(t, OpenRouterErrorType("authentication"), ErrTypeAuthentication)
	assert.Equal(t, OpenRouterErrorType("credits"), ErrTypeCredits)
	assert.Equal(t, OpenRouterErrorType("streaming"), ErrTypeStreaming)
	assert.Equal(t, OpenRouterErrorType("unknown"), ErrTypeUnknown)
}

func TestErrorPropagation(t *testing.T) {
	// Test that LLMAPIError works with error formatting
	testCases := []struct {
		name           string
		errorType      OpenRouterErrorType
		message        string
		statusCode     int
		expectedPrefix string
	}{
		{
			name:           "Rate Limit Error",
			errorType:      ErrTypeRateLimit,
			message:        "Rate limit exceeded",
			statusCode:     429,
			expectedPrefix: "LLM API Error (rate_limit)",
		},
		{
			name:           "Authentication Error",
			errorType:      ErrTypeAuthentication,
			message:        "Invalid API key",
			statusCode:     401,
			expectedPrefix: "LLM API Error (authentication)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := LLMAPIError{
				Message:      tc.message,
				StatusCode:   tc.statusCode,
				ResponseBody: "test response body",
				ErrorType:    tc.errorType,
			}

			// Test that Error() method includes the expected message
			errString := err.Error()
			assert.Contains(t, errString, tc.expectedPrefix, "Error string should contain the type")
			assert.Contains(t, errString, tc.message, "Error string should contain the message")
			assert.Contains(t, errString, fmt.Sprintf("%d", tc.statusCode), "Error string should contain the status code")
		})
	}
}
