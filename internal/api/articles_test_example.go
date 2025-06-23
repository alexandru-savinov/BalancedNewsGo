package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	testingutils "github.com/alexandru-savinov/BalancedNewsGo/internal/testing"
)

// TestArticlesAPI demonstrates API testing with the new testing infrastructure
func TestArticlesAPI(t *testing.T) {
	// Setup test database
	config := testingutils.DatabaseTestConfig{
		UseSQLite:      true,
		SQLiteInMemory: true,
	}

	testDB := testingutils.SetupTestDatabase(t, config)
	defer func() { _ = testDB.Cleanup() }()

	t.Run("GET /api/articles", func(t *testing.T) {
		// Create a test request
		req := httptest.NewRequest("GET", "/api/articles", nil)
		w := httptest.NewRecorder()

		// Mock handler for testing
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"articles": [], "total": 0}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		})

		// Execute request
		handler.ServeHTTP(w, req)

		// Validate response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})
}

// TestMockAPIHandlers demonstrates mock handler usage
func TestMockAPIHandlers(t *testing.T) {
	// Create mock handler
	mockHandler := testingutils.NewMockHandler()

	// Add mock responses
	mockHandler.AddResponse("/api/articles", http.StatusOK, map[string]interface{}{
		"articles": []interface{}{},
		"total":    0,
	})

	// Test server with mock handler
	server := httptest.NewServer(mockHandler)
	defer server.Close()

	// Test articles endpoint
	resp, err := http.Get(server.URL + "/api/articles")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Logf("Warning: failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify requests were captured
	requests := mockHandler.GetRequests()
	if len(requests) != 1 {
		t.Errorf("Expected 1 request, got %d", len(requests))
	}
}
