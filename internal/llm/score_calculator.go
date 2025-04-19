package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// ScoreCalculator defines the interface for composite score calculation
// Returns (score, confidence, error)
type ScoreCalculator interface {
	CalculateScore(scores []db.LLMScore) (float64, float64, error)
}

// DefaultScoreCalculator implements ScoreCalculator using the new averaging logic
// It preserves the -1.0 to +1.0 scale and averages confidences from model metadata
// Missing perspectives are treated as 0 for both score and confidence
// This implementation is pluggable for future algorithm changes
type DefaultScoreCalculator struct {
	Config *CompositeScoreConfig // Must be provided, not nil
}

func (c *DefaultScoreCalculator) CalculateScore(scores []db.LLMScore) (float64, float64, error) {
	if c.Config == nil {
		return 0, 0, fmt.Errorf("DefaultScoreCalculator: Config must not be nil")
	}

	scoreMap := map[string]*float64{
		"left":   nil,
		"center": nil,
		"right":  nil,
	}
	confMap := map[string]*float64{
		"left":   nil,
		"center": nil,
		"right":  nil,
	}

	for _, s := range scores {
		perspective := MapModelToPerspective(s.Model, c.Config)
		if perspective == "" {
			model := strings.ToLower(s.Model)
			if model == "left" || model == LabelLeft {
				perspective = "left"
			} else if model == "right" || model == LabelRight {
				perspective = "right"
			} else if model == "center" {
				perspective = "center"
			} else {
				continue
			}
		}
		if perspective != "left" && perspective != "center" && perspective != "right" {
			continue
		}
		val := s.Score
		if isInvalid(val) || val < c.Config.MinScore || val > c.Config.MaxScore {
			val = 0.0
		}
		scoreMap[perspective] = &val

		// Extract confidence from metadata if available
		conf := 0.0
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(s.Metadata), &meta); err == nil {
			if v, ok := meta["confidence"].(float64); ok {
				conf = v
			}
		}
		confMap[perspective] = &conf
	}

	sum := 0.0
	confSum := 0.0
	for _, p := range []string{"left", "center", "right"} {
		if scoreMap[p] != nil {
			sum += *scoreMap[p]
		} // else default 0
		if confMap[p] != nil {
			confSum += *confMap[p]
		} // else default 0
	}

	score := sum / 3.0
	confidence := confSum / 3.0
	return score, confidence, nil
}
