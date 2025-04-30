package llm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
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
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"score\":0.5,\"explanation\":\"ok\",\"confidence\":0.9}"}}]}`))
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
			expectedText: "LLM API Error (status 401): Invalid token",
		},
		{
			name: "Rate limit error",
			apiError: LLMAPIError{
				Message:      "Rate limit exceeded",
				StatusCode:   429,
				ResponseBody: `{"error": "Too many requests"}`,
			},
			expectedText: "LLM API Error (status 429): Rate limit exceeded",
		},
		{
			name: "Server error",
			apiError: LLMAPIError{
				Message:      "Internal server error",
				StatusCode:   500,
				ResponseBody: `{"error": "Internal server error"}`,
			},
			expectedText: "LLM API Error (status 500): Internal server error",
		},
		{
			name: "Empty message",
			apiError: LLMAPIError{
				Message:      "",
				StatusCode:   400,
				ResponseBody: `{}`,
			},
			expectedText: "LLM API Error (status 400): ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errorText := tc.apiError.Error()
			assert.Equal(t, tc.expectedText, errorText)
		})
	}
}
