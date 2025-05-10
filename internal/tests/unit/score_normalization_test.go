package unit

import (
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/stretchr/testify/assert"
)

// Benchmarking normalization performance
func BenchmarkScoreNormalization(b *testing.B) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &llm.DefaultScoreCalculator{}

	// Test with a mix of in-range and out-of-range scores
	scores := []db.LLMScore{
		TestScore{Model: "left", Score: -1.5, Confidence: 0.9}.ToLLMScore(),
		TestScore{Model: "center", Score: 0.0, Confidence: 0.8}.ToLLMScore(),
		TestScore{Model: "right", Score: 1.2, Confidence: 0.7}.ToLLMScore(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = calc.CalculateScore(scores, cfg)
	}
}

// Add performance benchmark specifically for clustered scores
func BenchmarkClusteredScoresNormalization(b *testing.B) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &llm.DefaultScoreCalculator{}

	// Test with clustered scores near boundary
	scores := []db.LLMScore{
		TestScore{Model: "left", Score: -0.98, Confidence: 0.9}.ToLLMScore(),
		TestScore{Model: "center", Score: -0.95, Confidence: 0.8}.ToLLMScore(),
		TestScore{Model: "right", Score: -0.92, Confidence: 0.7}.ToLLMScore(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = calc.CalculateScore(scores, cfg)
	}
}

func TestScoreNormalization(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:         -1.0,
		MaxScore:         1.0,
		Models:           []llm.ModelConfig{{Perspective: "left", ModelName: "left"}}, // Dummy model for perspective mapping
		Formula:          "average",
		DefaultMissing:   0.0,
		HandleInvalid:    "ignore", // Changed to ignore to match implementation
		ConfidenceMethod: "count_valid",
		MinConfidence:    0.0,
		MaxConfidence:    1.0,
	}
	// Initialize calculator without config
	calc := &llm.DefaultScoreCalculator{}

	testCases := []struct {
		name          string
		inputScores   []db.LLMScore
		expectedScore float64
		expectError   bool
	}{
		{
			name: "Score within range",
			inputScores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 0.5,
			expectError:   false,
		},
		{
			name: "Score below min",
			inputScores: []db.LLMScore{
				{Model: "left", Score: -1.5, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 0.0,
			expectError:   true,
		},
		{
			name: "Score above max",
			inputScores: []db.LLMScore{
				{Model: "left", Score: 1.5, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 0.0,
			expectError:   true,
		},
		{
			name: "Score at min",
			inputScores: []db.LLMScore{
				{Model: "left", Score: -1.0, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: -1.0,
			expectError:   false,
		},
		{
			name: "Score at max",
			inputScores: []db.LLMScore{
				{Model: "left", Score: 1.0, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 1.0,
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, _, err := calc.CalculateScore(tc.inputScores, cfg)
			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, 0.0, score)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tc.expectedScore, score, 0.001)
			}
		})
	}
}

func TestScoreNormalizationWithDifferentConfig(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:         0.0,
		MaxScore:         5.0,
		Models:           []llm.ModelConfig{{Perspective: "left", ModelName: "left"}},
		Formula:          "average",
		DefaultMissing:   0.0,
		HandleInvalid:    "ignore", // Changed to ignore to match implementation
		ConfidenceMethod: "count_valid",
		MinConfidence:    0.0,
		MaxConfidence:    1.0,
	}
	// Initialize calculator without config
	calc := &llm.DefaultScoreCalculator{}

	testCases := []struct {
		name          string
		inputScores   []db.LLMScore
		expectedScore float64
		expectError   bool
	}{
		{
			name: "Score within range (0-5)",
			inputScores: []db.LLMScore{
				{Model: "left", Score: 2.5, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 2.5,
			expectError:   false,
		},
		{
			name: "Score below min (0)",
			inputScores: []db.LLMScore{
				{Model: "left", Score: -1.0, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 0.0,
			expectError:   true,
		},
		{
			name: "Score above max (5)",
			inputScores: []db.LLMScore{
				{Model: "left", Score: 6.0, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 0.0,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, _, err := calc.CalculateScore(tc.inputScores, cfg)
			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, 0.0, score)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tc.expectedScore, score, 0.001)
			}
		})
	}
}
