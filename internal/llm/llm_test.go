package llm

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	// Locate project root by finding configs/composite_score_config.json
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get cwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(cwd, "configs", "composite_score_config.json")); err == nil {
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			log.Fatal("could not find project root (configs/composite_score_config.json)")
		}
		cwd = parent
	}
	if err := os.Chdir(cwd); err != nil {
		log.Fatalf("failed to chdir to project root: %v", err)
	}

	// Load .env file
	if err := godotenv.Load(filepath.Join(cwd, ".env")); err != nil {
		log.Fatalf("failed to load .env: %v", err)
	}

	// Ensure primary key is set
	if os.Getenv("LLM_API_KEY") == "" {
		log.Fatal("LLM_API_KEY must be set in .env for tests")
	}

	flag.Parse()
	os.Exit(m.Run())
}

// Helper to ensure config path is correct for tests
func ensureConfigPath() {
	configPath := "configs/composite_score_config.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try to find the project root
		cwd, _ := os.Getwd()
		for i := 0; i < 5; i++ {
			parent := filepath.Dir(cwd)
			candidate := filepath.Join(parent, configPath)
			if _, err := os.Stat(candidate); err == nil {
				os.Chdir(parent)
				return
			}
			cwd = parent
		}
	}
}

func TestComputeCompositeScore(t *testing.T) {
	ensureConfigPath()
	t.Setenv("LLM_API_KEY", "test-key")

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

	// Add test cases for edge scenarios
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
			expected: 0.0, // (-0.8 + 0.0 + 0.8) / 3 = 0.0
		},
		{
			name: "Missing scores",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8},
				{Model: "right", Score: 0.8},
			},
			expected: 0.0, // (-0.8 + 0.0 + 0.8) / 3 = 0.0 (center defaults to 0.0)
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
		{
			name: "Weighted mean",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8},
				{Model: "center", Score: 0.0},
				{Model: "right", Score: 0.6},
			},
			expected: (-0.8*1.0 + 0.0*1.0 + 0.6*1.0) / 3.0, // = -0.066666...
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := ComputeCompositeScore(tc.scores)
			if diff := score - tc.expected; diff < -1e-6 || diff > 1e-6 {
				t.Errorf("Expected score %.8f, got %.8f (diff %.8f)", tc.expected, score, diff)
			}
		})
	}
}

func TestLLMClientInitialization(t *testing.T) {
	primaryKey := os.Getenv("LLM_API_KEY")
	backupKey := os.Getenv("LLM_API_KEY_SECONDARY")
	t.Setenv("LLM_API_KEY", primaryKey)
	t.Setenv("LLM_API_KEY_SECONDARY", backupKey)

	if primaryKey == "" {
		t.Fatal("LLM_API_KEY must be set in .env for tests")
	}
	if backupKey == "" {
		t.Fatal("LLM_API_KEY_SECONDARY must be set in .env for tests")
	}

	client := NewLLMClient((*sqlx.DB)(nil))

	httpService, ok := client.llmService.(*HTTPLLMService)
	if !ok {
		t.Fatalf("Expected llmService to be *HTTPLLMService, got %T", client.llmService)
	}

	if httpService.apiKey != primaryKey {
		t.Errorf("Expected primary apiKey to be '%s', got '%s'", primaryKey, httpService.apiKey)
	}
	if httpService.backupKey != backupKey {
		t.Errorf("Expected backup apiKey to be '%s', got '%s'", backupKey, httpService.backupKey)
	}
	if httpService.baseURL == "" {
		t.Errorf("Expected baseURL to be set, got empty string")
	}
}

func TestNewLLMClientMissingPrimaryKey(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_API_KEY_SECONDARY", "test-backup-key")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected NewLLMClient to panic with missing primary key")
		}
	}()

	NewLLMClient((*sqlx.DB)(nil))
}

func TestModelConfiguration(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")

	client := NewLLMClient((*sqlx.DB)(nil))

	if client.config == nil {
		t.Fatal("Expected config to be loaded, got nil")
	}

	expectedModels := map[string]string{
		"left":   "meta-llama/llama-4-maverick",
		"center": "google/gemini-2.0-flash-001",
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
