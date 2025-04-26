package llm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
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
