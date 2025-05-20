package llm

import (
	"math"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestIsInvalid(t *testing.T) {
	cfg := &CompositeScoreConfig{
		MinScore: -1.0,
		MaxScore: 1.0,
	}

	tests := []struct {
		name     string
		val      float64
		expected bool
	}{
		{
			name:     "NaN value",
			val:      math.NaN(),
			expected: true,
		},
		{
			name:     "positive infinity",
			val:      math.Inf(1),
			expected: true,
		},
		{
			name:     "negative infinity",
			val:      math.Inf(-1),
			expected: true,
		},
		{
			name:     "below min score",
			val:      -1.5,
			expected: true,
		},
		{
			name:     "above max score",
			val:      1.5,
			expected: true,
		},
		{
			name:     "valid value",
			val:      0.5,
			expected: false,
		},
		{
			name:     "zero value",
			val:      0.0,
			expected: false,
		},
		{
			name:     "negative valid value",
			val:      -0.5,
			expected: false,
		},
		{
			name:     "min boundary",
			val:      -1.0,
			expected: false,
		},
		{
			name:     "max boundary",
			val:      1.0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInvalid(tt.val, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapModelsToPerspectives(t *testing.T) {
	cfg := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "progressive", Perspective: "left"},
			{ModelName: "neutral", Perspective: "center"},
			{ModelName: "conservative", Perspective: "right"},
		},
	}

	tests := []struct {
		name                 string
		scores               []db.LLMScore
		expectedPerspectives map[string]int // map of perspective -> expected count
	}{
		{
			name: "map models to perspectives",
			scores: []db.LLMScore{
				{Model: "neutral", Score: 0.0, Metadata: `{"confidence": 0.8}`},
				{Model: "progressive", Score: 0.7, Metadata: `{"confidence": 0.9}`},
				{Model: "conservative", Score: -0.5, Metadata: `{"confidence": 0.7}`},
			},
			expectedPerspectives: map[string]int{
				"center": 1,
				"left":   1,
				"right":  1,
			},
		},
		{
			name: "ensemble model maps to center",
			scores: []db.LLMScore{
				{Model: "ensemble", Score: 0.3, Metadata: `{"confidence": 0.6}`},
				{Model: "progressive", Score: 0.7, Metadata: `{"confidence": 0.9}`},
			},
			expectedPerspectives: map[string]int{
				"center": 1,
				"left":   1,
			},
		},
		{
			name: "unknown model is skipped",
			scores: []db.LLMScore{
				{Model: "unknown-model", Score: 0.3, Metadata: `{"confidence": 0.6}`},
				{Model: "progressive", Score: 0.7, Metadata: `{"confidence": 0.9}`},
			},
			expectedPerspectives: map[string]int{
				"left": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapModelsToPerspectives(tt.scores, cfg)

			// Check the perspective counts match
			for perspective, expectedCount := range tt.expectedPerspectives {
				if models, exists := result[perspective]; exists {
					assert.Equal(t, expectedCount, len(models), "Expected %d models for perspective %s", expectedCount, perspective)
				} else {
					assert.Fail(t, "Expected perspective %s not found in result", perspective)
				}
			}

			// Verify no unexpected perspectives
			for perspective := range result {
				_, expected := tt.expectedPerspectives[perspective]
				assert.True(t, expected, "Found unexpected perspective %s in result", perspective)
			}
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	tests := []struct {
		name           string
		validModels    map[string]bool
		scoreMap       map[string]float64
		expectedResult float64
		config         *CompositeScoreConfig
	}{
		{
			name: "count_valid method with all perspectives",
			validModels: map[string]bool{
				"left":   true,
				"right":  true,
				"center": true,
			},
			scoreMap: map[string]float64{
				"left":   0.8,
				"right":  -0.9,
				"center": 0.0,
			},
			expectedResult: 1.0, // 3/3 perspectives valid
			config: &CompositeScoreConfig{
				ConfidenceMethod: "count_valid",
			},
		},
		{
			name: "count_valid method with missing perspectives",
			validModels: map[string]bool{
				"left":  true,
				"right": true,
			},
			scoreMap: map[string]float64{
				"left":  0.8,
				"right": -0.9,
			},
			expectedResult: 0.667, // 2/3 perspectives valid
			config: &CompositeScoreConfig{
				ConfidenceMethod: "count_valid",
				MinConfidence:    0.1,
				MaxConfidence:    0.9,
			},
		},
		{
			name: "default method when unspecified",
			validModels: map[string]bool{
				"left":   true,
				"center": true,
			},
			scoreMap: map[string]float64{
				"left":   0.8,
				"center": 0.0,
			},
			expectedResult: 0.667, // Default to count_valid, 2/3 perspectives
			config: &CompositeScoreConfig{
				MinConfidence: 0.1,
				MaxConfidence: 0.9,
			},
		},
		{
			name: "confidence respects min and max bounds",
			validModels: map[string]bool{
				"left": true,
			},
			scoreMap: map[string]float64{
				"left": 0.8,
			},
                        expectedResult: 0.333, // 1/3 perspectives valid
			config: &CompositeScoreConfig{
				ConfidenceMethod: "count_valid",
				MinConfidence:    0.3,
				MaxConfidence:    0.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateConfidence(tt.config, tt.validModels, tt.scoreMap)
			assert.InDelta(t, tt.expectedResult, result, 0.001)
		})
	}
}
