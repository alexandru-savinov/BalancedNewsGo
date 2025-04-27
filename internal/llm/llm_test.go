package llm

import (
	"flag"
	"log"
	"math"
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
		log.Println("could not load .env, proceeding with defaults")
	}

	// Ensure primary key is set
	if os.Getenv("LLM_API_KEY") == "" {
		log.Println("LLM_API_KEY not set, using default test key")
		_ = os.Setenv("LLM_API_KEY", "test-key")
	}
	// Ensure secondary key is set
	if os.Getenv("LLM_API_KEY_SECONDARY") == "" {
		log.Println("LLM_API_KEY_SECONDARY not set, using default secondary key")
		_ = os.Setenv("LLM_API_KEY_SECONDARY", "test-secondary-key")
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
	tests := []struct {
		name        string
		scores      []db.LLMScore
		expected    float64
		expectPanic bool
	}{
		{
			name: "normal case - all models",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8, Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.2, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 0.6, Metadata: `{"confidence":0.85}`},
			},
			expected: 0.0, // Average of -0.8, 0.2, and 0.6
		},
		{
			name: "some models missing",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.5, Metadata: `{"confidence":0.85}`},
				{Model: "right", Score: 0.5, Metadata: `{"confidence":0.9}`},
			},
			expected: 0.0, // Average of -0.5, 0.0 (default), 0.5
		},
		{
			name: "single model",
			scores: []db.LLMScore{
				{Model: "center", Score: 0.3, Metadata: `{"confidence":0.95}`},
			},
			expected: 0.1, // Average of 0.0 (default), 0.3, 0.0 (default)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected a panic but none occurred")
					}
				}()
			}
			actual := ComputeCompositeScore(tc.scores)
			if !tc.expectPanic && math.Abs(actual-tc.expected) > 0.001 {
				t.Errorf("Expected score %v, got %v", tc.expected, actual)
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
