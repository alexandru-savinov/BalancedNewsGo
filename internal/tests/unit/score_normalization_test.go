package unit

import (
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/stretchr/testify/assert"
)

func TestScoreNormalization(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &llm.DefaultScoreCalculator{Config: cfg}

	type normalizationTest struct {
		name          string
		scores        []db.LLMScore
		expectedScore float64
		expectedConf  float64
		expectError   bool
	}

	var tests []normalizationTest

	// Test 1: Verify relative distances are preserved when scaling down larger ranges
	tests = append(tests, normalizationTest{
		name: "preserve relative distances when scaling down",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -2.0, Confidence: 1.0}.ToLLMScore(),  // Will be normalized to -1.0
			TestScore{Model: "center", Score: 0.0, Confidence: 1.0}.ToLLMScore(), // Will stay at 0.0
			TestScore{Model: "right", Score: 2.0, Confidence: 1.0}.ToLLMScore(),  // Will be normalized to 1.0
		},
		expectedScore: 0.0, // Average of normalized scores (-1.0 + 0.0 + 1.0) / 3
		expectedConf:  1.0,
		expectError:   false,
	})

	// Test 2: Verify proportional scaling of smaller ranges
	tests = append(tests, normalizationTest{
		name: "proportional scaling of smaller ranges",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -0.5, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "center", Score: 0.0, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "right", Score: 0.5, Confidence: 1.0}.ToLLMScore(),
		},
		expectedScore: 0.0, // Average remains at center
		expectedConf:  1.0,
		expectError:   false,
	})

	// Test 3: Verify handling of asymmetric ranges
	tests = append(tests, normalizationTest{
		name: "asymmetric range normalization",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -0.3, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "center", Score: 0.0, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "right", Score: 0.9, Confidence: 1.0}.ToLLMScore(),
		},
		expectedScore: 0.2, // Average of asymmetric but valid scores
		expectedConf:  1.0,
		expectError:   false,
	})

	// Test 4: Mixed ranges requiring normalization
	tests = append(tests, normalizationTest{
		name: "mixed ranges requiring normalization",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -1.5, Confidence: 1.0}.ToLLMScore(),  // Will be normalized to -1.0
			TestScore{Model: "center", Score: 0.2, Confidence: 1.0}.ToLLMScore(), // Valid, stays as is
			TestScore{Model: "right", Score: 1.2, Confidence: 1.0}.ToLLMScore(),  // Will be normalized to 1.0
		},
		expectedScore: 0.067, // Average of (-1.0 + 0.2 + 1.0) / 3
		expectedConf:  1.0,
		expectError:   false,
	})

	// Test 5: Extreme value clusters
	tests = append(tests, normalizationTest{
		name: "extreme value clusters",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -5.0, Confidence: 1.0}.ToLLMScore(),   // Will be normalized to -1.0
			TestScore{Model: "center", Score: -4.8, Confidence: 1.0}.ToLLMScore(), // Will be normalized close to -1.0
			TestScore{Model: "right", Score: 5.0, Confidence: 1.0}.ToLLMScore(),   // Will be normalized to 1.0
		},
		expectedScore: -0.333, // Average of normalized scores, preserving relative positions
		expectedConf:  1.0,
		expectError:   false,
	})

	// Test 6: All out-of-range scores
	tests = append(tests, normalizationTest{
		name: "all scores out of range",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -2.0, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "center", Score: -1.5, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "right", Score: 1.5, Confidence: 1.0}.ToLLMScore(),
		},
		expectedScore: 0.0, // All scores will be normalized and averaged
		expectedConf:  1.0,
		expectError:   false,
	})

	// Test 7: Non-uniform distribution with bias clustering
	tests = append(tests, normalizationTest{
		name: "non-uniform distribution with bias clustering",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -0.9, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "center", Score: -0.85, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "right", Score: 0.1, Confidence: 1.0}.ToLLMScore(),
		},
		expectedScore: -0.550, // Average of clustered scores on left side
		expectedConf:  1.0,
		expectError:   false,
	})

	// Test 8: Clustered scores near boundaries
	tests = append(tests, normalizationTest{
		name: "clustered scores near boundaries",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -0.98, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "center", Score: -0.95, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "right", Score: -0.92, Confidence: 1.0}.ToLLMScore(),
		},
		expectedScore: -0.950, // Average of tightly clustered scores near boundary
		expectedConf:  1.0,
		expectError:   false,
	})

	// Test 9: Mixed clustering with outlier
	tests = append(tests, normalizationTest{
		name: "mixed clustering with outlier",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: 0.85, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "center", Score: 0.82, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "right", Score: -0.5, Confidence: 1.0}.ToLLMScore(),
		},
		expectedScore: 0.390, // Average showing impact of outlier on clustered scores
		expectedConf:  1.0,
		expectError:   false,
	})

	// Test 10: Dense center distribution
	tests = append(tests, normalizationTest{
		name: "dense center distribution",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -0.05, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "center", Score: 0.0, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "right", Score: 0.05, Confidence: 1.0}.ToLLMScore(),
		},
		expectedScore: 0.0, // Average of tightly clustered scores around center
		expectedConf:  1.0,
		expectError:   false,
	})

	// Run normalization tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, conf, err := calc.CalculateScore(tt.scores)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.InDelta(t, tt.expectedScore, score, 0.001, "score mismatch")
			assert.InDelta(t, tt.expectedConf, conf, 0.001, "confidence mismatch")
		})
	}
}

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
