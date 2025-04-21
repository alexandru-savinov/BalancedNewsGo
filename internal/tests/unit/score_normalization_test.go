package unit

import (
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
)

// Benchmarking normalization performance
func BenchmarkScoreNormalization(b *testing.B) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &llm.DefaultScoreCalculator{Config: cfg}

	// Test with a mix of in-range and out-of-range scores
	scores := []db.LLMScore{
		TestScore{Model: "left", Score: -1.5, Confidence: 0.9}.ToLLMScore(),
		TestScore{Model: "center", Score: 0.0, Confidence: 0.8}.ToLLMScore(),
		TestScore{Model: "right", Score: 1.2, Confidence: 0.7}.ToLLMScore(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = calc.CalculateScore(scores)
	}
}

// Add performance benchmark specifically for clustered scores
func BenchmarkClusteredScoresNormalization(b *testing.B) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &llm.DefaultScoreCalculator{Config: cfg}

	// Test with clustered scores near boundary
	scores := []db.LLMScore{
		TestScore{Model: "left", Score: -0.98, Confidence: 0.9}.ToLLMScore(),
		TestScore{Model: "center", Score: -0.95, Confidence: 0.8}.ToLLMScore(),
		TestScore{Model: "right", Score: -0.92, Confidence: 0.7}.ToLLMScore(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = calc.CalculateScore(scores)
	}
}
