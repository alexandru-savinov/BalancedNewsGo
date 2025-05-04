package llm

import (
	"encoding/json"
	"fmt"
	"log"
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
	if normalizedModelName == "left" {
		return "left"
	} else if normalizedModelName == "center" {
		return "center"
	} else if normalizedModelName == "right" {
		return "right"
	}

	log.Printf("Warning: Model '%s' not found in composite score configuration", modelName)
	return ""
}

// checkForAllZeroResponses detects if all LLM responses have zero scores and zero confidence
func checkForAllZeroResponses(scores []db.LLMScore) (bool, error) {
	zeroCount := 0
	totalCount := 0

	for _, score := range scores {
		// Skip ensemble scores in the zero check
		if strings.ToLower(score.Model) == "ensemble" {
			continue
		}

		totalCount++
		if score.Score == 0 {
			// Check if we have non-zero confidence
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(score.Metadata), &metadata); err == nil {
				if confidence, ok := metadata["confidence"].(float64); ok && confidence > 0.0 {
					continue
				}
			}
			zeroCount++
		}
	}

	if totalCount == 0 {
		return false, nil // No non-ensemble scores to check
	}

	if zeroCount == totalCount {
		log.Printf("Critical warning: All %d LLM models returned empty responses or zero values", totalCount)
		return true, fmt.Errorf("all LLMs returned empty or zero-confidence responses (count: %d)", totalCount)
	}

	return false, nil
}

// ComputeCompositeScoreWithConfidenceFixed is an improved version of ComputeCompositeScoreWithConfidence
// that properly maps model names to their perspectives based on the configuration
func ComputeCompositeScoreWithConfidenceFixed(scores []db.LLMScore) (float64, float64, error) {
	// Check for empty scores array
	if len(scores) == 0 {
		return 0, 0, fmt.Errorf("no scores provided")
	}

	// First check if we have all zero responses
	if allZeros, err := checkForAllZeroResponses(scores); allZeros {
		log.Printf("[ERROR][CONFIDENCE] All scores are zero, returning error")
		return 0, 0, err
	}

	// Use the global test config if available (for tests), otherwise load from file
	var cfg *CompositeScoreConfig
	var err error

	if testModelConfig != nil {
		cfg = testModelConfig
	} else {
		cfg, err = LoadCompositeScoreConfig()
		if err != nil {
			log.Printf("[ERROR][CONFIDENCE] Failed to load config: %v", err)
			return 0, 0, fmt.Errorf("loading composite score config: %w", err)
		}
	}

	if cfg == nil {
		log.Printf("[ERROR][CONFIDENCE] Config is nil")
		return 0, 0, fmt.Errorf("composite score config is nil")
	}

	// Map for left/center/right
	scoreMap := map[string]float64{
		"left":   cfg.DefaultMissing,
		"center": cfg.DefaultMissing,
		"right":  cfg.DefaultMissing,
	}

	validCount := 0
	sum := 0.0
	weightedSum := 0.0
	weightTotal := 0.0
	validModels := make(map[string]bool)

	// Process scores by perspective
	perspectiveModels := mapModelsToPerspectives(scores, cfg)
	processScoresByPerspective(perspectiveModels, cfg, scoreMap, &validCount, &validModels)

	// Check if we have any valid scores
	if validCount == 0 {
		log.Printf("[ERROR][CONFIDENCE] No valid model scores found")
		return 0, 0, fmt.Errorf("no valid model scores to compute composite score (input count: %d)", len(scores))
	}

	// Check if we have only ensemble scores
	if validCount == 1 && validModels["center"] && len(perspectiveModels["center"]) > 0 {
		// Check if the only valid score is from an ensemble model
		for _, s := range perspectiveModels["center"] {
			if strings.ToLower(s.Model) == "ensemble" {
				log.Printf("[ERROR][CONFIDENCE] Only ensemble scores found, no valid individual model scores")
				return 0, 0, fmt.Errorf("only ensemble scores found, no valid individual model scores")
			}
		}
	}

	// Calculate weighted sums
	for k, v := range scoreMap {
		score := v
		w := 1.0
		if cfg.Formula == "weighted" {
			if weight, ok := cfg.Weights[k]; ok {
				w = weight
			}
		}
		weightedSum += score * w
		weightTotal += w
		sum += score
	}

	// Calculate composite score
	composite := calculateCompositeScore(cfg, scoreMap, sum, weightedSum, weightTotal)

	// Calculate confidence using the proper calculation function
	confidence := calculateConfidence(cfg, &validModels, scoreMap)

	return composite, confidence, nil
}

// processScoresByPerspective handles selecting the best score for each perspective
func processScoresByPerspective(
	perspectiveModels map[string][]db.LLMScore,
	cfg *CompositeScoreConfig,
	scoreMap map[string]float64,
	validCount *int,
	validModels *map[string]bool) {

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

		// Find the model with highest confidence
		bestScore := findBestConfidenceScore(models)
		log.Printf("Selected for %s: Model: %s, Score: %.2f", perspective, bestScore.Model, bestScore.Score)

		// Use the best score for this perspective
		val := bestScore.Score

		// Only consider a score invalid if it's truly invalid (NaN, +/-Inf)
		// or outside the configured range when specified
		hasScoreRange := cfg.MinScore > -1e9 || cfg.MaxScore < 1e9
		isOutsideRange := hasScoreRange && (val < cfg.MinScore || val > cfg.MaxScore)

		if cfg.HandleInvalid == "ignore" && (isInvalid(val) || isOutsideRange) {
			log.Printf("Ignoring invalid score %.2f for perspective %s", val, perspective)
			continue
		}

		if isInvalid(val) || isOutsideRange {
			val = cfg.DefaultMissing
			log.Printf("Using default value %.2f for invalid score from perspective %s", val, perspective)
		} else {
			log.Printf("Using actual score %.2f for perspective %s from model %s", val, perspective, bestScore.Model)
		}

		log.Printf("Adding score %.2f for perspective %s from model %s", val, perspective, bestScore.Model)
		scoreMap[perspective] = val
		(*validCount)++
		(*validModels)[perspective] = true
	}
}

// mapModelsToPerspectives groups LLM scores by their corresponding perspectives
func mapModelsToPerspectives(scores []db.LLMScore, cfg *CompositeScoreConfig) map[string][]db.LLMScore {
	perspectiveModels := make(map[string][]db.LLMScore)
	for _, s := range scores {
		// Handle ensemble scores specially - map to center perspective
		if strings.ToLower(s.Model) == "ensemble" {
			log.Printf("Mapping ensemble model (score %.2f) to perspective 'center'", s.Score)
			perspectiveModels["center"] = append(perspectiveModels["center"], s)
			continue
		}

		// First try to map the model to its perspective
		perspective := MapModelToPerspective(s.Model, cfg)

		// If mapping failed, try the old way (legacy model names)
		if perspective == "" {
			model := strings.ToLower(s.Model)
			// Direct check for legacy model names - these are the model names themselves
			if model == "left" {
				perspective = "left"
			} else if model == "center" {
				perspective = "center"
			} else if model == "right" {
				perspective = "right"
			} else if model == LabelLeft {
				perspective = "left"
			} else if model == LabelRight {
				perspective = "right"
			} else {
				// Skip unknown models
				log.Printf("Skipping unknown model: %s", s.Model)
				continue
			}
		}

		// Ensure perspective is one of the expected values
		if perspective != "left" && perspective != "center" && perspective != "right" {
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

// findBestConfidenceScore selects the score with highest confidence from a group of models
func findBestConfidenceScore(models []db.LLMScore) db.LLMScore {
	if len(models) == 1 {
		return models[0]
	}

	bestScore := models[0]
	bestConfidence := extractConfidence(bestScore.Metadata)
	allSameConfidence := true

	for _, model := range models[1:] {
		modelConfidence := extractConfidence(model.Metadata)
		if modelConfidence != bestConfidence {
			allSameConfidence = false
		}
		if modelConfidence > bestConfidence {
			bestScore = model
			bestConfidence = modelConfidence
		} else if modelConfidence == bestConfidence {
			if model.Score > bestScore.Score {
				bestScore = model
			}
		}
	}

	// If all confidences are equal, pick the highest score
	if allSameConfidence {
		maxScore := bestScore.Score
		for _, model := range models {
			if model.Score > maxScore {
				bestScore = model
				maxScore = model.Score
			}
		}
	}

	return bestScore
}

// extractConfidence gets the confidence value from model metadata, defaulting to 0.5
func extractConfidence(metadata string) float64 {
	defaultConfidence := 0.5

	var metaMap map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &metaMap); err != nil {
		return defaultConfidence
	}

	if conf, ok := metaMap["confidence"].(float64); ok {
		return conf
	}

	return defaultConfidence
}

// calculateCompositeScore computes the final score based on the configured formula
func calculateCompositeScore(cfg *CompositeScoreConfig, scoreMap map[string]float64, sum float64, weightedSum float64, weightTotal float64) float64 {
	// Check if all scores are 0
	allZeros := true
	for _, score := range scoreMap {
		if score != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		return 0.0
	}

	// Calculate composite score based on formula
	var composite float64
	switch cfg.Formula {
	case "weighted":
		if weightTotal > 0 {
			composite = weightedSum / weightTotal
		} else {
			composite = sum / float64(len(scoreMap))
		}
	case "average":
		composite = sum / float64(len(scoreMap))
	default:
		composite = sum / float64(len(scoreMap))
	}

	// Ensure the score is within bounds
	if composite < cfg.MinScore {
		composite = cfg.MinScore
	}
	if composite > cfg.MaxScore {
		composite = cfg.MaxScore
	}

	return composite
}

// calculateConfidence determines the confidence level based on the configured method
func calculateConfidence(cfg *CompositeScoreConfig, validModels *map[string]bool, scoreMap map[string]float64) float64 {
	if cfg == nil {
		log.Printf("[ERROR][CONFIDENCE] Config is nil in calculateConfidence")
		return 0.0
	}

	// Count how many perspectives we have
	perspectiveCount := 0
	if _, ok := (*validModels)["left"]; ok {
		perspectiveCount++
	}
	if _, ok := (*validModels)["center"]; ok {
		perspectiveCount++
	}
	if _, ok := (*validModels)["right"]; ok {
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
