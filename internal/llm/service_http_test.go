package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPLLMServiceUsesOpenRouterURL verifies HTTPLLMService uses the OpenRouter endpoint and parses the response
func TestHTTPLLMServiceUsesOpenRouterURL(t *testing.T) {
	var called bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("Expected path /api/v1/chat/completions, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"score\":0.5,\"explanation\":\"ok\",\"confidence\":0.9}"}}]}`))
	}))
	defer ts.Close()

	client := resty.New()
	client.SetBaseURL(ts.URL)
	// Updated to match NewHTTPLLMService signature
	svc := NewHTTPLLMService(client, "dummy-key", "backup-key", ts.URL+"/api/v1/chat/completions")

	resp, err := svc.callLLMAPIWithKey("openai/gpt-3.5-turbo", "prompt", "dummy-key")
	if err != nil {
		t.Fatalf("callLLMAPIWithKey failed: %v", err)
	}
	if !called {
		t.Errorf("Expected test server to be called")
	}
	body := resp.String()
	if !strings.Contains(body, "score") {
		t.Errorf("Expected response to contain 'score', got %s", body)
	}
}

// TestLLMAPIError_Error tests the Error method of the LLMAPIError type
func TestLLMAPIError_Error(t *testing.T) {
	testCases := []struct {
		name         string
		apiError     LLMAPIError
		expectedText string
	}{
		{
			name: "Standard API error",
			apiError: LLMAPIError{
				Message:      "Invalid token",
				StatusCode:   401,
				ResponseBody: `{"error": "Unauthorized"}`,
			},
			expectedText: "LLM API Error (unknown): Invalid token (status 401)",
		},
		{
			name: "Rate limit error",
			apiError: LLMAPIError{
				Message:      "Rate limit exceeded",
				StatusCode:   429,
				ResponseBody: `{"error": "Too many requests"}`,
			},
			expectedText: "LLM API Error (unknown): Rate limit exceeded (status 429)",
		},
		{
			name: "Server error",
			apiError: LLMAPIError{
				Message:      "Internal server error",
				StatusCode:   500,
				ResponseBody: `{"error": "Internal server error"}`,
			},
			expectedText: "LLM API Error (unknown): Internal server error (status 500)",
		},
		{
			name: "Empty message",
			apiError: LLMAPIError{
				Message:      "",
				StatusCode:   400,
				ResponseBody: `{}`,
			},
			expectedText: "LLM API Error (unknown):  (status 400)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errorText := tc.apiError.Error()
			assert.Equal(t, tc.expectedText, errorText)
		})
	}
}

func TestOpenRouterErrorHandling(t *testing.T) {
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
			client := resty.New()
			service := &HTTPLLMService{
				client:  client,
				baseURL: server.URL,
				apiKey:  "test-key",
			}

			// Create test article and prompt
			article := &db.Article{
				ID:      1,
				Content: "test content",
			}

			promptVariant := PromptVariant{
				Template: "system prompt\n\nuser prompt",
				Model:    "test-model",
			}

			// Call API
			_, _, err := service.ScoreContent(context.Background(), promptVariant, article)

			// Assert error type
			require.Error(t, err, "Expected an error to be returned")

			// Check error details
			llmErr, ok := err.(LLMAPIError)
			require.True(t, ok, "Expected error to be of type LLMAPIError, got %T", err)

			assert.Equal(t, tc.expectedType, llmErr.ErrorType, "Error type mismatch")
			assert.Equal(t, tc.expectedRetry, llmErr.RetryAfter, "Retry-After value mismatch")
			assert.Equal(t, tc.expectedStatus, llmErr.StatusCode, "Status code mismatch")
		})
	}
}

func TestSanitizeResponse(t *testing.T) {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeResponse(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatHTTPError(t *testing.T) {
	testCases := []struct {
		name          string
		responseBody  string
		statusCode    int
		headers       map[string]string
		expectedType  OpenRouterErrorType
		expectedError string
	}{
		{
			name:          "Rate Limit With Details",
			responseBody:  `{"error":{"message":"Rate limit exceeded","type":"rate_limit_error","code":"too_many_requests"}}`,
			statusCode:    429,
			headers:       map[string]string{"Retry-After": "60"},
			expectedType:  ErrTypeRateLimit,
			expectedError: "LLM rate limit exceeded: Rate limit exceeded",
		},
		{
			name:          "Empty Response",
			responseBody:  ``,
			statusCode:    500,
			expectedType:  ErrTypeUnknown,
			expectedError: "LLM service error: 500 Internal Server Error",
		},
		{
			name:          "Complex Nested Error",
			responseBody:  `{"error":{"message":"Complex error","details":{"reason":"Some technical reason"}}}`,
			statusCode:    400,
			expectedType:  ErrTypeUnknown,
			expectedError: "LLM service error: Complex error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a resty response for testing
			client := resty.New()
			resp := &resty.Response{
				RawResponse: &http.Response{
					StatusCode: tc.statusCode,
					Header:     http.Header{},
					Status:     http.StatusText(tc.statusCode),
				},
				Request: client.R(),
			}
			// Set the response body
			resp.SetBody([]byte(tc.responseBody))

			// Add headers
			for k, v := range tc.headers {
				resp.RawResponse.Header.Add(k, v)
			}

			// Call formatHTTPError
			err := formatHTTPError(resp)

			// Check error type
			llmErr, ok := err.(LLMAPIError)
			require.True(t, ok, "Expected LLMAPIError but got %T", err)

			// Verify error properties
			assert.Equal(t, tc.expectedType, llmErr.ErrorType)
			assert.Equal(t, tc.statusCode, llmErr.StatusCode)
			assert.Contains(t, llmErr.Message, tc.expectedError)
		})
	}
}
