package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const testContent = "test content"

func setupTestLLMClient(t *testing.T) (*LLMClient, *sqlx.DB) {
	dbConn, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	_, err = dbConn.Exec(`CREATE TABLE IF NOT EXISTS llm_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id INTEGER,
		model TEXT,
		score REAL,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	client := NewLLMClient(dbConn)

	return client, dbConn
}

func TestCallAPI(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"score":0.7,"metadata":{"foo":"bar"}}`))
	}))
	defer mockServer.Close()

	client, _ := setupTestLLMClient(t)

	resp, err := client.llmService.Analyze("test content")
	if err != nil {
		t.Fatalf("callAPI failed: %v", err)
	}

	if resp.Score != 0.7 {
		t.Errorf("Expected score 0.7, got %v", resp.Score)
	}

	var meta map[string]interface{}

	err = json.Unmarshal([]byte(resp.Metadata), &meta)
	if err != nil {
		t.Errorf("Failed to parse metadata JSON: %v", err)
	} else if meta["foo"] != "bar" {
		t.Errorf("Expected metadata foo=bar, got %v", meta["foo"])
	}
}

func TestCallAPIFailure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Internal Server Error`))
	}))
	defer mockServer.Close()

	client, _ := setupTestLLMClient(t)

	_, err := client.llmService.Analyze(testContent)
	if err == nil {
		t.Errorf("Expected error on 500 response, got nil")
	}
}

func TestCallAPIAuthError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`Unauthorized`))
	}))
	defer mockServer.Close()

	client, _ := setupTestLLMClient(t)

	_, err := client.llmService.Analyze(testContent)
	if err == nil {
		t.Errorf("Expected error on 401 response, got nil")
	}
}

func TestCallAPIInvalidResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`not a json`)); err != nil {
			t.Logf("Warning: failed to write mock response: %v", err)
		}
	}))
	defer mockServer.Close()

	client, _ := setupTestLLMClient(t)

	_, err := client.llmService.Analyze(testContent)
	if err == nil {
		t.Errorf("Expected error on invalid JSON response, got nil")
	}
}

func TestCallAPIPersistence(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"score":0.9,"metadata":{"source":"mock"}}`)); err != nil {
			t.Logf("Warning: failed to write mock response: %v", err)
		}
	}))
	defer mockServer.Close()

	client, db := setupTestLLMClient(t)

	resp, err := client.llmService.Analyze(testContent)
	if err != nil {
		t.Fatalf("callAPI failed: %v", err)
	}

	// Simulate saving result
	_, err = db.Exec(`INSERT INTO llm_scores(article_id, model, score, metadata) VALUES (?, ?, ?, ?)`,
		1, "mock-model", resp.Score, `{"source":"mock"}`)
	if err != nil {
		t.Fatalf("Failed to insert LLM score: %v", err)
	}

	var count int

	err = db.Get(&count, "SELECT COUNT(*) FROM llm_scores WHERE article_id=1 AND model='mock-model'")
	if err != nil {
		t.Fatalf("Failed to query llm_scores: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 llm_score record, got %d", count)
	}
}

func TestParseBiasResult(t *testing.T) {
	contentResp := `{"bias":"left","confidence":0.85}`
	result := parseBiasResult(contentResp)

	if result.Category == "" {
		t.Errorf("Expected bias category to be parsed, got empty string")
	}
	if result.Confidence == 0 {
		t.Errorf("Expected confidence to be parsed, got 0")
	}
}

func TestHeuristicCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"This is a left-leaning statement", "left"},
		{"This is a right-leaning statement", "right"},
		{"This is a neutral statement", "neutral"},
	}

	for _, tt := range tests {
		got := heuristicCategory(tt.input)
		if got != tt.expected {
			t.Errorf("heuristicCategory(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
