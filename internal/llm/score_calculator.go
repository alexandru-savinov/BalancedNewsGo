package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

var perspectives = []string{"left", "center", "right"}

// ScoreCalculator defines the interface for composite score calculation
// Returns (score, confidence, error)
type ScoreCalculator interface {
	CalculateScore(scores []db.LLMScore) (float64, float64, error)
}

// DefaultScoreCalculator implements ScoreCalculator using the new averaging logic
// It preserves the -1.0 to +1.0 scale and averages confidences from model metadata
// Missing perspectives are treated as 0 for both score and confidence
type DefaultScoreCalculator struct {
	Config *CompositeScoreConfig // Must be provided, not nil
}

// initializeMaps creates and initializes maps for scores and confidence values
func (c *DefaultScoreCalculator) initializeMaps() (map[string]*float64, map[string]*float64) {
	scoreMap := make(map[string]*float64)
	confMap := make(map[string]*float64)
	for _, p := range perspectives {
		scoreMap[p] = nil
		confMap[p] = nil
	}
	return scoreMap, confMap
}

// getPerspective determines the perspective (left/center/right) for a given model
func (c *DefaultScoreCalculator) getPerspective(model string) string {
	perspective := MapModelToPerspective(model, c.Config)
	if perspective != "" {
		return perspective
	}

	model = strings.ToLower(model)
	switch model {
	case LabelLeft:
		return "left"
	case LabelRight:
		return "right"
	case "center":
		return "center"
	default:
		return ""
	}
}

// extractConfidence extracts confidence value from score metadata
func (c *DefaultScoreCalculator) extractConfidence(metadata string) float64 {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &meta); err != nil {
		return 0.0
	}
	if conf, ok := meta["confidence"]; ok {
		// Handle both float64 and integer confidence values
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

func (c *DefaultScoreCalculator) CalculateScore(scores []db.LLMScore) (float64, float64, error) {
	if c.Config == nil {
		return 0, 0, fmt.Errorf("DefaultScoreCalculator: Config must not be nil")
	}

	scoreMap, confMap := c.initializeMaps()

	// Process each score
	for _, s := range scores {
		perspective := c.getPerspective(s.Model)
		if perspective == "" || perspective != "left" && perspective != "center" && perspective != "right" {
			continue
		}

		val := s.Score
		if isInvalid(val) || val < c.Config.MinScore || val > c.Config.MaxScore {
			val = 0.0
		}
		scoreMap[perspective] = &val

		conf := c.extractConfidence(s.Metadata)
		confMap[perspective] = &conf
	}

	// Calculate final score and confidence
	sum := 0.0
	confSum := 0.0
	for _, p := range perspectives {
		if scoreMap[p] != nil {
			sum += *scoreMap[p]
		} // else default 0
		if confMap[p] != nil {
			confSum += *confMap[p]
		} // else default 0
	}

	return sum / 3.0, confSum / 3.0, nil
}
