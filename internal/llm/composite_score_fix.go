package llm

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// isInvalid checks if a score is NaN, ±Inf, or outside the configured MinScore/MaxScore bounds.
func isInvalid(v float64, cfg *CompositeScoreConfig) bool {
	return math.IsNaN(v) || math.IsInf(v, 0) || v < cfg.MinScore || v > cfg.MaxScore
}

// normalizeModelName removes version suffix and whitespace from model name
func normalizeModelName(modelName string) string {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	if colonIndex := strings.Index(normalized, ":"); colonIndex != -1 {
		normalized = normalized[:colonIndex]
	}
	return normalized
}

// findExactMatch looks for exact match in configuration models
func findExactMatch(normalizedName string, cfg *CompositeScoreConfig) string {
	for _, model := range cfg.Models {
		normalizedConfigModel := normalizeModelName(model.ModelName)
		if normalizedName == normalizedConfigModel {
			return strings.ToLower(model.Perspective)
		}
	}
	return ""
}

// findPrefixMatch looks for prefix match in configuration models
func findPrefixMatch(normalizedName string, cfg *CompositeScoreConfig) string {
	for _, model := range cfg.Models {
		normalizedConfigModel := normalizeModelName(model.ModelName)
		if strings.HasPrefix(normalizedName, normalizedConfigModel) {
			return strings.ToLower(model.Perspective)
		}
	}
	return ""
}

// checkLegacyNames checks if model name matches legacy perspective names
func checkLegacyNames(normalizedName string) string {
	switch normalizedName {
	case LabelLeft:
		return LabelLeft
	case LabelCenter:
		return LabelCenter
	case LabelRight:
		return LabelRight
	default:
		return ""
	}
}

// handleEmptyModelName handles the case when model name is empty
func handleEmptyModelName(cfg *CompositeScoreConfig) string {
	for _, model := range cfg.Models {
		if model.ModelName == "" {
			return strings.ToLower(model.Perspective)
		}
	}
	return ""
}

// MapModelToPerspective maps a model name to its perspective (left, center, right)
// based on the provided composite score configuration.
//
// Matching order:
//  1. If the model name is empty, return the perspective for the first model in the config with an empty name (if any).
//  2. If the normalized model name exactly matches a normalized config model name, return its perspective (first match wins, including duplicates).
//  3. If no exact match, but the normalized model name has a config model name as a prefix (for cases like extra slashes or suffixes), return its perspective (first match wins).
//  4. If no config match, but the normalized model name is "left", "center", or "right", return that as the perspective.
//  5. If none of the above, return an empty string.
func MapModelToPerspective(modelName string, cfg *CompositeScoreConfig) string {
	if cfg == nil {
		log.Printf("Error: CompositeScoreConfig is nil in MapModelToPerspective")
		return ""
	}

	// 1. Handle empty model name
	if modelName == "" {
		return handleEmptyModelName(cfg)
	}

	// 2. Normalize the model name
	normalizedModelName := normalizeModelName(modelName)

	// 3. Look for exact match in the configuration
	if perspective := findExactMatch(normalizedModelName, cfg); perspective != "" {
		return perspective
	}

	// 4. Look for prefix match in the configuration
	if perspective := findPrefixMatch(normalizedModelName, cfg); perspective != "" {
		return perspective
	}

	// 5. Fallback to legacy names
	if perspective := checkLegacyNames(normalizedModelName); perspective != "" {
		return perspective
	}

	// 6. No match found
	log.Printf("Warning: Model '%s' not found in composite score configuration", modelName)
	return ""
}

// isEnsembleModel checks if a model is an ensemble model
func isEnsembleModel(modelName string) bool {
	return strings.ToLower(modelName) == "ensemble"
}

// hasNonZeroConfidenceInMetadata checks if score has non-zero confidence in metadata
func hasNonZeroConfidenceInMetadata(score db.LLMScore) bool {
	if score.Metadata == "" {
		return false
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(score.Metadata), &metadata); err != nil {
		return false // Malformed metadata treated as zero confidence
	}

	confidenceValue, ok := metadata["confidence"]
	if !ok {
		return false
	}

	confidenceFloat, ok := confidenceValue.(float64)
	return ok && confidenceFloat > 0.0
}

// countNonEnsembleScores counts non-ensemble scores and checks for zero confidence
func countNonEnsembleScores(scores []db.LLMScore) (int, bool) {
	nonEnsembleCount := 0
	allZeroConfidence := true

	for _, score := range scores {
		if isEnsembleModel(score.Model) {
			continue
		}

		nonEnsembleCount++
		if hasNonZeroConfidenceInMetadata(score) {
			allZeroConfidence = false
			break // Found one with non-zero confidence, no need to check further
		}
	}

	return nonEnsembleCount, allZeroConfidence
}

// checkForAllZeroResponses detects if all non-ensemble LLM responses have zero confidence.
func checkForAllZeroResponses(scores []db.LLMScore) (bool, error) {
	nonEnsembleCount, allZeroConfidence := countNonEnsembleScores(scores)

	if nonEnsembleCount == 0 {
		return false, nil // No non-ensemble scores to check
	}

	if allZeroConfidence {
		log.Printf("Critical warning: All %d non-ensemble LLM models returned zero confidence", nonEnsembleCount)
		return true, ErrAllScoresZeroConfidence
	}

	return false, nil
}

// extractConfidenceFromMetadata extracts confidence value from metadata string
func extractConfidenceFromMetadata(metadata string) float64 {
	if metadata == "" {
		return 0.0
	}

	var metadataMap map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &metadataMap); err == nil {
		if conf, ok := metadataMap["confidence"].(float64); ok {
			return conf
		}
	}
	return 0.0
}

// sortModelsByConfidenceAndTime sorts models by confidence (highest first) and then by created_at
func sortModelsByConfidenceAndTime(models []db.LLMScore) {
	sort.Slice(models, func(i, j int) bool {
		iConf := extractConfidenceFromMetadata(models[i].Metadata)
		jConf := extractConfidenceFromMetadata(models[j].Metadata)

		// First sort by confidence (highest first)
		if iConf != jConf {
			return iConf > jConf
		}
		// If confidence is equal, sort by created_at (newest first)
		return models[i].CreatedAt.After(models[j].CreatedAt)
	})
}

// processScoreForPerspective processes a single score and updates the perspective state
func processScoreForPerspective(score db.LLMScore, perspective string, cfg *CompositeScoreConfig, scoreMap map[string]float64, validCount *int, validModels map[string]bool) bool {
	if isInvalid(score.Score, cfg) {
		if cfg.HandleInvalid == "ignore" {
			return false // Skip invalid scores when ignoring
		}
		// Default to default value
		scoreMap[perspective] = cfg.DefaultMissing
		(*validCount)++
		validModels[perspective] = true
		return true
	}

	// Valid score
	scoreMap[perspective] = score.Score
	(*validCount)++
	validModels[perspective] = true
	return true
}

// logPerspectiveCandidates logs the candidate models for a perspective
func logPerspectiveCandidates(perspective string, models []db.LLMScore) {
	log.Printf("Candidates for %s: ", perspective)
	for _, m := range models {
		log.Printf("  Model: %s, Score: %.2f, Metadata: %s", m.Model, m.Score, m.Metadata)
	}
}

// processScoresByPerspective processes scores and updates maps based on the provided configuration
func processScoresByPerspective(perspectiveModels map[string][]db.LLMScore, cfg *CompositeScoreConfig, scoreMap map[string]float64, validCount *int, validModels map[string]bool) {
	for perspective, models := range perspectiveModels {
		if len(models) == 0 {
			log.Printf("No models found for perspective %s", perspective)
			continue
		}

		logPerspectiveCandidates(perspective, models)
		sortModelsByConfidenceAndTime(models)

		// Select the first (highest confidence) valid score for this perspective
		foundValidScore := false
		for _, s := range models {
			if processScoreForPerspective(s, perspective, cfg, scoreMap, validCount, validModels) {
				foundValidScore = true
				break
			}
		}

		// If we ignored all invalid scores and found no valid ones, don't mark this perspective as valid
		if !foundValidScore && cfg.HandleInvalid == "ignore" {
			// The perspective will keep its default value but won't be counted as valid
		}
	}
}

// mapEnsembleScore handles ensemble model mapping to center perspective
func mapEnsembleScore(score db.LLMScore, perspectiveModels map[string][]db.LLMScore) {
	log.Printf("Mapping ensemble model (score %.2f) to perspective '%s'", score.Score, LabelCenter)
	perspectiveModels[LabelCenter] = append(perspectiveModels[LabelCenter], score)
}

// tryLegacyModelMapping attempts to map model using legacy names
func tryLegacyModelMapping(modelName string) string {
	modelLower := strings.ToLower(modelName)
	switch modelLower {
	case LabelLeft:
		return LabelLeft
	case LabelCenter, LabelNeutral:
		return LabelCenter
	case LabelRight:
		return LabelRight
	default:
		return ""
	}
}

// isValidPerspective checks if perspective is one of the expected values
func isValidPerspective(perspective string) bool {
	return perspective == LabelLeft || perspective == LabelCenter || perspective == LabelRight
}

// mapScoreToPerspective determines the perspective for a score and adds it to the map
func mapScoreToPerspective(score db.LLMScore, cfg *CompositeScoreConfig, perspectiveModels map[string][]db.LLMScore) {
	// Handle ensemble scores specially
	if isEnsembleModel(score.Model) {
		mapEnsembleScore(score, perspectiveModels)
		return
	}

	// First try to map the model to its perspective
	perspective := MapModelToPerspective(score.Model, cfg)

	// If mapping failed, try the legacy way
	if perspective == "" {
		perspective = tryLegacyModelMapping(score.Model)
		if perspective == "" {
			log.Printf("Skipping unknown model: %s", score.Model)
			return
		}
	}

	// Ensure perspective is valid
	if !isValidPerspective(perspective) {
		log.Printf("Skipping model with invalid perspective: %s -> %s", score.Model, perspective)
		return
	}

	// Add to perspective models map
	log.Printf("Mapping model '%s' (score %.2f) to perspective '%s'", score.Model, score.Score, perspective)
	perspectiveModels[perspective] = append(perspectiveModels[perspective], score)
}

// logPerspectiveResults logs the final perspective mapping results
func logPerspectiveResults(perspectiveModels map[string][]db.LLMScore) {
	for perspective, models := range perspectiveModels {
		log.Printf("Perspective %s has %d models", perspective, len(models))
	}
}

// mapModelsToPerspectives groups scores by perspective based on the configuration
func mapModelsToPerspectives(scores []db.LLMScore, cfg *CompositeScoreConfig) map[string][]db.LLMScore {
	perspectiveModels := make(map[string][]db.LLMScore)

	for _, score := range scores {
		mapScoreToPerspective(score, cfg, perspectiveModels)
	}

	logPerspectiveResults(perspectiveModels)
	return perspectiveModels
}

// checkAllScoresZero checks if all scores in the map are zero
func checkAllScoresZero(scoreMap map[string]float64, actualValidCount int) bool {
	if actualValidCount == 0 {
		return false
	}

	for _, score := range scoreMap {
		if score != 0 {
			return false
		}
	}
	return true
}

// findSingleValidScore returns the single valid score if only one exists
func findSingleValidScore(scoreMap map[string]float64, validModels map[string]bool, cfg *CompositeScoreConfig, actualValidCount int) (float64, bool) {
	if actualValidCount != 1 {
		return 0, false
	}

	for perspective, score := range scoreMap {
		if _, isValid := validModels[perspective]; isValid {
			// Apply bounds before returning
			if score < cfg.MinScore {
				return cfg.MinScore, true
			}
			if score > cfg.MaxScore {
				return cfg.MaxScore, true
			}
			return score, true
		}
	}
	return 0, false
}

// computeScoreByFormula calculates score based on the specified formula
func computeScoreByFormula(cfg *CompositeScoreConfig, sum float64, weightedSum float64, weightTotal float64, actualValidCount int) float64 {
	var composite float64

	switch cfg.Formula {
	case "weighted":
		if weightTotal > 0 {
			composite = weightedSum / weightTotal
		} else {
			// Fallback if weights are zero or missing
			composite = sum / float64(actualValidCount)
		}
	case "average":
		composite = sum / float64(actualValidCount)
	default:
		composite = sum / float64(actualValidCount)
	}

	return composite
}

// applyScoreBounds ensures the score is within configured bounds
func applyScoreBounds(score float64, cfg *CompositeScoreConfig) float64 {
	if score < cfg.MinScore {
		return cfg.MinScore
	}
	if score > cfg.MaxScore {
		return cfg.MaxScore
	}
	return score
}

// calculateCompositeScore calculates the final composite score based on the configuration and intermediate values
func calculateCompositeScore(cfg *CompositeScoreConfig, scoreMap map[string]float64, sum float64, weightedSum float64, weightTotal float64, actualValidCount int, validModels map[string]bool) (float64, error) {
	if actualValidCount == 0 {
		return 0.0, ErrAllPerspectivesInvalid
	}

	// Check if all scores are zero
	if checkAllScoresZero(scoreMap, actualValidCount) {
		return 0.0, ErrAllPerspectivesInvalid
	}

	// If only one valid score, return it directly (avoids averaging with defaults)
	if singleScore, hasSingle := findSingleValidScore(scoreMap, validModels, cfg, actualValidCount); hasSingle {
		return singleScore, nil
	}

	// Calculate composite score based on formula
	composite := computeScoreByFormula(cfg, sum, weightedSum, weightTotal, actualValidCount)

	// Ensure the score is within bounds
	composite = applyScoreBounds(composite, cfg)

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

	// Only apply confidence limits if we don't have all perspectives AND limits are properly configured
	if perspectiveCount < 3 && cfg.MaxConfidence > cfg.MinConfidence {
		if confidence < cfg.MinConfidence {
			confidence = cfg.MinConfidence
		}
		if confidence > cfg.MaxConfidence {
			confidence = cfg.MaxConfidence
		}
	}

	return confidence
}

// validateInputs performs initial validation of scores and configuration
func validateInputs(scores []db.LLMScore, cfg *CompositeScoreConfig) error {
	if len(scores) == 0 {
		return fmt.Errorf("no scores provided: %w", ErrAllPerspectivesInvalid)
	}

	// Check if we have all zero responses
	if allZeros, err := checkForAllZeroResponses(scores); allZeros {
		log.Printf("[ERROR][CONFIDENCE] All scores are zero, returning error")
		return fmt.Errorf("%w: %w", ErrAllPerspectivesInvalid, err)
	}

	if cfg == nil {
		log.Printf("[ERROR][CONFIDENCE] Config is nil")
		return fmt.Errorf("composite score config is nil: %w", ErrAllPerspectivesInvalid)
	}

	return nil
}

// initializeScoreMap creates and initializes the score map with default values
func initializeScoreMap(cfg *CompositeScoreConfig) map[string]float64 {
	return map[string]float64{
		LabelLeft:   cfg.DefaultMissing,
		LabelCenter: cfg.DefaultMissing,
		LabelRight:  cfg.DefaultMissing,
	}
}

// calculateSums computes sums based only on valid models
func calculateSums(scoreMap map[string]float64, validModels map[string]bool, cfg *CompositeScoreConfig) (float64, float64, float64, int) {
	sum := 0.0
	weightedSum := 0.0
	weightTotal := 0.0
	actualValidCount := 0

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
			actualValidCount++
		}
	}

	return sum, weightedSum, weightTotal, actualValidCount
}

// logDebugInfo logs debug information about the calculation state
func logDebugInfo(validCount int, validModels map[string]bool, scoreMap map[string]float64, sum float64, weightedSum float64, weightTotal float64, actualValidCount int) {
	log.Printf("[DEBUG] Pre-Sum: validCount=%d, len(validModels)=%d", validCount, len(validModels))
	log.Printf("[DEBUG] Pre-Sum: Score map: %v", scoreMap)
	log.Printf("[DEBUG] Pre-Sum: Valid models map: %v", validModels)
	log.Printf("[DEBUG] Pre-Calc: sum=%.4f, weightedSum=%.4f, weightTotal=%.4f, actualValidCount=%d",
		sum, weightedSum, weightTotal, actualValidCount)
}

// ComputeCompositeScoreWithConfidenceFixed calculates the composite score and confidence based on provided scores and configuration
func ComputeCompositeScoreWithConfidenceFixed(scores []db.LLMScore, cfg *CompositeScoreConfig) (float64, float64, error) {
	// Validate inputs
	if err := validateInputs(scores, cfg); err != nil {
		return 0, 0, err
	}

	// Initialize score map
	scoreMap := initializeScoreMap(cfg)
	validCount := 0
	validModels := make(map[string]bool)

	// Process scores by perspective
	perspectiveModels := mapModelsToPerspectives(scores, cfg)
	processScoresByPerspective(perspectiveModels, cfg, scoreMap, &validCount, validModels)

	// Check if no valid scores were found after processing
	if validCount == 0 {
		log.Printf("[WARN][CONFIDENCE] No valid model scores found after processing. Returning default score.")
		return cfg.DefaultMissing, 0.0, ErrAllPerspectivesInvalid
	}

	// Calculate sums based only on valid models
	sum, weightedSum, weightTotal, actualValidCount := calculateSums(scoreMap, validModels, cfg)

	// Log debug information
	logDebugInfo(validCount, validModels, scoreMap, sum, weightedSum, weightTotal, actualValidCount)

	// Handle division by zero
	if actualValidCount == 0 {
		log.Printf("[ERROR][CONFIDENCE] Logic error: validCount > 0 but actualValidCount is 0.")
		return cfg.DefaultMissing, 0.0, fmt.Errorf("internal calculation error: no valid scores counted")
	}

	// Final calculation
	compositeScore, calcErr := calculateCompositeScore(cfg, scoreMap, sum, weightedSum, weightTotal, actualValidCount, validModels)
	if calcErr != nil {
		log.Printf("[ERROR] Error in calculateCompositeScore: %v. actualValidCount=%d", calcErr, actualValidCount)
		return 0.0, 0.0, calcErr
	}

	// Calculate confidence
	confidence := calculateConfidence(cfg, validModels, scoreMap)

	return compositeScore, confidence, nil
}
