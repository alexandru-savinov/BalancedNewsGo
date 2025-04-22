package unit

import (
	"encoding/json"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/stretchr/testify/assert"
)

type testScoreCalculator struct {
	*llm.DefaultScoreCalculator
}

// extractConfidence extracts confidence value from score metadata
func (c *testScoreCalculator) extractConfidence(metadata string) float64 {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &meta); err != nil {
		return 0.0
	}

	if conf, ok := meta["confidence"]; ok {
		switch v := conf.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		default:
			return 0.0
		}
	}
	return 0.0
}

func TestExtractConfidence(t *testing.T) {
	cfg := &llm.CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	calc := &testScoreCalculator{&llm.DefaultScoreCalculator{Config: cfg}}

	tests := []struct {
		name         string
		scores       []db.LLMScore
		expectedConf float64
		description  string
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
		},
		{
			name: "empty metadata",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: ""},
			},
			expectedConf: 0.0,
			description:  "Tests empty metadata string",
		},
		{
			name: "string confidence value",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": "0.8"}`},
			},
			expectedConf: 0.0,
			description:  "Tests string instead of number",
		},
		{
			name: "boolean confidence value",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": true}`},
			},
			expectedConf: 0.0,
			description:  "Tests boolean instead of number",
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
			expectedConf: 0.5,
			description:  "Tests boundary confidence values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, conf, err := calc.CalculateScore(tt.scores)
			assert.NoError(t, err)
			assert.InDelta(t, tt.expectedConf, conf, 0.001, tt.description)
		})
	}
}
