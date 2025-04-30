package llm

import (
	"strings"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// Helper function to normalize model names for tests
func testNormalizeModelName(modelName string) string {
	// Trim spaces and convert to lowercase for case-insensitive comparison
	name := strings.TrimSpace(modelName)
	name = strings.ToLower(name)

	// Remove any version or revision information after colon
	if colonIndex := strings.Index(name, ":"); colonIndex != -1 {
		name = name[:colonIndex]
	}

	// Remove any complex version info (e.g. @v1.2.3+experimental)
	if atIndex := strings.Index(name, "@"); atIndex != -1 {
		name = name[:atIndex]
	}

	return name
}

// Helper function for model perspective mapping in tests
func testGetPerspectiveFromModel(modelName string, cfg *CompositeScoreConfig) string {
	// First try using MapModelToPerspective function
	perspective := MapModelToPerspective(modelName, cfg)
	if perspective != "" {
		return perspective
	}

	// Fall back to legacy mapping
	normalizedModel := testNormalizeModelName(modelName)
	if normalizedModel == "left" || strings.Contains(normalizedModel, "left") {
		return "left"
	} else if normalizedModel == "center" || strings.Contains(normalizedModel, "center") {
		return "center"
	} else if normalizedModel == "right" || strings.Contains(normalizedModel, "right") {
		return "right"
	}
	return ""
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

// TestModelNameNormalization tests the normalization of model names in different contexts
func TestModelNameNormalization(t *testing.T) {
	// Create a test config
	cfg := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: modelLeft, Perspective: "left"},
			{ModelName: modelCenter, Perspective: "center"},
			{ModelName: modelRight, Perspective: "right"},
		},
	}

	testCases := []struct {
		name                string
		modelName           string
		expectedPerspective string
	}{
		{
			name:                "Model with extra spaces",
			modelName:           "  " + modelLeft + "  ",
			expectedPerspective: "left",
		},
		{
			name:                "Model with mixed case",
			modelName:           "Meta-Llama/Llama-4-Maverick",
			expectedPerspective: "left",
		},
		{
			name:                "Model with unusual spacing",
			modelName:           modelCenter + "\t",
			expectedPerspective: "center",
		},
		{
			name:                "Model with non-breaking space",
			modelName:           modelRight + "\u00A0",
			expectedPerspective: "right",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			perspective := MapModelToPerspective(tc.modelName, cfg)
			assert.Equal(t, tc.expectedPerspective, perspective)
		})
	}
}

// TestModelNameFallbackLogic tests the fallback logic when model names aren't found in config
func TestModelNameFallbackLogic(t *testing.T) {
	// Save original config and restore after test
	originalConfig := testModelConfig
	defer func() { testModelConfig = originalConfig }()

	// Create a test config
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

	// Set our test config
	testModelConfig = testConfig

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

	// Calculate score
	score, confidence, err := ComputeCompositeScoreWithConfidenceFixed(scores)

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
