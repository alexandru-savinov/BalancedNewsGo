package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// MapModelToPerspective maps a model name to its perspective (left, center, right)
// based on the composite score configuration
func MapModelToPerspective(modelName string) string {
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		log.Printf("Error loading composite score config: %v", err)
		return ""
	}

	// Normalize model name for comparison
	normalizedModelName := strings.ToLower(strings.TrimSpace(modelName))

	// Look up the model in the configuration
	for _, model := range cfg.Models {
		if strings.ToLower(strings.TrimSpace(model.ModelName)) == normalizedModelName {
			return strings.ToLower(model.Perspective)
		}
	}

	// If not found, log a warning and return empty string
	log.Printf("Warning: Model '%s' not found in composite score configuration", modelName)
	return ""
}

// ComputeCompositeScoreWithConfidenceFixed is an improved version of ComputeCompositeScoreWithConfidence
// that properly maps model names to their perspectives based on the configuration
func ComputeCompositeScoreWithConfidenceFixed(scores []db.LLMScore) (float64, float64, error) {
	cfg, err := LoadCompositeScoreConfig()
	if err != nil {
		return 0, 0, fmt.Errorf("loading composite score config: %w", err)
	}

	// Map for left/center/right
	scoreMap := map[string]*float64{
		"left":   nil,
		"center": nil,
		"right":  nil,
	}
	validCount := 0
	sum := 0.0
	weightedSum := 0.0
	weightTotal := 0.0
	validModels := make(map[string]bool)

	// Log the scores we're processing
	log.Printf("ComputeCompositeScoreWithConfidenceFixed: Processing %d scores", len(scores))
	for i, s := range scores {
		log.Printf("Score[%d]: Model=%s, Score=%.2f", i, s.Model, s.Score)
	}

	// First pass: Map models to their perspectives and count valid models per perspective
	perspectiveModels := make(map[string][]db.LLMScore)
	for _, s := range scores {
		// Skip ensemble scores
		if strings.ToLower(s.Model) == "ensemble" {
			continue
		}

		// First try to map the model to its perspective
		perspective := MapModelToPerspective(s.Model)

		// If mapping failed, try the old way
		if perspective == "" {
			model := strings.ToLower(s.Model)
			if model == LabelLeft || model == "left" {
				perspective = "left"
			} else if model == LabelRight || model == "right" {
				perspective = "right"
			} else if model == "center" {
				perspective = "center"
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
		perspectiveModels[perspective] = append(perspectiveModels[perspective], s)
	}

	// Log the perspective mapping results
	for perspective, models := range perspectiveModels {
		log.Printf("Perspective %s has %d models", perspective, len(models))
	}

	// Second pass: Use the best score from each perspective
	for perspective, models := range perspectiveModels {
		if len(models) == 0 {
			continue
		}

		// Find the model with highest confidence
		bestScore := models[0]
		for _, model := range models[1:] {
			// Extract confidence from metadata if available
			bestConfidence := 0.5  // Default confidence
			modelConfidence := 0.5 // Default confidence

			var bestMeta map[string]interface{}
			if err := json.Unmarshal([]byte(bestScore.Metadata), &bestMeta); err == nil {
				if conf, ok := bestMeta["confidence"].(float64); ok {
					bestConfidence = conf
				}
			}

			var modelMeta map[string]interface{}
			if err := json.Unmarshal([]byte(model.Metadata), &modelMeta); err == nil {
				if conf, ok := modelMeta["confidence"].(float64); ok {
					modelConfidence = conf
				}
			}

			// Use the model with higher confidence
			if modelConfidence > bestConfidence {
				bestScore = model
			}
		}

		// Use the best score for this perspective
		val := bestScore.Score
		if cfg.HandleInvalid == "ignore" && (isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore) {
			log.Printf("Ignoring invalid score %.2f for perspective %s", val, perspective)
			continue
		}
		if isInvalid(val) || val < cfg.MinScore || val > cfg.MaxScore {
			val = cfg.DefaultMissing
			log.Printf("Using default value %.2f for invalid score from perspective %s", val, perspective)
		}

		log.Printf("Adding score %.2f for perspective %s from model %s", val, perspective, bestScore.Model)
		scoreMap[perspective] = &val
		validCount++
		validModels[perspective] = true
	}

	if validCount == 0 {
		return 0, 0, fmt.Errorf("no valid model scores to compute composite score (input count: %d)", len(scores))
	}

	// Use weights if formula is weighted
	for k, v := range scoreMap {
		score := cfg.DefaultMissing
		if v != nil {
			score = *v
		}
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

	var composite float64
	switch cfg.Formula {
	case "average":
		composite = sum / 3.0
	case "weighted":
		if weightTotal > 0 {
			composite = weightedSum / weightTotal
		} else {
			composite = 0.0
		}
	case "min":
		composite = minNonNil(scoreMap, cfg.DefaultMissing)
	case "max":
		composite = maxNonNil(scoreMap, cfg.DefaultMissing)
	default:
		composite = sum / 3.0
	}

	// Composite favors balance (closer to center = higher score)
	composite = 1.0 - abs(composite)

	// Confidence metric
	var confidence float64
	switch cfg.ConfidenceMethod {
	case "count_valid":
		confidence = float64(len(validModels)) / 3.0
	case "spread":
		confidence = 1.0 - scoreSpread(scoreMap)
	default:
		confidence = float64(len(validModels)) / 3.0
	}
	if confidence < cfg.MinConfidence {
		confidence = cfg.MinConfidence
	}
	if confidence > cfg.MaxConfidence {
		confidence = cfg.MaxConfidence
	}

	log.Printf("ComputeCompositeScoreWithConfidenceFixed: Final composite=%.2f, confidence=%.2f", composite, confidence)
	return composite, confidence, nil
}
