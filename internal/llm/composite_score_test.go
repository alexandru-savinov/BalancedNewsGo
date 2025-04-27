package llm

import (
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
)

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
