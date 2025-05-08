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
		MinScore: -1.0,
		MaxScore: 1.0,
		Models:   []llm.ModelConfig{{Perspective: "left", ModelName: "left"}}, // Dummy model for perspective mapping
		// Add other fields like Formula if needed by CalculateScore
		Formula: "average", DefaultMissing: 0.0, HandleInvalid: "default",
		ConfidenceMethod: "count_valid", MinConfidence: 0.0, MaxConfidence: 1.0,
	}
	// Initialize calculator without config
	calc := &llm.DefaultScoreCalculator{}

	testCases := []struct {
		name          string
		inputScores   []db.LLMScore
		expectedScore float64
	}{
		{"Score within range", []db.LLMScore{{Model: "left", Score: 0.5}}, 0.5},
		{"Score below min", []db.LLMScore{{Model: "left", Score: -1.5}}, 0.0},
		{"Score above max", []db.LLMScore{{Model: "left", Score: 1.5}}, 0.0},
		{"Score at min", []db.LLMScore{{Model: "left", Score: -1.0}}, -1.0},
		{"Score at max", []db.LLMScore{{Model: "left", Score: 1.0}}, 1.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Pass config to CalculateScore
			score, _, err := calc.CalculateScore(tc.inputScores, cfg)
			assert.NoError(t, err)
			assert.InDelta(t, tc.expectedScore, score, 0.001)
		})
	}
}

func TestScoreNormalizationWithDifferentConfig(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore: 0.0,
		MaxScore: 5.0,
		Models:   []llm.ModelConfig{{Perspective: "left", ModelName: "left"}},
		// Add other fields like Formula if needed
		Formula: "average", DefaultMissing: 0.0, HandleInvalid: "default",
		ConfidenceMethod: "count_valid", MinConfidence: 0.0, MaxConfidence: 1.0,
	}
	// Initialize calculator without config
	calc := &llm.DefaultScoreCalculator{}

	testCases := []struct {
		name          string
		inputScores   []db.LLMScore
		expectedScore float64
	}{
		{"Score within range (0-5)", []db.LLMScore{{Model: "left", Score: 2.5}}, 2.5},
		{"Score below min (0)", []db.LLMScore{{Model: "left", Score: -1.0}}, 0.0},
		{"Score above max (5)", []db.LLMScore{{Model: "left", Score: 6.0}}, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Pass config to CalculateScore
			score, _, err := calc.CalculateScore(tc.inputScores, cfg)
			assert.NoError(t, err)
			assert.InDelta(t, tc.expectedScore, score, 0.001)
		})
	}
}
