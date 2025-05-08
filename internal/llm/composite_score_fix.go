package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// MapModelToPerspective maps a model name to its perspective (left, center, right)
// based on the provided composite score configuration
func MapModelToPerspective(modelName string, cfg *CompositeScoreConfig) string {
	if cfg == nil {
		log.Printf("Error: CompositeScoreConfig is nil in MapModelToPerspective")
		return ""
	}

	// Normalize the model name by removing version suffix and whitespace
	normalizedModelName := strings.ToLower(strings.TrimSpace(modelName))
	if colonIndex := strings.Index(normalizedModelName, ":"); colonIndex != -1 {
		normalizedModelName = normalizedModelName[:colonIndex]
	}

	// Look up the model in the configuration
	for _, model := range cfg.Models {
		// Normalize the configured model name the same way
		normalizedConfigModel := strings.ToLower(strings.TrimSpace(model.ModelName))
		if colonIndex := strings.Index(normalizedConfigModel, ":"); colonIndex != -1 {
			normalizedConfigModel = normalizedConfigModel[:colonIndex]
		}

		if normalizedConfigModel == normalizedModelName {
			return strings.ToLower(model.Perspective)
		}
	}

	// Fallback to legacy names
	if normalizedModelName == LabelLeft {
		return LabelLeft
	} else if normalizedModelName == LabelCenter {
		return LabelCenter
	} else if normalizedModelName == LabelRight {
		return LabelRight
	}

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

// processScoresByPerspective processes scores and updates maps based on the provided configuration
func processScoresByPerspective(perspectiveModels map[string][]db.LLMScore, cfg *CompositeScoreConfig, scoreMap map[string]float64, validCount *int, validModels map[string]bool) {
	// Use the best score from each perspective
	for perspective, models := range perspectiveModels {
		if len(models) == 0 {
			log.Printf("No models found for perspective %s", perspective)
			continue
		}

		log.Printf("Candidates for %s: ", perspective)
		for _, m := range models {
			log.Printf("  Model: %s, Score: %.2f, Metadata: %s", m.Model, m.Score, m.Metadata)
		}

		// Sort models by created_at descending (or another criteria if needed)
		sort.Slice(models, func(i, j int) bool {
			return models[i].CreatedAt.After(models[j].CreatedAt)
		})

		// Select the first (latest) valid score for this perspective
		for _, s := range models {
			if isInvalid(s.Score) {
				if cfg.HandleInvalid == "ignore" {
					continue
				} else { // Default to default value
					scoreMap[perspective] = cfg.DefaultMissing
					(*validCount)++                 // Count this perspective as valid because we are using a default
					validModels[perspective] = true // Mark as valid for averaging
					break                           // Use default, don't look further for this perspective
				}
			} else {
				// Valid score found
				scoreMap[perspective] = s.Score
				(*validCount)++
				validModels[perspective] = true // Mark perspective as valid
				break                           // Use this score, don't look further
			}
		}
	}
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
func calculateCompositeScore(cfg *CompositeScoreConfig, scoreMap map[string]float64, sum float64, weightedSum float64, weightTotal float64, actualValidCount int, validModels map[string]bool) (float64, error) {
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
		return 0, 0, fmt.Errorf("no scores provided")
	}

	// First check if we have all zero responses
	if allZeros, err := checkForAllZeroResponses(scores); allZeros {
		log.Printf("[ERROR][CONFIDENCE] All scores are zero, returning error")
		return 0, 0, err
	}

	// Use the provided config directly
	if cfg == nil {
		log.Printf("[ERROR][CONFIDENCE] Config is nil")
		return 0, 0, fmt.Errorf("composite score config is nil")
	}

	// Map for left/center/right
	scoreMap := map[string]float64{
		LabelLeft:   cfg.DefaultMissing,
		LabelCenter: cfg.DefaultMissing,
		LabelRight:  cfg.DefaultMissing,
	}

	validCount := 0
	sum := 0.0
	weightedSum := 0.0
	weightTotal := 0.0
	validModels := make(map[string]bool)

	// Process scores by perspective
	perspectiveModels := mapModelsToPerspectives(scores, cfg)
	processScoresByPerspective(perspectiveModels, cfg, scoreMap, &validCount, validModels)

	// Check if no valid scores were found after processing
	if validCount == 0 {
		// Handle the case where only invalid scores existed and were ignored/defaulted
		log.Printf("[WARN][CONFIDENCE] No valid model scores found after processing. Returning default score.")
		return cfg.DefaultMissing, 0.0, fmt.Errorf("no valid scores found") // Return default and 0 confidence
	}

	// Check if we have only ensemble scores (This check might be redundant now with validCount check above)
	// if validCount == 1 && validModels["center"] { ... } // Consider removing if validCount handles it

	// Calculate sums based ONLY on valid models
	sum = 0.0
	weightedSum = 0.0
	weightTotal = 0.0
	actualValidCount := 0 // Use a new counter for the loop

	for perspective, score := range scoreMap {
		// Only include scores from perspectives that had valid models processed
		// AND handle 'ignore_invalid' - skip perspectives that were marked invalid
		// (scoreMap might hold default value if invalid score was encountered and not ignored)
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
			actualValidCount++
		}
	}

	// Handle division by zero if somehow actualValidCount is 0 despite validCount > 0
	if actualValidCount == 0 {
		log.Printf("[ERROR][CONFIDENCE] Logic error: validCount > 0 but actualValidCount is 0.")
		return cfg.DefaultMissing, 0.0, fmt.Errorf("internal calculation error: no valid scores counted")
	}

	// Calculate composite score using the new actualValidCount if needed
	compositeScore, calcErr := calculateCompositeScore(cfg, scoreMap, sum, weightedSum, weightTotal, actualValidCount, validModels)
	if calcErr != nil {
		return 0.0, 0.0, calcErr
	}

	// Calculate confidence using the proper calculation function
	confidence := calculateConfidence(cfg, validModels, scoreMap)

	return compositeScore, confidence, nil
}
