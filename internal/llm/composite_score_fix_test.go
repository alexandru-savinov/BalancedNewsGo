package llm

import (
	"encoding/json"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock config for testing
type mockConfigLoader struct {
	mock.Mock
}

// TestMapModelToPerspective tests the mapping of model names to perspectives
func TestMapModelToPerspective(t *testing.T) {
	// Create a test config based on the actual config file
	cfg := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "meta-llama/llama-4-maverick", Perspective: "left"},
			{ModelName: "google/gemini-2.0-flash-001", Perspective: "center"},
			{ModelName: "openai/gpt-4.1-nano", Perspective: "right"},
			{ModelName: "mixed_case", Perspective: "Left"},
		},
	}

	testCases := []struct {
		name                string
		modelName           string
		expectedPerspective string
	}{
		{
			name:                "Known left model",
			modelName:           "meta-llama/llama-4-maverick",
			expectedPerspective: "left",
		},
		{
			name:                "Known center model",
			modelName:           "google/gemini-2.0-flash-001",
			expectedPerspective: "center",
		},
		{
			name:                "Known right model",
			modelName:           "openai/gpt-4.1-nano",
			expectedPerspective: "right",
		},
		{
			name:                "Case insensitive matching",
			modelName:           "META-LLAMA/LLAMA-4-MAVERICK",
			expectedPerspective: "left",
		},
		{
			name:                "Whitespace handling",
			modelName:           " google/gemini-2.0-flash-001 ",
			expectedPerspective: "center",
		},
		{
			name:                "Unknown model",
			modelName:           "unknown_model",
			expectedPerspective: "",
		},
		{
			name:                "Mixed case perspective",
			modelName:           "mixed_case",
			expectedPerspective: "left", // Should be normalized to lowercase
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			perspective := MapModelToPerspective(tc.modelName, cfg)
			assert.Equal(t, tc.expectedPerspective, perspective)
		})
	}

	// Test with nil config
	t.Run("Nil config", func(t *testing.T) {
		perspective := MapModelToPerspective("model_name", nil)
		assert.Equal(t, "", perspective)
	})
}

// Helper to create a score with metadata containing confidence
func createScoreWithConfidence(articleID int64, model string, score float64, confidence float64) db.LLMScore {
	metadata := map[string]interface{}{
		"confidence": confidence,
	}
	metadataBytes, _ := json.Marshal(metadata)

	return db.LLMScore{
		ArticleID: articleID,
		Model:     model,
		Score:     score,
		Metadata:  string(metadataBytes),
	}
}

// TestComputeCompositeScoreWithConfidenceFixed tests the composite score calculation
func TestComputeCompositeScoreWithConfidenceFixed(t *testing.T) {
	// We need to mock the LoadCompositeScoreConfig since it reads from file
	// For this test, we'll temporarily replace the global variable fileCompositeScoreConfig

	// Save the original and restore after test
	originalConfig := fileCompositeScoreConfig
	defer func() { fileCompositeScoreConfig = originalConfig }()

	// Create a test config using the actual model names from the config file
	testConfig := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "meta-llama/llama-4-maverick", Perspective: "left"},
			{ModelName: "google/gemini-2.0-flash-001", Perspective: "center"},
			{ModelName: "openai/gpt-4.1-nano", Perspective: "right"},
			{ModelName: "left", Perspective: "left"}, // Adding legacy model for test case
		},
		Formula:          "average",
		DefaultMissing:   0.0,
		ConfidenceMethod: "count_valid",
		MinConfidence:    0.1,
		MaxConfidence:    0.95,
		MinScore:         -1.0,
		MaxScore:         1.0,
		HandleInvalid:    "replace",
		Weights: map[string]float64{
			"left":   1.0,
			"center": 1.0,
			"right":  1.0,
		},
	}

	// Set our test config
	fileCompositeScoreConfig = testConfig

	testCases := []struct {
		name          string
		scores        []db.LLMScore
		expectedScore float64
		expectedConf  float64
		expectError   bool
	}{
		{
			name: "All valid scores",
			scores: []db.LLMScore{
				createScoreWithConfidence(1, "meta-llama/llama-4-maverick", 0.1, 0.8),
				createScoreWithConfidence(1, "google/gemini-2.0-flash-001", 0.5, 0.9),
				createScoreWithConfidence(1, "openai/gpt-4.1-nano", 0.9, 0.7),
			},
			expectedScore: 0.5,  // Average of 0.1, 0.5, 0.9
			expectedConf:  0.95, // All 3 models valid, max confidence is 0.95
			expectError:   false,
		},
		{
			name: "Missing one perspective",
			scores: []db.LLMScore{
				createScoreWithConfidence(1, "meta-llama/llama-4-maverick", 0.2, 0.8),
				createScoreWithConfidence(1, "openai/gpt-4.1-nano", 0.8, 0.9),
			},
			expectedScore: 0.33, // (0.2 + 0 + 0.8) / 3 = 0.33
			expectedConf:  0.67, // 2/3 valid models
			expectError:   false,
		},
		{
			name: "Multiple models same perspective, use highest confidence",
			scores: []db.LLMScore{
				createScoreWithConfidence(1, "meta-llama/llama-4-maverick", 0.1, 0.6),
				createScoreWithConfidence(1, "google/gemini-2.0-flash-001", 0.5, 0.9),
				createScoreWithConfidence(1, "openai/gpt-4.1-nano", 0.9, 0.7),
				// Second model for left with higher confidence
				{
					ArticleID: 1,
					Model:     "left", // Added to config now
					Score:     0.3,
					Metadata:  `{"confidence": 0.8}`,
				},
			},
			expectedScore: 0.57, // Average of 0.3, 0.5, 0.9
			expectedConf:  0.95, // All 3 perspectives valid, max confidence is 0.95
			expectError:   false,
		},
		{
			name: "No valid scores",
			scores: []db.LLMScore{
				// Only ensemble score, no perspective scores
				{ArticleID: 1, Model: "ensemble", Score: 0.5},
			},
			expectedScore: 0,
			expectedConf:  0,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, confidence, err := ComputeCompositeScoreWithConfidenceFixed(tc.scores)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tc.expectedScore, score, 0.1)
				assert.InDelta(t, tc.expectedConf, confidence, 0.1)
			}
		})
	}
}
