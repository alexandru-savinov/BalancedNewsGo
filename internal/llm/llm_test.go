//go:build test

package llm

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestComputeCompositeScore(t *testing.T) {
	os.Setenv("LLM_API_KEY", "test-key")

	// Add debug logging
	t.Log("Starting TestComputeCompositeScore")
	// Load configuration for debugging
	config, err := LoadCompositeScoreConfig()
	if err != nil {
		t.Logf("Warning: Failed to load composite score config: %v", err)
	} else {
		t.Logf("Loaded config with %d models", len(config.Models))
		for _, model := range config.Models {
			t.Logf("Configured model: %s", model.Perspective)
		}
	}

	testCases := []struct {
		name     string
		scores   []db.LLMScore
		expected float64
	}{
		{
			name: "All valid scores",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8},
				{Model: "center", Score: 0.0},
				{Model: "right", Score: 0.8},
			},
			expected: 0.0,
		},
		{
			name: "Missing scores",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8},
				{Model: "right", Score: 0.8},
			},
			expected: 0.0, 
		},
		{
			name: "Case insensitive models",
			scores: []db.LLMScore{
				{Model: "LEFT", Score: -0.8},
				{Model: "Center", Score: 0.0},
				{Model: "RiGhT", Score: 0.8},
			},
			expected: 0.0,
		},
		{
			name: "Invalid model names",
			scores: []db.LLMScore{
				{Model: "unknown", Score: 0.5},
				{Model: "", Score: 0.3},
			},
			expected: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := ComputeCompositeScore(tc.scores)
			if score != tc.expected {
				t.Errorf("Expected score %v, got %v", tc.expected, score)
			}
		})
	}
}

	client := NewLLMClient((*sqlx.DB)(nil))

	httpService, ok := client.llmService.(*HTTPLLMService)
	if !ok {
		t.Fatalf("Expected llmService to be *HTTPLLMService, got %T", client.llmService)
	}

	// Check primary key
	if httpService.apiKey != "test-primary-key" {
		t.Errorf("Expected primary apiKey to be 'test-primary-key', got '%s'", httpService.apiKey)
	}

	// Check backup key
	if httpService.backupKey != "test-backup-key" {
		t.Errorf("Expected backup apiKey to be 'test-backup-key', got '%s'", httpService.backupKey)
	}

	// Check base URL
	if httpService.baseURL != "https://openrouter.ai/api/v1/chat/completions" {
		t.Errorf("Expected baseURL to be OpenRouter endpoint, got '%s'", httpService.baseURL)
	}
}

func TestNewLLMClientMissingPrimaryKey(t *testing.T) {
	os.Setenv("LLM_API_KEY", "")
	os.Setenv("LLM_API_KEY_SECONDARY", "test-backup-key")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected NewLLMClient to panic with missing primary key")
		}
	}()

	NewLLMClient((*sqlx.DB)(nil))
}

func TestModelConfiguration(t *testing.T) {
	os.Setenv("LLM_API_KEY", "test-key")

	client := NewLLMClient((*sqlx.DB)(nil))

	if client.config == nil {
		t.Fatal("Expected config to be loaded, got nil")
	}

	expectedModels := map[string]string{
		"left":   "meta-llama/llama-guard-3-8b",
		"center": "mistralai/mistral-small-3.1-24b-instruct",
		"right":  "openai/gpt-4.1-nano",
	}

	for _, model := range client.config.Models {
		expectedName, ok := expectedModels[model.Perspective]
		if !ok {
			t.Errorf("Unexpected perspective in config: %s", model.Perspective)
			continue
		}
		if model.ModelName != expectedName {
			t.Errorf("For perspective %s, expected model %s, got %s",
				model.Perspective, expectedName, model.ModelName)
		}
	}
}
