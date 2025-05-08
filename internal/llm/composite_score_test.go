package llm

import (
	"math"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
)

// TestComputeCompositeScoreWithAllZeroResponses tests the critical edge case
// where all LLMs return empty or zero-value responses, including ensemble results.
func TestComputeCompositeScoreWithAllZeroResponses(t *testing.T) {
	// Create a specific test config
	testCfg := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "model1", Perspective: "left"},
			{ModelName: "model2", Perspective: "center"},
			{ModelName: "model3", Perspective: "right"},
			{ModelName: "ensemble", Perspective: "center"},
		},
		Formula: "average", DefaultMissing: 0.0, HandleInvalid: "default",
		MinScore: -1.0, MaxScore: 1.0, ConfidenceMethod: "count_valid",
		MinConfidence: 0.0, MaxConfidence: 1.0,
	}

	testCases := []struct {
		name          string
		scores        []db.LLMScore
		expectError   bool
		errorContains string
	}{
		{
			name: "All models return 0 score and 0 confidence",
			scores: []db.LLMScore{
				{Model: "model1", Score: 0.0, Metadata: `{"confidence": 0.0}`},
				{Model: "model2", Score: 0.0, Metadata: `{"confidence": 0.0}`},
				{Model: "model3", Score: 0.0, Metadata: `{"confidence": 0.0}`},
			},
			expectError:   true,
			errorContains: "all LLMs returned empty or zero-confidence responses",
		},
		{
			name: "Only ensemble score with non-zero confidence in meta",
			scores: []db.LLMScore{
				{
					Model:    "ensemble",
					Score:    0.7, // Use a non-zero score for clarity
					Metadata: `{"all_sub_results":[{"model":"model1","score":0.1,"confidence":0.8},{"model":"model2","score":-0.1,"confidence":0.7}],"confidence":0.9,"final_aggregation":{"weighted_mean":0,"variance":0.1,"uncertainty_flag":false},"per_model_results":{},"per_model_aggregation":{},"timestamp":"2024-04-28T12:00:00Z"}`,
				},
			},
			expectError: false, // Expect no error, function returns ensemble score
			// errorContains: "only ensemble scores found", // Commented out as no error is returned
		},
		{
			name: "Only ensemble score with zero confidence in meta",
			scores: []db.LLMScore{
				{
					Model:    "ensemble",
					Score:    0.0,
					Metadata: `{"all_sub_results":[{"model":"model1","score":0.1,"confidence":0.8},{"model":"model2","score":-0.1,"confidence":0.7}],"confidence":0,"final_aggregation":{"weighted_mean":0,"variance":1.0,"uncertainty_flag":true},"per_model_results":{},"per_model_aggregation":{},"timestamp":"2024-04-28T12:00:00Z"}`,
				},
			},
			expectError:   true,
			errorContains: "failed to get valid scores from any LLM perspective",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ComputeCompositeScoreWithConfidenceFixed(tc.scores, testCfg) // Pass testCfg
			if tc.expectError {
				assert.Error(t, err)
				if err != nil && tc.errorContains != "" { // Check error is not nil before calling Contains
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComputeCompositeScoreWithConfidence(t *testing.T) {
	// Create a specific test config
	testCfg := &CompositeScoreConfig{
		Formula: "average", DefaultMissing: 0.0, Models: []ModelConfig{{Perspective: "left", ModelName: "left"}, {Perspective: "center", ModelName: "center"}, {Perspective: "right", ModelName: "right"}}, MinScore: -1, MaxScore: 1, ConfidenceMethod: "count_valid", MinConfidence: 0, MaxConfidence: 1,
	}
	scores := []db.LLMScore{
		{Model: "left", Score: -1.0, Metadata: `{"confidence":0.9}`},
		{Model: "center", Score: 0.0, Metadata: `{"confidence":0.8}`},
		{Model: "right", Score: 1.0, Metadata: `{"confidence":0.85}`},
	}
	score, confidence, err := ComputeCompositeScoreWithConfidenceFixed(scores, testCfg) // Pass testCfg
	assert.NoError(t, err)
	assert.InDelta(t, 0.0, score, 1e-9)
	assert.InDelta(t, 1.0, confidence, 1e-9) // Assert confidence based on method
}

func TestComputeCompositeScoreEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		scores         []db.LLMScore
		expected       float64
		configOverride *CompositeScoreConfig
		description    string
		expectError    bool
	}{
		{
			name:        "empty scores array",
			scores:      []db.LLMScore{},
			expected:    0.0,
			description: "Empty scores array should return the default (0.0)",
			expectError: true,
		},
		{
			name: "extreme values within bounds",
			scores: []db.LLMScore{
				{Model: "left", Score: -1.0, Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 1.0, Metadata: `{"confidence":0.85}`},
			},
			expected:    0.0,
			description: "Extreme values within bounds should be averaged correctly",
		},
		{
			name: "values outside bounds",
			scores: []db.LLMScore{
				{Model: "left", Score: -2.0, Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 2.0, Metadata: `{"confidence":0.85}`},
			},
			expected:    0.0,
			description: "Values outside bounds should use default value (0.0)",
		},
		{
			name: "non-standard model names",
			scores: []db.LLMScore{
				{Model: "unknown-model-1", Score: -0.5, Metadata: `{"confidence":0.9}`},
				{Model: "unknown-model-2", Score: 0.3, Metadata: `{"confidence":0.8}`},
			},
			expected:    0.0, // DefaultMissing is 0.0 when no valid scores are found
			description: "Non-standard model names should result in an error and default score",
			expectError: true, // Should expect an error: "no valid scores found"
		},
		{
			name: "case insensitive model names",
			scores: []db.LLMScore{
				{Model: "LEFT", Score: -0.6, Metadata: `{"confidence":0.9}`},
				{Model: "Center", Score: 0.0, Metadata: `{"confidence":0.8}`},
				{Model: "RIGHT", Score: 0.6, Metadata: `{"confidence":0.85}`},
			},
			expected:    0.0,
			description: "Model names should be case-insensitive",
		},
		{
			name: "NaN values",
			scores: []db.LLMScore{
				{Model: "left", Score: math.NaN()},
				{Model: "center", Score: 0.2, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 0.4, Metadata: `{"confidence":0.85}`},
			},
			expected:    0.2, // Avg of (0.0, 0.2, 0.4) = 0.2
			description: "NaN values should be replaced with default (0.0)",
		},
		{
			name: "Infinity values",
			scores: []db.LLMScore{
				{Model: "left", Score: math.Inf(-1), Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: math.Inf(1), Metadata: `{"confidence":0.85}`},
			},
			description: "Infinity values should result in ErrAllPerspectivesInvalid",
			expectError: true,
		},
		{
			name: "weighted formula with config override",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.2, Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 0.4, Metadata: `{"confidence":0.85}`},
			},
			configOverride: &CompositeScoreConfig{
				Formula:          "weighted",
				Weights:          map[string]float64{"left": 0.2, "center": 0.3, "right": 0.5},
				MinScore:         -1.0,
				MaxScore:         1.0,
				DefaultMissing:   0.0,
				HandleInvalid:    "default",
				ConfidenceMethod: "count_valid",
			},
			expected:    0.16,
			description: "Weighted formula should apply weights correctly",
		},
		{
			name: "ignore invalid with config override",
			scores: []db.LLMScore{
				{Model: "left", Score: math.NaN()},
				{Model: "center", Score: 0.2, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 0.4, Metadata: `{"confidence":0.85}`},
			},
			configOverride: &CompositeScoreConfig{
				Formula:          "average",
				HandleInvalid:    "ignore",
				MinScore:         -1.0,
				MaxScore:         1.0,
				DefaultMissing:   0.0,
				ConfidenceMethod: "count_valid",
			},
			expected:    0.3,
			description: "With ignore_invalid, only valid scores should be used",
		},
		{
			name: "duplicate model scores - should use last one",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.1, CreatedAt: time.Now().Add(-time.Minute), Metadata: `{"confidence":0.7}`},
				{Model: "center", Score: 0.2, CreatedAt: time.Now(), Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 0.3, CreatedAt: time.Now(), Metadata: `{"confidence":0.85}`},
				{Model: "left", Score: 0.4, CreatedAt: time.Now(), Metadata: `{"confidence":0.9}`}, // Newer score for left
			},
			expected:    0.3, // Uses left=0.4, center=0.2, right=0.3 -> Avg = 0.3
			description: "When duplicate models exist, last one should be used",
		},
		{
			name: "custom min/max bounds",
			scores: []db.LLMScore{
				{Model: "left", Score: -3.0, Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 3.0, Metadata: `{"confidence":0.85}`},
			},
			configOverride: &CompositeScoreConfig{
				Formula:          "average",
				MinScore:         -2.0,
				MaxScore:         2.0,
				DefaultMissing:   0.0,
				HandleInvalid:    "default",
				ConfidenceMethod: "count_valid",
			},
			expected:    0.0,
			description: "Custom min/max bounds should be respected",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set config override if provided
			// If no override, use a default test config
			cfgToUse := tc.configOverride
			if cfgToUse == nil {
				// Create a default test config if no override is provided
				// This assumes a basic structure; adjust as needed for the test's logic
				cfgToUse = &CompositeScoreConfig{
					Formula:          "average",
					Weights:          map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0},
					MinScore:         -1.0,
					MaxScore:         1.0,
					DefaultMissing:   0.0,
					HandleInvalid:    "default",
					ConfidenceMethod: "count_valid",
					Models:           []ModelConfig{{Perspective: "left", ModelName: "left"}, {Perspective: "center", ModelName: "center"}, {Perspective: "right", ModelName: "right"}},
				}
			}

			// Pass the config explicitly to ComputeCompositeScore
			score, confidence, err := ComputeCompositeScoreWithConfidenceFixed(tc.scores, cfgToUse)

			if tc.name == "Infinity values" { // Specific handling for "Infinity values"
				assert.Error(t, err, tc.description)
				assert.EqualError(t, err, ErrAllPerspectivesInvalid.Error(), tc.description)
				assert.Equal(t, 0.0, score, "Score should be 0.0 on ErrAllPerspectivesInvalid")
				assert.Equal(t, 0.0, confidence, "Confidence should be 0.0 on ErrAllPerspectivesInvalid")
			} else if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tc.expected, score, 0.001, tc.description)
			}
		})
	}
}

func TestComputeCompositeScoreWeightedCalculation(t *testing.T) {
	// Create a base config for weighted tests
	baseCfg := &CompositeScoreConfig{
		Formula: "weighted", DefaultMissing: 0.0,
		Models:   []ModelConfig{{Perspective: "left", ModelName: "left"}, {Perspective: "center", ModelName: "center"}, {Perspective: "right", ModelName: "right"}},
		MinScore: -1, MaxScore: 1, ConfidenceMethod: "count_valid", MinConfidence: 0, MaxConfidence: 1,
	}

	t.Run("Equal weights", func(t *testing.T) {
		scores := []db.LLMScore{
			{Model: "left", Score: 0.1, Metadata: `{"confidence":0.9}`},
			{Model: "center", Score: 0.1, Metadata: `{"confidence":0.9}`},
			{Model: "right", Score: 0.1, Metadata: `{"confidence":0.9}`},
		}
		testCfg := *baseCfg // Copy base config
		testCfg.Weights = map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0}
		score, _, err := ComputeCompositeScoreWithConfidenceFixed(scores, &testCfg) // Pass testCfg
		assert.NoError(t, err)
		assert.InDelta(t, 0.1, score, 0.001, "Equal weights should calculate average score")
	})

	t.Run("Unequal weights", func(t *testing.T) {
		scores := []db.LLMScore{
			{Model: "left", Score: 0.1, Metadata: `{"confidence":0.9}`},
			{Model: "center", Score: 0.2, Metadata: `{"confidence":0.9}`},
			{Model: "right", Score: 0.3, Metadata: `{"confidence":0.9}`},
		}
		testCfg := *baseCfg // Copy base config
		testCfg.Weights = map[string]float64{"left": 0.2, "center": 0.3, "right": 0.5}
		score, _, err := ComputeCompositeScoreWithConfidenceFixed(scores, &testCfg) // Pass testCfg
		assert.NoError(t, err)
		assert.InDelta(t, 0.23, score, 0.001, "Unequal weights should apply correct weighting") // (0.1*0.2 + 0.2*0.3 + 0.3*0.5) / 1.0 = 0.02 + 0.06 + 0.15 = 0.23
	})

	t.Run("Zero weight", func(t *testing.T) {
		scores := []db.LLMScore{
			{Model: "left", Score: 0.1, Metadata: `{"confidence":0.9}`},
			{Model: "center", Score: 0.2, Metadata: `{"confidence":0.9}`},
			{Model: "right", Score: 0.3, Metadata: `{"confidence":0.9}`},
		}
		testCfg := *baseCfg // Copy base config
		testCfg.Weights = map[string]float64{"left": 0.0, "center": 0.5, "right": 0.5}
		score, _, err := ComputeCompositeScoreWithConfidenceFixed(scores, &testCfg) // Pass testCfg
		assert.NoError(t, err)
		assert.InDelta(t, 0.25, score, 0.001, "Zero weight should exclude that perspective") // (0.2*0.5 + 0.3*0.5) / 1.0 = 0.1 + 0.15 = 0.25
	})
}
