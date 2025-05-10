package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

var perspectives = []string{LabelLeft, LabelCenter, LabelRight}

// ScoreCalculator defines the interface for composite score calculation
// Returns (score, confidence, error)
type ScoreCalculator interface {
	CalculateScore(scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error)
}

// DefaultScoreCalculator implements ScoreCalculator using the new averaging logic
// It preserves the -1.0 to +1.0 scale and averages confidences from model metadata
// Missing perspectives are treated as 0 for both score and confidence
type DefaultScoreCalculator struct {
	// Config *CompositeScoreConfig // Config is now passed via method
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
func (c *DefaultScoreCalculator) getPerspective(model string, cfg *CompositeScoreConfig) string {
	perspective := MapModelToPerspective(model, cfg)
	if perspective != "" {
		// Ensure mapped perspective is one of the known constants
		switch perspective {
		case LabelLeft, LabelCenter, LabelRight:
			return perspective
		default:
			// If mapped perspective is not a known constant, log and fall through
			log.Printf("[WARN] Mapped perspective '%s' for model '%s' is not a standard label. Falling back to default logic.", perspective, model)
		}
	}

	model = strings.ToLower(model)
	// Allow model names to directly match constants (e.g. "left", "center", "right" from config)
	// or common variations.
	switch model {
	case LabelLeft, "left_leaning", "liberal": // "left" is already LabelLeft
		return LabelLeft
	case LabelRight, "right_leaning", "conservative": // "right" is already LabelRight
		return LabelRight
	case LabelCenter, "centrist", "neutral": // "center" is already LabelCenter
		return LabelCenter
	default:
		// Try matching based on contains, as a fallback
		if strings.Contains(model, LabelLeft) {
			return LabelLeft
		}
		if strings.Contains(model, LabelRight) {
			return LabelRight
		}
		if strings.Contains(model, LabelCenter) {
			return LabelCenter
		}
		return ""
	}
}

// extractConfidence extracts confidence value from score metadata
func (c *DefaultScoreCalculator) extractConfidence(metadata string) float64 {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &meta); err != nil {
		log.Printf("[ERROR][CONFIDENCE] Failed to parse metadata JSON: %v", err)
		return 0.0
	}

	conf, exists := meta["confidence"]
	if !exists {
		log.Printf("[DEBUG][CONFIDENCE] No confidence field in metadata, defaulting to 0.0")
		return 0.0
	}

	switch v := conf.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		log.Printf("[WARN][CONFIDENCE] String confidence value '%s', defaulting to 0.0", v)
		return 0.0
	case bool:
		log.Printf("[WARN][CONFIDENCE] Boolean confidence value %v, defaulting to 0.0", v)
		return 0.0
	case nil:
		log.Printf("[DEBUG][CONFIDENCE] Null confidence value, defaulting to 0.0")
		return 0.0
	default:
		log.Printf("[WARN][CONFIDENCE] Unknown confidence type %T, defaulting to 0.0", v)
		return 0.0
	}
}

func (c *DefaultScoreCalculator) CalculateScore(scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
	if cfg == nil {
		return 0.0, 0.0, fmt.Errorf("DefaultScoreCalculator: Config must not be nil: %w", ErrAllPerspectivesInvalid)
	}

	if len(scores) == 0 {
		return 0.0, 0.0, ErrAllPerspectivesInvalid
	}

	// Initialize maps for scores and confidences
	scoreMap, confMap := c.initializeMaps()

	// Process each score
	validCount := 0
	var sumScore float64
	var sumConf float64

	for _, score := range scores {
		perspective := c.getPerspective(score.Model, cfg)
		if perspective == "" {
			log.Printf("Warning: Model '%s' not found in composite score configuration", score.Model)
			continue
		}

		// Check if score is valid
		if math.IsNaN(score.Score) || math.IsInf(score.Score, 0) || score.Score < cfg.MinScore || score.Score > cfg.MaxScore {
			log.Printf("[DEBUG][CONFIDENCE] Ignoring invalid score %.2f for model %s", score.Score, score.Model)
			continue
		}

		// Extract confidence from metadata
		confidence := c.extractConfidence(score.Metadata)
		if confidence == 0.0 {
			log.Printf("[DEBUG][CONFIDENCE] No confidence field in metadata, defaulting to 0.0")
			continue
		}

		// Store the score and confidence
		scoreMap[perspective] = &score.Score
		confMap[perspective] = &confidence

		validCount++
		sumScore += score.Score
		sumConf += confidence
	}

	if validCount == 0 {
		return 0.0, 0.0, ErrAllPerspectivesInvalid
	}

	// Calculate average score and confidence
	avgScore := sumScore / float64(validCount)
	avgConf := sumConf / float64(validCount)

	if avgConf == 0.0 {
		return 0.0, 0.0, ErrAllPerspectivesInvalid
	}

	log.Printf("[DEBUG][CONFIDENCE] Calculated composite score: %.4f with confidence %.4f from %d valid scores", avgScore, avgConf, validCount)
	return avgScore, avgConf, nil
}
