package llm

import (
	"fmt"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a score with metadata containing confidence
// (Copied from deleted composite_score_fix_test.go)
func createScoreWithConfidence(model string, score float64, confidence float64) db.LLMScore {
	metadata := fmt.Sprintf(`{"confidence": %.2f}`, confidence)
	return db.LLMScore{
		ArticleID: 1, // Hardcode article ID as it's not relevant for these tests
		Model:     model,
		Score:     score,
		Metadata:  metadata,
		CreatedAt: time.Now(), // Set a consistent time for tests if needed
	}
}

// Model name constants to avoid duplication
const (
	modelLeft   = "meta-llama/llama-4-maverick"
	modelCenter = "google/gemini-2.0-flash-001"
	modelRight  = "openai/gpt-4.1-nano"
)

// ArticleScore represents a single score for an article from a specific model
type ArticleScore struct {
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
	Model      string  `json:"model"`
}

// CompositeScore represents the combined score from multiple models
type CompositeScore struct {
	Scores       map[string]*ArticleScore `json:"scores"`
	Formula      string                   `json:"formula"`
	FinalScore   float64                  `json:"final_score"`
	Perspectives []string                 `json:"perspectives"`
}

// CalculateCompositeScore calculates the final score from individual model scores
func CalculateCompositeScore(composite *CompositeScore) error {
	if composite == nil || len(composite.Scores) == 0 {
		return nil
	}

	// Simple average implementation for tests
	total := 0.0
	count := 0

	for _, score := range composite.Scores {
		if score != nil {
			total += score.Score
			count++
		}
	}

	if count > 0 {
		composite.FinalScore = total / float64(count)
	}

	return nil
}

// Helper to create a test config for model name handling tests
func createModelNameTestConfig() *CompositeScoreConfig {
	return &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "meta-llama/llama-3-maverick:123", Perspective: "left"},
			{ModelName: "google/gemini-pro", Perspective: "center"},
			{ModelName: "openai/gpt-4-turbo", Perspective: "right"},
			{ModelName: "Vendor/Legacy-Left", Perspective: "left"},
			{ModelName: " Vendor / Mixed-Case-Center ", Perspective: "center"},
		},
		Formula: "average", DefaultMissing: 0.0, HandleInvalid: "default",
		MinScore: -1.0, MaxScore: 1.0, ConfidenceMethod: "count_valid",
		MinConfidence: 0.0, MaxConfidence: 1.0,
	}
}

func TestMapModelToPerspective(t *testing.T) {
	testCfg := createModelNameTestConfig()

	testCases := []struct {
		modelName string
		expected  string
	}{
		{"meta-llama/llama-3-maverick:123", "left"},
		{"meta-llama/llama-3-maverick:latest", "left"}, // Should match base name
		{"meta-llama/llama-3-maverick", "left"},
		{"google/gemini-pro", "center"},
		{"openai/gpt-4-turbo", "right"},
		{"Vendor/Legacy-Left", "left"},
		{" Vendor / Mixed-Case-Center ", "center"}, // Should handle spacing and case
		{"unknown/model", ""},                      // Not found
		{":123", ""},                               // Invalid format
		{"", ""},                                   // Empty string
		{"left", "left"},                           // Legacy fallback
		{"CENTER", "center"},                       // Legacy fallback (case-insensitive)
	}

	for _, tc := range testCases {
		t.Run(tc.modelName, func(t *testing.T) {
			actual := MapModelToPerspective(tc.modelName, testCfg) // Pass testCfg
			assert.Equal(t, tc.expected, actual)
		})
	}

	// Test with nil config
	t.Run("Nil Config", func(t *testing.T) {
		actual := MapModelToPerspective("google/gemini-pro", nil)
		assert.Equal(t, "", actual, "Should return empty string for nil config")
	})
}

func TestComputeWithMixedModelNames(t *testing.T) {
	testCfg := createModelNameTestConfig()

	scores := []db.LLMScore{
		createScoreWithConfidence("meta-llama/llama-3-maverick:123", -0.8, 0.9), // left
		createScoreWithConfidence("google/gemini-pro", 0.1, 0.8),                // center
		createScoreWithConfidence(" Vendor / Mixed-Case-Center ", 0.3, 0.7),     // center (duplicate, lower conf)
		createScoreWithConfidence("openai/gpt-4-turbo", 0.9, 0.95),              // right
		createScoreWithConfidence("unknown/model", 0.5, 0.5),                    // unknown
	}

	// Expected score (average): Uses -0.8 (left), 0.1 (center, higher conf), 0.9 (right)
	// (-0.8 + 0.1 + 0.9) / 3 = 0.2 / 3 = 0.0667
	expectedScore := 0.0667
	expectedConfidence := 1.0 // Assumes count_valid, 3/3 perspectives found

	score, confidence, err := ComputeCompositeScoreWithConfidenceFixed(scores, testCfg) // Pass testCfg

	assert.NoError(t, err)
	assert.InDelta(t, expectedScore, score, 0.001)
	assert.InDelta(t, expectedConfidence, confidence, 0.001)
}

// TestModelNameNormalization tests the normalization of model names in different contexts
func TestModelNameNormalization(t *testing.T) {
	testCfg := createModelNameTestConfig()

	t.Run("MapModelToPerspective Normalization", func(t *testing.T) {
		assert.Equal(t, "left", MapModelToPerspective("meta-llama/llama-3-maverick:latest", testCfg))
		assert.Equal(t, "center", MapModelToPerspective(" google/gemini-pro ", testCfg))
	})

	t.Run("CompositeScore Calculation Normalization", func(t *testing.T) {
		scores := []db.LLMScore{
			createScoreWithConfidence("meta-llama/llama-3-maverick:latest", -0.5, 0.9),
			createScoreWithConfidence(" google/gemini-pro ", 0.0, 0.8),
			createScoreWithConfidence("openai/gpt-4-turbo", 0.5, 0.85),
		}
		score, _, err := ComputeCompositeScoreWithConfidenceFixed(scores, testCfg) // Pass testCfg
		assert.NoError(t, err)
		assert.InDelta(t, 0.0, score, 0.001) // (-0.5 + 0.0 + 0.5) / 3 = 0
	})
}

// TestModelNameFallbackLogic tests the fallback logic when model names aren't found in config
func TestModelNameFallbackLogic(t *testing.T) {
	// Create a test config locally instead of using a global
	testConfig := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: modelLeft, Perspective: "left"},
			{ModelName: modelCenter, Perspective: "center"},
			{ModelName: modelRight, Perspective: "right"},
		},
		Formula:          "average",
		DefaultMissing:   0.0,
		ConfidenceMethod: "count_valid",
		MinConfidence:    0.1,
		MaxConfidence:    1.0,
		HandleInvalid:    "replace",
		MinScore:         -1.0, // Accept all valid scores
		MaxScore:         1.0,  // Accept all valid scores
	}

	// Create test scores using both configured and legacy model names
	scores := []db.LLMScore{
		// Modern model names (configured)
		{ArticleID: 1, Model: modelLeft, Score: 0.2},
		// Legacy model names (not in config)
		{ArticleID: 1, Model: "left", Score: 0.1},
		{ArticleID: 1, Model: "center", Score: 0.5},
		{ArticleID: 1, Model: "right", Score: 0.9},
		// Unknown model (should be skipped)
		{ArticleID: 1, Model: "unknown", Score: 0.7},
	}

	// Calculate score, passing the local testConfig
	score, confidence, err := ComputeCompositeScoreWithConfidenceFixed(scores, testConfig)

	// Assertions
	require.NoError(t, err)
	// Only the highest confidence score from each perspective should be used
	// We expect the function to use left=0.2, center=0.5, right=0.9
	// Average: (0.2 + 0.5 + 0.9) / 3 = 0.53
	assert.InDelta(t, 0.53, score, 0.01)
	// Expect confidence to be 1.0 when all perspectives are present
	assert.InDelta(t, 1.0, confidence, 0.01)
}

// TestModelMappingWithInvalidConfiguration tests model mapping with invalid configurations
func TestModelMappingWithInvalidConfiguration(t *testing.T) {
	// Test with nil config
	assert.Equal(t, "", MapModelToPerspective("any-model", nil))

	// Test with empty models list
	emptyConfig := &CompositeScoreConfig{
		Models: []ModelConfig{},
	}
	assert.Equal(t, "", MapModelToPerspective("any-model", emptyConfig))

	// Test with invalid perspective values
	invalidConfig := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "model1", Perspective: "invalid"},
			{ModelName: "model2", Perspective: ""},
		},
	}

	// The function should return the perspective as is, even if invalid
	assert.Equal(t, "invalid", MapModelToPerspective("model1", invalidConfig))
	assert.Equal(t, "", MapModelToPerspective("model2", invalidConfig))
}

// TestModelNameEdgeCases tests how the system handles unusual edge cases
func TestModelNameEdgeCases(t *testing.T) {
	// Create config with edge cases
	config := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "empty-string-model", Perspective: "left"},
			{ModelName: " space-prefix", Perspective: "left"},
			{ModelName: "space-suffix ", Perspective: "left"},
			{ModelName: "model/with/multiple/slashes", Perspective: "special"},
			{ModelName: "model:with:colons", Perspective: "special"},
			{ModelName: "", Perspective: "empty-name-perspective"}, // Empty model name
			{ModelName: "duplicate-model", Perspective: "perspective1"},
			{ModelName: "duplicate-model", Perspective: "perspective2"}, // Duplicate
		},
	}

	// Test the edge cases
	assert.Equal(t, "left", MapModelToPerspective("empty-string-model", config))
	assert.Equal(t, "left", MapModelToPerspective(" space-prefix", config))
	assert.Equal(t, "left", MapModelToPerspective("space-suffix ", config))
	assert.Equal(t, "special", MapModelToPerspective("model/with/multiple/slashes", config))
	assert.Equal(t, "special", MapModelToPerspective("model:with:colons", config))
	assert.Equal(t, "empty-name-perspective", MapModelToPerspective("", config)) // Empty model name

	// For duplicates, the first match in the Models array should be used
	assert.Equal(t, "perspective1", MapModelToPerspective("duplicate-model", config))
}
