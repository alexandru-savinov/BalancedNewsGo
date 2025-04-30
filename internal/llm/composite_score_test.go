package llm

import (
	"math"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
)

// TestComputeCompositeScoreWithAllZeroResponses tests the critical edge case
// where all LLMs return empty or zero-value responses
func TestComputeCompositeScoreWithAllZeroResponses(t *testing.T) {
	// Create scores with all zeros and empty metadata
	scores := []db.LLMScore{
		{Model: "left", Score: 0.0, Metadata: `{}`},
		{Model: "center", Score: 0.0, Metadata: `{}`},
		{Model: "right", Score: 0.0, Metadata: `{}`},
	}

	// Also test with explicit zero confidence
	scoresWithZeroConfidence := []db.LLMScore{
		{Model: "left", Score: 0.0, Metadata: `{"confidence":0.0}`},
		{Model: "center", Score: 0.0, Metadata: `{"confidence":0.0}`},
		{Model: "right", Score: 0.0, Metadata: `{"confidence":0.0}`},
	}

	// Test standard ComputeCompositeScoreWithConfidenceFixed function
	_, _, err := ComputeCompositeScoreWithConfidenceFixed(scores)
	assert.Error(t, err, "Should detect all-zero responses and return an error")
	assert.Contains(t, err.Error(), "all LLMs returned empty or zero-confidence responses",
		"Error message should indicate the specific issue")

	// Test with explicit zero confidence
	_, _, err = ComputeCompositeScoreWithConfidenceFixed(scoresWithZeroConfidence)
	assert.Error(t, err, "Should detect all-zero responses with explicit zero confidence")
	assert.Contains(t, err.Error(), "all LLMs returned empty or zero-confidence responses")
}

func TestComputeCompositeScoreWithConfidence(t *testing.T) {
	// Prepare input scores for left, center, right
	scores := []db.LLMScore{
		{Model: "left", Score: -1.0},
		{Model: "center", Score: 0.0},
		{Model: "right", Score: 1.0},
	}

	// Default config is average with count_valid
	s, c, err := ComputeCompositeScoreWithConfidence(scores)
	assert.NoError(t, err)
	// average(-1,0,1)=0
	assert.InDelta(t, 0.0, s, 1e-6)
	// all models valid => confidence=3/3
	assert.InDelta(t, 1.0, c, 1e-6)

	// Test weighted formula
	cfg := &CompositeScoreConfig{
		Formula:  "weighted",
		Weights:  map[string]float64{"left": 1, "center": 2, "right": 3},
		MinScore: -1, MaxScore: 1,
		HandleInvalid:    "default",
		DefaultMissing:   0,
		ConfidenceMethod: "spread",
	}
	// override global config
	compositeScoreConfig = cfg

	s2, c2, err := ComputeCompositeScoreWithConfidence(scores)
	assert.NoError(t, err)
	// weighted sum = -1*1 +0*2 +1*3 =2, total weights=6 => 0.333
	assert.InDelta(t, 2.0/6.0, s2, 1e-6)
	// spread = max-min = 2 => clamp between min and max confidence
	assert.InDelta(t, 1.0, c2, 1e-6)
}

func TestComputeCompositeScoreEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		scores         []db.LLMScore
		expected       float64
		configOverride *CompositeScoreConfig
		description    string
	}{
		{
			name:        "empty scores array",
			scores:      []db.LLMScore{},
			expected:    0.0,
			description: "Empty scores array should return the default (0.0)",
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
			expected:    0.0,
			description: "Non-standard model names should be ignored and defaults used",
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
				{Model: "left", Score: math.NaN(), Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 0.6, Metadata: `{"confidence":0.85}`},
			},
			expected:    0.2,
			description: "NaN values should be replaced with default (0.0)",
		},
		{
			name: "Infinity values",
			scores: []db.LLMScore{
				{Model: "left", Score: math.Inf(-1), Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: math.Inf(1), Metadata: `{"confidence":0.85}`},
			},
			expected:    0.0,
			description: "Infinity values should be replaced with default (0.0)",
		},
		{
			name: "weighted formula with config override",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8, Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.2, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 0.6, Metadata: `{"confidence":0.85}`},
			},
			configOverride: &CompositeScoreConfig{
				Formula:          "weighted",
				Weights:          map[string]float64{"left": 2.0, "center": 1.0, "right": 2.0},
				MinScore:         -1.0,
				MaxScore:         1.0,
				DefaultMissing:   0.0,
				HandleInvalid:    "default",
				ConfidenceMethod: "count_valid",
			},
			expected:    -0.04,
			description: "Weighted formula should apply weights correctly",
		},
		{
			name: "ignore invalid with config override",
			scores: []db.LLMScore{
				{Model: "left", Score: -2.0, Metadata: `{"confidence":0.9}`},
				{Model: "center", Score: 0.2, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 2.0, Metadata: `{"confidence":0.85}`},
			},
			configOverride: &CompositeScoreConfig{
				Formula:          "average",
				Weights:          map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0},
				MinScore:         -1.0,
				MaxScore:         1.0,
				DefaultMissing:   0.0,
				HandleInvalid:    "ignore",
				ConfidenceMethod: "count_valid",
			},
			expected:    0.0667,
			description: "With ignore_invalid, only valid scores should be used",
		},
		{
			name: "duplicate model scores - should use last one",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8, Metadata: `{"confidence":0.9}`},
				{Model: "left", Score: -0.4, Metadata: `{"confidence":0.95}`},
				{Model: "center", Score: 0.2, Metadata: `{"confidence":0.8}`},
				{Model: "right", Score: 0.6, Metadata: `{"confidence":0.85}`},
			},
			expected:    0.13333,
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
				Weights:          map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0},
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
			if tc.configOverride != nil {
				compositeScoreConfig = tc.configOverride
			}

			actual := ComputeCompositeScore(tc.scores)

			// Reset config override for next test
			compositeScoreConfig = nil

			assert.InDelta(t, tc.expected, actual, 0.001, tc.description)
		})
	}
}

func TestComputeCompositeScoreWeightedCalculation(t *testing.T) {
	// Test with specific weighted calculations
	scores := []db.LLMScore{
		{Model: "left", Score: -0.5, Metadata: `{"confidence":0.9}`},
		{Model: "center", Score: 0.1, Metadata: `{"confidence":0.8}`},
		{Model: "right", Score: 0.7, Metadata: `{"confidence":0.85}`},
	}

	// Test 1: Equal weights (should match average formula)
	equalWeightsConfig := &CompositeScoreConfig{
		Formula:          "weighted",
		Weights:          map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0},
		MinScore:         -1.0,
		MaxScore:         1.0,
		DefaultMissing:   0.0,
		HandleInvalid:    "default",
		ConfidenceMethod: "count_valid",
	}
	compositeScoreConfig = equalWeightsConfig
	equalWeightScore := ComputeCompositeScore(scores)
	compositeScoreConfig = nil // Reset

	// Calculate expected: (-0.5 + 0.1 + 0.7) / 3 = 0.1
	expectedEqualWeight := 0.1
	assert.InDelta(t, expectedEqualWeight, equalWeightScore, 0.001,
		"Equal weights should calculate average score")

	// Test 2: Unequal weights
	unequalWeightsConfig := &CompositeScoreConfig{
		Formula:          "weighted",
		Weights:          map[string]float64{"left": 2.0, "center": 1.0, "right": 3.0},
		MinScore:         -1.0,
		MaxScore:         1.0,
		DefaultMissing:   0.0,
		HandleInvalid:    "default",
		ConfidenceMethod: "count_valid",
	}
	compositeScoreConfig = unequalWeightsConfig
	unequalWeightScore := ComputeCompositeScore(scores)
	compositeScoreConfig = nil // Reset

	// After examining the actual implementation behavior
	expectedUnequalWeight := 0.2
	assert.InDelta(t, expectedUnequalWeight, unequalWeightScore, 0.001,
		"Unequal weights should apply correct weighting")

	// Test 3: Zero weight for one perspective
	zeroWeightConfig := &CompositeScoreConfig{
		Formula:          "weighted",
		Weights:          map[string]float64{"left": 0.0, "center": 1.0, "right": 1.0},
		MinScore:         -1.0,
		MaxScore:         1.0,
		DefaultMissing:   0.0,
		HandleInvalid:    "default",
		ConfidenceMethod: "count_valid",
	}
	compositeScoreConfig = zeroWeightConfig
	zeroWeightScore := ComputeCompositeScore(scores)
	compositeScoreConfig = nil // Reset

	// Calculate expected: (0.1*1 + 0.7*1) / (1+1) = 0.4
	expectedZeroWeight := 0.4
	assert.InDelta(t, expectedZeroWeight, zeroWeightScore, 0.001,
		"Zero weight should exclude that perspective")
}
