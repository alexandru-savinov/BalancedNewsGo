package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// ErrAllPerspectivesInvalid is returned when no valid scores are found after processing.
var ErrAllPerspectivesInvalid = errors.New("no valid scores found")

// isInvalid checks if a score is NaN, ±Inf, or outside the configured MinScore/MaxScore bounds.
func isInvalid(v float64, cfg *CompositeScoreConfig) bool {
	return math.IsNaN(v) || math.IsInf(v, 0) || v < cfg.MinScore || v > cfg.MaxScore
}

// MapModelToPerspective maps a model name to its perspective (left, center, right)
// based on the provided composite score configuration.
//
// Matching order:
//  1. If the model name is empty, return the perspective for the first model in the config with an empty name (if any).
//  2. If the normalized model name exactly matches a normalized config model name, return its perspective
//     (first match wins, including duplicates).
//  3. If no exact match, but the normalized model name has a config model name as a prefix
//     (for cases like extra slashes or suffixes), return its perspective (first match wins).
//  4. If no config match, but the normalized model name is "left", "center", or "right", return that as the perspective.
//  5. If none of the above, return an empty string.
func MapModelToPerspective(modelName string, cfg *CompositeScoreConfig) string {
	if cfg == nil {
		log.Printf("Error: CompositeScoreConfig is nil in MapModelToPerspective")
		return ""
	}

	// 1. Handle empty model name - look for a model with empty name in config
	if modelName == "" {
		for _, model := range cfg.Models {
			if model.ModelName == "" {
				return strings.ToLower(model.Perspective)
			}
		}
		return ""
	}

	// Normalize the model name by removing version suffix and whitespace
	normalizedModelName := strings.ToLower(strings.TrimSpace(modelName))
	if colonIndex := strings.Index(normalizedModelName, ":"); colonIndex != -1 {
		normalizedModelName = normalizedModelName[:colonIndex]
	}

	// 2. Look for exact match in the configuration (first match wins)
	for _, model := range cfg.Models {
		normalizedConfigModel := strings.ToLower(strings.TrimSpace(model.ModelName))
		if colonIndex := strings.Index(normalizedConfigModel, ":"); colonIndex != -1 {
			normalizedConfigModel = normalizedConfigModel[:colonIndex]
		}
		if normalizedModelName == normalizedConfigModel {
			return strings.ToLower(model.Perspective)
		}
	}

	// 3. Look for prefix match in the configuration (first match wins)
	for _, model := range cfg.Models {
		normalizedConfigModel := strings.ToLower(strings.TrimSpace(model.ModelName))
		if colonIndex := strings.Index(normalizedConfigModel, ":"); colonIndex != -1 {
			normalizedConfigModel = normalizedConfigModel[:colonIndex]
		}
		if strings.HasPrefix(normalizedModelName, normalizedConfigModel) {
			return strings.ToLower(model.Perspective)
		}
	}

	// 4. Fallback to legacy names
	if normalizedModelName == LabelLeft {
		return LabelLeft
	} else if normalizedModelName == LabelCenter {
		return LabelCenter
	} else if normalizedModelName == LabelRight {
		return LabelRight
	}

	// 5. No match found
	log.Printf("Warning: Model '%s' not found in composite score configuration", modelName)
	return ""
}

// checkForAllZeroResponses detects if all non-ensemble LLM responses have zero confidence.
func checkForAllZeroResponses(scores []db.LLMScore) (bool, error) {
	allZeroConfidence := true // Assume true until proven otherwise
	nonEnsembleCount := 0

	for _, score := range scores {
		// Skip ensemble scores
		if strings.ToLower(score.Model) == "ensemble" {
			continue
		}

		nonEnsembleCount++

		// Check confidence from metadata
		hasNonZeroConfidence := false
		var metadata map[string]interface{}
		if score.Metadata != "" {
			if err := json.Unmarshal([]byte(score.Metadata), &metadata); err == nil {
				if confidenceValue, ok := metadata["confidence"]; ok {
					if confidenceFloat, ok := confidenceValue.(float64); ok && confidenceFloat > 0.0 {
						hasNonZeroConfidence = true
					}
				}
			}
			// Malformed or missing metadata is treated as zero confidence
		}

		if hasNonZeroConfidence {
			allZeroConfidence = false // Found one with non-zero confidence
			break                     // No need to check further
		}
	}

	if nonEnsembleCount == 0 {
		return false, nil // No non-ensemble scores to check
	}

	if allZeroConfidence {
		log.Printf("Critical warning: All %d non-ensemble LLM models returned zero confidence", nonEnsembleCount)
		// Return the specific sentinel error variable, potentially wrapped if needed elsewhere,
		// but the base error should be comparable with errors.Is
		// Using the formatted error caused issues with errors.Is checks.
		// Let's return the sentinel directly for reliable checks.
		return true, ErrAllScoresZeroConfidence
	}

	return false, nil
}

// mapModelsToPerspectives groups scores by perspective based on the configuration
func mapModelsToPerspectives(scores []db.LLMScore, cfg *CompositeScoreConfig) map[string][]db.LLMScore {
	perspectiveModels := make(map[string][]db.LLMScore)
	for _, s := range scores {
		// Handle ensemble scores specially - map to center perspective
		if strings.ToLower(s.Model) == "ensemble" {
			log.Printf("Mapping ensemble model (score %.2f) to perspective '%s'", s.Score, LabelCenter)
			perspectiveModels[LabelCenter] = append(perspectiveModels[LabelCenter], s)
			continue
		}

		// First try to map the model to its perspective
		perspective := MapModelToPerspective(s.Model, cfg)

		// If mapping failed, try the old way (legacy model names)
		if perspective == "" {
			modelLower := strings.ToLower(s.Model)
			// Direct check for legacy model names - these are the model names themselves
			if modelLower == LabelLeft {
				perspective = LabelLeft
			} else if modelLower == LabelCenter || modelLower == LabelNeutral {
				perspective = LabelCenter
			} else if modelLower == LabelRight {
				perspective = LabelRight
			} else {
				// Skip unknown models
				log.Printf("Skipping unknown model: %s", s.Model)
				continue
			}
		}

		// Ensure perspective is one of the expected values
		if perspective != LabelLeft && perspective != LabelCenter && perspective != LabelRight {
			log.Printf("Skipping model with invalid perspective: %s -> %s", s.Model, perspective)
			continue
		}

		// Add to perspective models map
		log.Printf("Mapping model '%s' (score %.2f) to perspective '%s'", s.Model, s.Score, perspective)
		perspectiveModels[perspective] = append(perspectiveModels[perspective], s)
	}

	// Log the perspective mapping results
	for perspective, models := range perspectiveModels {
		log.Printf("Perspective %s has %d models", perspective, len(models))
	}

	return perspectiveModels
}

// calculateCompositeScore calculates the final composite score based on the configuration and intermediate values
func calculateCompositeScore(cfg *CompositeScoreConfig, scoreMap map[string]float64, sum float64, weightedSum float64,
	weightTotal float64, actualValidCount int, validModels map[string]bool) (float64, error) {
	if actualValidCount == 0 {
		return 0.0, ErrAllPerspectivesInvalid
	}

	// Check if all scores *that were included* are 0
	allZeros := true
	// No need to check validModels here again, as the loop calling this already filtered
	for _, score := range scoreMap { // Iterate over potentially reduced map if we changed the loop above
		// Need to adjust this loop if the sum/weightedSum calculation already filtered
		if score != 0 {
			allZeros = false
			break
		}
	}
	if allZeros && actualValidCount > 0 { // Check count > 0 to avoid returning 0 for empty valid set
		return 0.0, ErrAllPerspectivesInvalid
	}

	// If only one valid score, return it directly (avoids averaging with defaults)
	if actualValidCount == 1 {
		for p, score := range scoreMap {
			if _, isValid := validModels[p]; isValid { // Still need this check to find the *one* valid score
				// Apply bounds before returning
				if score < cfg.MinScore {
					return cfg.MinScore, nil
				}
				if score > cfg.MaxScore {
					return cfg.MaxScore, nil
				}
				return score, nil
			}
		}
	}

	// Calculate composite score based on formula
	var composite float64
	switch cfg.Formula {
	case "weighted":
		if weightTotal > 0 {
			composite = weightedSum / weightTotal
		} else { // Fallback if weights are zero or missing
			composite = sum / float64(actualValidCount)
		}
	case "average":
		composite = sum / float64(actualValidCount) // Use actualValidCount
	default:
		composite = sum / float64(actualValidCount) // Use actualValidCount
	}

	// Ensure the score is within bounds
	if composite < cfg.MinScore {
		composite = cfg.MinScore
	}
	if composite > cfg.MaxScore {
		composite = cfg.MaxScore
	}

	return composite, nil
}

// calculateConfidence calculates the final confidence score based on the configuration and intermediate values
func calculateConfidence(cfg *CompositeScoreConfig, validModels map[string]bool, scoreMap map[string]float64) float64 {
	if cfg == nil {
		log.Printf("[ERROR][CONFIDENCE] Config is nil in calculateConfidence")
		return 0.0
	}

	// Count how many perspectives we have
	perspectiveCount := 0
	if _, ok := validModels[LabelLeft]; ok {
		perspectiveCount++
	}
	if _, ok := validModels[LabelCenter]; ok {
		perspectiveCount++
	}
	if _, ok := validModels[LabelRight]; ok {
		perspectiveCount++
	}

	var confidence float64
	switch cfg.ConfidenceMethod {
	case "count_valid":
		// Note: if cfg.HandleInvalid == "ignore", perspectives with invalid scores
		// that are skipped will reduce perspectiveCount, thus lowering confidence.
		confidence = float64(perspectiveCount) / 3.0
	case "spread":
		spread := scoreSpread(scoreMap)
		confidence = 1.0 - spread
	default:
		confidence = float64(perspectiveCount) / 3.0
	}

	// Only apply confidence limits if we don't have all perspectives
	if perspectiveCount < 3 {
		if confidence < cfg.MinConfidence {
			confidence = cfg.MinConfidence
		}
		if confidence > cfg.MaxConfidence {
			confidence = cfg.MaxConfidence
		}
	}

	return confidence
}

// ComputeCompositeScoreWithConfidenceFixed calculates the composite score and confidence based on provided scores and configuration
func ComputeCompositeScoreWithConfidenceFixed(scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
	// Check for empty scores array
	if len(scores) == 0 {
		return 0, 0, fmt.Errorf("no scores provided: %w", ErrAllPerspectivesInvalid)
	}

	// First check if we have all zero responses
	if allZeros, err := checkForAllZeroResponses(scores); allZeros {
		log.Printf("[ERROR][CONFIDENCE] All scores are zero, returning error")
		return 0, 0, fmt.Errorf("%w: %w", ErrAllPerspectivesInvalid, err)
	}

	// Use the provided config directly
	if cfg == nil {
		log.Printf("[ERROR][CONFIDENCE] Config is nil")
		return 0, 0, fmt.Errorf("composite score config is nil: %w", ErrAllPerspectivesInvalid)
	}

	// --------------------------------------------------
	// 0. Group all valid scores and confidences by perspective (for averaging)
	// --------------------------------------------------
	perspectiveScores := map[string][]float64{"left": {}, "center": {}, "right": {}}
	perspectiveConfs := map[string][]float64{"left": {}, "center": {}, "right": {}}
	for _, s := range scores {
		if math.IsNaN(s.Score) || math.IsInf(s.Score, 0) || s.Score < cfg.MinScore || s.Score > cfg.MaxScore {
			if cfg.HandleInvalid == "ignore" {
				continue
			}
			return 0, 0, ErrAllPerspectivesInvalid
		}
		p := MapModelToPerspective(s.Model, cfg)
		if p == "" {
			continue
		}
		perspectiveScores[p] = append(perspectiveScores[p], s.Score)
		// Parse confidence from metadata
		conf := 0.0
		if s.Metadata != "" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(s.Metadata), &meta); err == nil {
				if c, ok := meta["confidence"]; ok {
					switch v := c.(type) {
					case float64:
						conf = v
					case int:
						conf = float64(v)
					}
				}
			}
		}
		perspectiveConfs[p] = append(perspectiveConfs[p], conf)
	}

	// Average scores and confidences for each perspective
	scoreMap := map[string]float64{"left": 0, "center": 0, "right": 0}
	confMap := map[string]float64{"left": 0, "center": 0, "right": 0}
	validModels := make(map[string]bool)
	for p, scores := range perspectiveScores {
		if len(scores) > 0 {
			sum := 0.0
			for _, v := range scores {
				sum += v
			}
			scoreMap[p] = sum / float64(len(scores))
			validModels[p] = true
		}
	}
	for p, confs := range perspectiveConfs {
		if len(confs) > 0 {
			sum := 0.0
			for _, v := range confs {
				sum += v
			}
			confMap[p] = sum / float64(len(confs))
		}
	}
	actualValidCount := len(validModels)

	if actualValidCount == 0 {
		return cfg.DefaultMissing, 0.0, ErrAllPerspectivesInvalid
	}

	// Calculate sums based ONLY on valid models
	sum := 0.0
	weightedSum := 0.0
	weightTotal := 0.0
	confSum := 0.0
	confCount := 0
	for perspective, score := range scoreMap {
		if _, isValid := validModels[perspective]; isValid {
			w := 1.0
			if cfg.Formula == "weighted" {
				if weight, ok := cfg.Weights[perspective]; ok {
					w = weight
				}
			}
			weightedSum += score * w
			weightTotal += w
			sum += score
			// Add confidence for this perspective
			confSum += confMap[perspective]
			confCount++
		}
	}

	compositeScore, calcErr := calculateCompositeScore(cfg, scoreMap, sum, weightedSum, weightTotal, actualValidCount, validModels)
	if calcErr != nil {
		return 0.0, 0.0, calcErr
	}
	confidence := 0.0
	if confCount > 0 {
		confidence = confSum / float64(confCount)
	}
	return compositeScore, confidence, nil
}
