package llm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

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
		w.WriteHeader(200)
		w.Write([]byte(`{"score":0.7,"metadata":{"foo":"bar"}}`))
	}))
	defer mockServer.Close()

	client, _ := setupTestLLMClient(t)

	resp, err := client.callAPI(mockServer.URL, "test content")
	if err != nil {
		t.Fatalf("callAPI failed: %v", err)
	}
	if resp.Score != 0.7 {
		t.Errorf("Expected score 0.7, got %v", resp.Score)
	}
	if resp.Metadata["foo"] != "bar" {
		t.Errorf("Expected metadata foo=bar, got %v", resp.Metadata)
	}
}
