package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMServer creates a mock HTTP server for LLM API testing
func MockLLMServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *resty.Client) {
	server := httptest.NewServer(handler)
	client := resty.New()
	client.SetBaseURL(server.URL)

	// Log server URL for debugging
	t.Logf("Mock server URL: %s", server.URL)

	return server, client
}

// TestHTTPLLMServiceWithMockServer tests the HTTP LLM service with a mock server
func TestHTTPLLMServiceWithMockServer(t *testing.T) {
	// Define test cases with different responses
	testCases := []struct {
		name           string
		responseBody   string
		responseStatus int
		expectError    bool
		expectedScore  float64
		expectedConf   float64
		description    string
	}{
		{
			name: "Successful response",
			responseBody: `{
				"id": "test-id",
				"object": "chat.completion",
				"created": 1714349818,
				"model": "meta-llama/llama-4-maverick",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "The article has a left-leaning bias. Score: -0.7\\nConfidence: 0.9\\n"+
								"Reasoning: The article emphasizes progressive viewpoints and criticizes conservative positions."
						},
						"finish_reason": "stop"
					}
				]
			}`,
			responseStatus: http.StatusOK,
			expectError:    false,
			expectedScore:  -0.7,
			expectedConf:   0.9,
			description:    "Valid response with score and confidence extracted successfully",
		},
		{
			name: "Response with missing content",
			responseBody: `{
				"id": "test-id",
				"object": "chat.completion",
				"created": 1714349818,
				"model": "meta-llama/llama-4-maverick",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant"
						},
						"finish_reason": "stop"
					}
				]
			}`,
			responseStatus: http.StatusOK,
			expectError:    true,
			expectedScore:  0,
			expectedConf:   0,
			description:    "Response missing content should cause an error",
		},
		{
			name: "Response with no score",
			responseBody: `{
				"id": "test-id",
				"object": "chat.completion",
				"created": 1714349818,
				"model": "meta-llama/llama-4-maverick",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "I cannot determine a bias score for this content."
						},
						"finish_reason": "stop"
					}
				]
			}`,
			responseStatus: http.StatusOK,
			expectError:    true,
			expectedScore:  0,
			expectedConf:   0,
			description:    "Response without a score should cause an error",
		},
		{
			name: "Rate limit error",
			responseBody: `{
				"error": {
					"message": "Rate limit exceeded, please try again later",
					"type": "rate_limit_error"
				}
			}`,
			responseStatus: http.StatusTooManyRequests,
			expectError:    true,
			expectedScore:  0,
			expectedConf:   0,
			description:    "Rate limit error should trigger backup key usage",
		},
		{
			name: "Server error",
			responseBody: `{
				"error": {
					"message": "Internal server error",
					"type": "server_error"
				}
			}`,
			responseStatus: http.StatusInternalServerError,
			expectError:    true,
			expectedScore:  0,
			expectedConf:   0,
			description:    "Server error should be returned",
		},
		{
			name:           "Malformed JSON response",
			responseBody:   `{"this is not valid json`,
			responseStatus: http.StatusOK,
			expectError:    true,
			expectedScore:  0,
			expectedConf:   0,
			description:    "Malformed JSON should cause an error",
		},
		{
			name: "Response with invalid score format",
			responseBody: `{
				"id": "test-id",
				"object": "chat.completion",
				"created": 1714349818,
				"model": "meta-llama/llama-4-maverick",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "The article has a left-leaning bias. Score: not-a-number\\nConfidence: 0.9\\nReasoning: Invalid score format."
						},
						"finish_reason": "stop"
					}
				]
			}`,
			responseStatus: http.StatusOK,
			expectError:    true,
			expectedScore:  0,
			expectedConf:   0,
			description:    "Invalid score format should cause an error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server that returns the predefined response
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				// Verify that required headers are present
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Content-Type header should be application/json")
				assert.NotEmpty(t, r.Header.Get("Authorization"), "Authorization header should be present")

				// Check if request body contains expected fields
				var requestBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				require.NoError(t, err, "Request body should be valid JSON")
				require.Contains(t, requestBody, "model", "Request should contain model field")
				require.Contains(t, requestBody, "messages", "Request should contain messages field")

				// Set response status and body
				w.WriteHeader(tc.responseStatus)
				_, _ = fmt.Fprintln(w, tc.responseBody)
			}

			server, client := MockLLMServer(t, mockHandler)
			defer server.Close()

			// Create the service with mock client
			service := NewHTTPLLMService(client, "test-primary-key", "test-backup-key", server.URL)

			// Create a simple article for testing
			article := &db.Article{
				ID:      1,
				Title:   "Test Article",
				Content: "This is a test article content for LLM scoring.",
			}

			// Create a simple prompt variant for testing
			promptVariant := PromptVariant{
				Model:    "meta-llama/llama-4-maverick",
				Template: "Please analyze the bias of the following article: {{.Content}}",
			}

			// Call the service
			score, confidence, err := service.ScoreContent(context.Background(), promptVariant, article)

			// Check results
			if tc.expectError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
				assert.InDelta(t, tc.expectedScore, score, 0.001, "Score should match expected value")
				assert.InDelta(t, tc.expectedConf, confidence, 0.001, "Confidence should match expected value")
			}
		})
	}
}

// TestHTTPLLMServiceWithBackupKey tests the fallback to backup key when rate limited
func TestHTTPLLMServiceWithBackupKey(t *testing.T) {
	// Create a mock server that simulates rate limit on primary key but success on backup key
	var requestCount int

	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if strings.Contains(authHeader, "test-primary-key") {
			// Simulate rate limit on primary key
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprintln(w, `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`)
		} else if strings.Contains(authHeader, "test-backup-key") {
			// Success on backup key
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{
				"id": "backup-response",
				"object": "chat.completion",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "The article has a right-leaning bias. Score: 0.6\\nConfidence: 0.85\\nReasoning: Backup key analysis."
						}
					}
				]
			}`)
		} else {
			// Invalid key
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprintln(w, `{"error": {"message": "Invalid API key", "type": "auth_error"}}`)
		}

		requestCount++
	}

	server, client := MockLLMServer(t, mockHandler)
	defer server.Close()

	// Create the service with mock client
	service := NewHTTPLLMService(client, "test-primary-key", "test-backup-key", server.URL)

	// Create a simple article for testing
	article := &db.Article{
		ID:      1,
		Title:   "Test Article",
		Content: "This is a test article content for LLM scoring.",
	}

	// Create a simple prompt variant for testing
	promptVariant := PromptVariant{
		Model:    "meta-llama/llama-4-maverick",
		Template: "Please analyze the bias of the following article: {{.Content}}",
	}

	// Call the service (should try primary key, fail with rate limit, then succeed with backup key)
	score, confidence, err := service.ScoreContent(context.Background(), promptVariant, article)

	// Assertions
	assert.NoError(t, err, "Service should succeed with backup key")
	assert.InDelta(t, 0.6, score, 0.001, "Score should match expected value from backup key response")
	assert.InDelta(t, 0.85, confidence, 0.001, "Confidence should match expected value from backup key response")
	assert.Equal(t, 2, requestCount, "Service should have made exactly 2 requests (1 for primary key, 1 for backup key)")
}

// TestHTTPLLMServiceWithServerErrors tests handling of various server errors
func TestHTTPLLMServiceWithServerErrors(t *testing.T) {
	// Test cases for different server error scenarios
	testCases := []struct {
		name           string
		serverBehavior func(http.ResponseWriter, *http.Request)
		description    string
	}{
		{
			name: "Connection reset by peer",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				// Simulate connection reset by peer
				if conn, ok := w.(http.Hijacker); ok {
					netConn, _, _ := conn.Hijack()
					_ = netConn.Close() // Abruptly close the connection
				} else {
					t.Fatal("Response writer does not support hijacking")
				}
			},
			description: "Connection reset errors should be handled gracefully",
		},
		{
			name: "Timeout",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				// In a real test, we'd use time.Sleep but in this mock we'll just return an error
				w.WriteHeader(http.StatusGatewayTimeout)
				_, _ = fmt.Fprintln(w, `{"error": {"message": "Request timed out", "type": "timeout_error"}}`)
			},
			description: "Timeout errors should be handled gracefully",
		},
		{
			name: "Empty response",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				// Return empty response
				w.WriteHeader(http.StatusOK)
			},
			description: "Empty responses should be handled gracefully",
		},
		{
			name: "Non-JSON response",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintln(w, "This is not a JSON response")
			},
			description: "Non-JSON responses should be handled gracefully",
		},
		{
			name: "Missing required fields",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintln(w, `{"id": "missing-fields-response", "object": "chat.completion"}`)
			},
			description: "Responses missing required fields should be handled gracefully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server, client := MockLLMServer(t, tc.serverBehavior)
			defer server.Close()

			// Create the service with mock client
			service := NewHTTPLLMService(client, "test-key", "", server.URL)

			// Create a simple article for testing
			article := &db.Article{
				ID:      1,
				Title:   "Test Article",
				Content: "This is a test article content for LLM scoring.",
			}

			// Create a simple prompt variant for testing
			promptVariant := PromptVariant{
				Model:    "meta-llama/llama-4-maverick",
				Template: "Please analyze the bias of the following article: {{.Content}}",
			}

			// Call the service - we expect it to handle errors gracefully
			_, _, err := service.ScoreContent(context.Background(), promptVariant, article)
			assert.Error(t, err, tc.description)

			// We don't test for specific error messages as they could change,
			// but we do ensure the function returns an error when expected
		})
	}
}
