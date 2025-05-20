package unit

import (
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/stretchr/testify/assert"
)

func TestExtractConfidence(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	// Initialize calculator without config
	calc := &llm.DefaultScoreCalculator{}

	tests := []struct {
		name         string
		scores       []db.LLMScore
		expectedConf float64
		description  string
		expectError  bool
	}{
		{
			name: "valid float confidence",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.85}`},
			},
			expectedConf: 0.85,
			description:  "Tests standard float confidence value",
		},
		{
			name: "valid integer confidence",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 1}`},
			},
			expectedConf: 1.0,
			description:  "Tests integer confidence value",
		},
		{
			name: "malformed JSON",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{malformed`},
			},
			expectedConf: 0.0,
			description:  "Tests invalid JSON metadata",
			expectError:  true,
		},
		{
			name: "empty metadata",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: ""},
			},
			expectedConf: 0.0,
			description:  "Tests empty metadata string",
			expectError:  true,
		},
		{
			name: "string confidence value",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": "0.8"}`},
			},
			expectedConf: 0.0,
			description:  "Tests string instead of number",
			expectError:  true,
		},
		{
			name: "boolean confidence value",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": true}`},
			},
			expectedConf: 0.0,
			description:  "Tests boolean instead of number",
			expectError:  true,
		},
		{
			name: "multiple scores",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.8}`},
				{Model: "right", Score: -0.3, Metadata: `{"confidence": 0.9}`},
			},
			expectedConf: 0.85,
			description:  "Tests confidence averaging across multiple scores",
		},
		{
			name: "boundary values",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.0}`},
				{Model: "right", Score: -0.3, Metadata: `{"confidence": 1.0}`},
			},
			expectedConf: 1.0,
			description:  "Tests boundary confidence values",
		},
		{
			name: "deep nested confidence",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"analysis": {"metrics": {"confidence": 0.75}}}`},
			},
			expectedConf: 0.0,
			description:  "Tests deeply nested confidence value (not supported)",
			expectError:  true,
		},
		{
			name: "confidence with scientific notation",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 9.5e-01}`},
			},
			expectedConf: 0.95,
			description:  "Tests scientific notation for confidence value",
		},
		{
			name: "mixed confidence formats",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.8}`},
				{Model: "right", Score: -0.3, Metadata: `{"confidence": "invalid"}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.6}`},
			},
			expectedConf: 0.7,
			description:  "Tests mixed valid and invalid confidence formats",
		},
		{
			name: "multiple perspectives with different confidences",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.9}`},
				{Model: "right", Score: -0.7, Metadata: `{"confidence": 0.8}`},
				{Model: "center", Score: 0.1, Metadata: `{"confidence": 0.7}`},
			},
			expectedConf: 0.8,
			description:  "Tests multiple perspectives with different confidence values",
		},
		{
			name: "null confidence value",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": null}`},
			},
			expectedConf: 0.0,
			description:  "Tests null confidence value",
			expectError:  true,
		},
		{
			name: "missing confidence field",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"other_field": 0.9}`},
			},
			expectedConf: 0.0,
			description:  "Tests missing confidence field",
			expectError:  true,
		},
		{
			name: "inconsistent perspectives",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.9}`},
				{Model: "unknown", Score: 0.3, Metadata: `{"confidence": 0.7}`},
			},
			expectedConf: 0.9,
			description:  "Tests unknown perspective handling",
		},
		{
			name:         "no scores",
			scores:       []db.LLMScore{},
			expectedConf: 0.0,
			description:  "Tests empty scores array",
			expectError:  true,
		},
		{
			name: "confidence objects instead of values",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": {"value": 0.9, "source": "model"}}`},
			},
			expectedConf: 0.0,
			description:  "Tests complex confidence objects",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, conf, err := calc.CalculateScore(tt.scores, cfg)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, 0.0, conf)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expectedConf, conf, 0.001, tt.description)
			}
		})
	}
}

func TestConfidenceCalculationWithLimitedPerspectives(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &llm.DefaultScoreCalculator{}

	scores := []db.LLMScore{
		{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.85}`},
	}

	_, conf, err := calc.CalculateScore(scores, cfg)
	assert.NoError(t, err)
	assert.InDelta(t, 0.85, conf, 0.001, "Confidence should be 0.85 with just left perspective")

	scores = []db.LLMScore{
		{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.85}`},
		{Model: "center", Score: 0.1, Metadata: `{"confidence": 0.75}`},
	}

	_, conf, err = calc.CalculateScore(scores, cfg)
	assert.NoError(t, err)
	assert.InDelta(t, 0.8, conf, 0.001, "Confidence should be average of available perspectives")
}

func TestConfidenceInheritance(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
		HandleInvalid:  "ignore",
	}
	calc := &llm.DefaultScoreCalculator{}

	// Test that confidence is inherited from the latest score for a perspective
	scores := []db.LLMScore{
		{Model: "left", Score: -0.5, Metadata: `{"confidence": 0.7}`},
		{Model: "left", Score: -0.3, Metadata: `{"confidence": 0.9}`}, // This confidence should be used
		{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`},
	}

	_, conf, err := calc.CalculateScore(scores, cfg)
	assert.NoError(t, err)
	assert.InDelta(t, 0.8, conf, 0.001, "Should use average of all confidences")

	// Test that confidence is inherited from the latest model for a perspective
	scores = []db.LLMScore{
		{Model: "left-leaning", Score: -0.5, Metadata: `{"confidence": 0.7}`},
		{Model: "left", Score: -0.3, Metadata: `{"confidence": 0.9}`}, // This confidence should be used
		{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`},
	}

	_, conf, err = calc.CalculateScore(scores, cfg)
	assert.NoError(t, err)
	assert.InDelta(t, 0.8, conf, 0.001, "Should use average of all confidences")
}

func TestConfidenceCalculationWithNilConfig(t *testing.T) {
	calc := &llm.DefaultScoreCalculator{}

	scores := []db.LLMScore{
		{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.85}`},
	}

	_, _, err := calc.CalculateScore(scores, nil)
	assert.Error(t, err, "Should return error when Config is nil")
	assert.Contains(t, err.Error(), "Config must not be nil", "Error message should mention nil Config")
}

func TestConfidenceCalculationWithExtremeValues(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &llm.DefaultScoreCalculator{}

	scores := []db.LLMScore{
		{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.0000001}`},
		{Model: "right", Score: -0.5, Metadata: `{"confidence": 0.0000002}`},
	}

	_, conf, err := calc.CalculateScore(scores, cfg)
	assert.NoError(t, err)
	assert.InDelta(t, 0.00000015, conf, 0.0000001, "Should handle extremely small confidence values")

	largeScores := make([]db.LLMScore, 1000)
	for i := 0; i < 1000; i++ {
		largeScores[i] = db.LLMScore{
			Model:    "left",
			Score:    0.5,
			Metadata: `{"confidence": 0.9}`,
		}
	}

	_, conf, err = calc.CalculateScore(largeScores, cfg)
	assert.NoError(t, err)
	assert.InDelta(t, 0.9, conf, 0.001, "Should handle large number of scores")
}

func TestZeroScoreWithNonZeroConfidence(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
		HandleInvalid:  "ignore",
	}
	calc := &llm.DefaultScoreCalculator{}

	scores := []db.LLMScore{
		{Model: "left", Score: 0.0, Metadata: `{"confidence": 0.85}`},
		{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.75}`},
		{Model: "right", Score: 0.0, Metadata: `{"confidence": 0.90}`},
	}

	score, conf, err := calc.CalculateScore(scores, cfg)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, score, "Score should be 0.0 when all scores are zero")
	assert.InDelta(t, 0.833, conf, 0.001, "Confidence should be average of non-zero confidences")

	// Test with all zero confidence
	scores = []db.LLMScore{
		{Model: "left", Score: 0.0, Metadata: `{"confidence": 0.0}`},
		{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.0}`},
		{Model: "right", Score: 0.0, Metadata: `{"confidence": 0.0}`},
	}

	score, conf, err = calc.CalculateScore(scores, cfg)
	assert.Error(t, err)
	assert.ErrorIs(t, err, llm.ErrAllPerspectivesInvalid)
	assert.Equal(t, 0.0, score, "Score should be 0.0 when error occurs")
	assert.Equal(t, 0.0, conf, "Confidence should be 0.0 when error occurs")
}

func TestCalculateConfidence_NilConfig(t *testing.T) {
	calc := &llm.DefaultScoreCalculator{}
	scores := []db.LLMScore{{Model: "left", Score: 0.5}}
	_, _, err := calc.CalculateScore(scores, nil) // Pass nil config
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Config must not be nil")
}
