package unit

import (
	"fmt"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/stretchr/testify/assert"
)

func TestScoreBoundaryValidation(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &llm.DefaultScoreCalculator{}

	// Test structure definition
	type boundaryTest struct {
		name          string
		scores        []db.LLMScore
		expectedScore float64
		expectedConf  float64
		expectError   bool
	}

	// Preallocate slice with estimated capacity
	tests := make([]boundaryTest, 0, 25)

	// Exact boundary tests
	tests = append(tests, boundaryTest{
		name:          "exact lower boundary -1.0",
		scores:        GenerateScoreSet(-1.0, 1.0),
		expectedScore: -1.0,
		expectedConf:  1.0,
		expectError:   false,
	})

	tests = append(tests, boundaryTest{
		name:          "exact upper boundary 1.0",
		scores:        GenerateScoreSet(1.0, 1.0),
		expectedScore: 1.0,
		expectedConf:  1.0,
		expectError:   false,
	})

	// Generate tests at 0.1 intervals from -1.0 to 1.0
	steppedScores := GenerateSteppedScores(-1.0, 1.0, 0.1, 1.0)
	for i, scores := range steppedScores {
		score := -1.0 + (float64(i) * 0.1)
		tests = append(tests, boundaryTest{
			name:          fmt.Sprintf("interval test at %.1f", score),
			scores:        scores,
			expectedScore: score,
			expectedConf:  1.0,
			expectError:   false,
		})
	}

	// Just beyond boundaries tests
	tests = append(tests, boundaryTest{
		name: "just below lower boundary -1.001",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: -1.001, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "center", Score: 0.0, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "right", Score: 0.0, Confidence: 1.0}.ToLLMScore(),
		},
		expectedScore: 0.0, // Should be normalized to 0.0
		expectedConf:  1.0,
		expectError:   false,
	})

	tests = append(tests, boundaryTest{
		name: "just above upper boundary 1.001",
		scores: []db.LLMScore{
			TestScore{Model: "left", Score: 0.0, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "center", Score: 0.0, Confidence: 1.0}.ToLLMScore(),
			TestScore{Model: "right", Score: 1.001, Confidence: 1.0}.ToLLMScore(),
		},
		expectedScore: 0.0, // Should be normalized to 0.0
		expectedConf:  1.0,
		expectError:   false,
	})

	// Run all tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, conf, err := calc.CalculateScore(tt.scores, cfg)

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

// Add benchmark for score calculation performance
func BenchmarkScoreCalculation(b *testing.B) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &llm.DefaultScoreCalculator{}
	scores := GenerateScoreSet(-0.8, 0.9)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = calc.CalculateScore(scores, cfg)
	}
}

func TestScoreBoundaries(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore: -1.0,
		MaxScore: 1.0,
		// Add other fields if needed
		Formula: "average", DefaultMissing: 0.0, Models: []llm.ModelConfig{{Perspective: "left", ModelName: "left"}}, ConfidenceMethod: "count_valid",
	}
	calc := &llm.DefaultScoreCalculator{}

	testCases := []struct {
		name          string
		inputScores   []db.LLMScore
		expectedScore float64
	}{
		// ... test cases ...
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, _, err := calc.CalculateScore(tc.inputScores, cfg) // Pass cfg
			assert.NoError(t, err)
			assert.InDelta(t, tc.expectedScore, score, 0.001)
		})
	}
}

func TestScoreBoundariesWithDifferentConfig(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore: 0.0,
		MaxScore: 10.0,
		// Add other fields if needed
		Formula: "average", DefaultMissing: 0.0, Models: []llm.ModelConfig{{Perspective: "left", ModelName: "left"}}, ConfidenceMethod: "count_valid",
	}
	calc := &llm.DefaultScoreCalculator{}

	testCases := []struct {
		name          string
		inputScores   []db.LLMScore
		expectedScore float64
	}{
		// ... test cases ...
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score, _, err := calc.CalculateScore(tc.inputScores, cfg) // Pass cfg
			assert.NoError(t, err)
			assert.InDelta(t, tc.expectedScore, score, 0.001)
		})
	}
}
