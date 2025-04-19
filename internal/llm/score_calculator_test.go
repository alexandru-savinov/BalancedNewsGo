package llm

import (
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestDefaultScoreCalculator_CalculateScore(t *testing.T) {
	cfg := &CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &DefaultScoreCalculator{Config: cfg}
	tests := []struct {
		name          string
		scores        []db.LLMScore
		expectedScore float64
		expectedConf  float64
		expectError   bool
	}{
		{
			name: "All perspectives present",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8, Metadata: `{"confidence": 0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`},
				{Model: "right", Score: 0.6, Metadata: `{"confidence": 0.7}`},
			},
			expectedScore: -0.067, // (-0.8 + 0.0 + 0.6) / 3
			expectedConf:  0.8,    // (0.9 + 0.8 + 0.7) / 3
			expectError:   false,
		},
		{
			name: "Missing perspective",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8, Metadata: `{"confidence": 0.9}`},
				{Model: "right", Score: 0.6, Metadata: `{"confidence": 0.7}`},
			},
			expectedScore: -0.067, // (-0.8 + 0.0 + 0.6) / 3
			expectedConf:  0.533,  // (0.9 + 0.0 + 0.7) / 3
			expectError:   false,
		},
		{
			name:          "No valid scores",
			scores:        []db.LLMScore{},
			expectError:   false, // Should not error, just return 0,0
			expectedScore: 0.0,
			expectedConf:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, conf, err := calc.CalculateScore(tt.scores)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expectedScore, score, 0.01)
				assert.InDelta(t, tt.expectedConf, conf, 0.01)
			}
		})
	}
}
