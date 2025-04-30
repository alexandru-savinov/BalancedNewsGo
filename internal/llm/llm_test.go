package llm

import (
	"flag"
	"log"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
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

// TestSetHTTPLLMTimeout tests the SetHTTPLLMTimeout method of LLMClient
func TestSetHTTPLLMTimeout(t *testing.T) {
	// Create a test HTTP LLM service
	restyClient := resty.New()
	initialTimeout := 10 * time.Second
	restyClient.SetTimeout(initialTimeout)

	service := NewHTTPLLMService(restyClient, "test-key", "backup-key", "")

	// Create a test LLMClient with the HTTP LLM service
	client := &LLMClient{
		llmService: service,
	}

	// Verify initial timeout
	httpService, ok := client.llmService.(*HTTPLLMService)
	assert.True(t, ok, "Expected llmService to be HTTPLLMService")
	assert.Equal(t, initialTimeout, httpService.client.GetClient().Timeout, "Initial timeout should match")

	// Set a new timeout
	newTimeout := 20 * time.Second
	client.SetHTTPLLMTimeout(newTimeout)

	// Verify the timeout was updated
	assert.Equal(t, newTimeout, httpService.client.GetClient().Timeout, "Timeout should be updated to new value")
}

// Utility function for test to simulate loading a config
func loadTestCompositeScoreConfig() *CompositeScoreConfig {
	return &CompositeScoreConfig{
		Formula:          "average",
		Weights:          map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0},
		MinScore:         -1.0,
		MaxScore:         1.0,
		DefaultMissing:   0.0,
		HandleInvalid:    "default",
		ConfidenceMethod: "count_valid",
		MinConfidence:    0.0,
		MaxConfidence:    1.0,
		Models: []ModelConfig{
			{Perspective: "left", ModelName: "left", URL: ""},
			{Perspective: "center", ModelName: "center", URL: ""},
			{Perspective: "right", ModelName: "right", URL: ""},
		},
	}
}

// TestCompositeScoreWithConfig tests the ComputeCompositeScore function with a specific test config
func TestCompositeScoreWithConfig(t *testing.T) {
	testCases := []struct {
		name           string
		scores         []db.LLMScore
		expectedResult float64
	}{
		{
			name: "Basic average calculation",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8},
				{Model: "center", Score: 0.0},
				{Model: "right", Score: 0.8},
			},
			expectedResult: 0.0, // Average of -0.8, 0.0, and 0.8
		},
		{
			name: "Missing score uses default value (0.0)",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.5},
				{Model: "right", Score: 0.5},
			},
			expectedResult: 0.0, // Average of -0.5, 0.0 (default), and 0.5
		},
		{
			name:           "Empty scores array",
			scores:         []db.LLMScore{},
			expectedResult: 0.0, // Average of three default values
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compositeScoreConfig = loadTestCompositeScoreConfig() // Set the global config for testing
			result := ComputeCompositeScore(tc.scores)
			assert.InDelta(t, tc.expectedResult, result, 0.01, "Composite score calculation error")
		})
	}
}
