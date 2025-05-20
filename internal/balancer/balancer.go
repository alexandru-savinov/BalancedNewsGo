package balancer

import (
	"fmt"
)

// ComputeWithMixedModelNames assigns scores to models and calculates a composite score
func ComputeWithMixedModelNames(models []string) map[string]float64 {
	scores := make(map[string]float64)

	// Assign scores from 0.1 to 1.0 to models 1-10
	for i, model := range models {
		score := float64(i+1) * 0.1 // This will give 0.1, 0.2, ..., 1.0
		scores[model] = score
	}

	// Calculate composite score (simple average)
	var sum float64
	for _, model := range models {
		sum += scores[model]
	}
	// Debug print before adding composite
	fmt.Printf("DEBUG: scores before composite: %+v\n", scores)
	fmt.Printf("DEBUG: sum before composite: %f\n", sum)

	scores["composite"] = sum / float64(len(models))
	// Debug print after adding composite
	fmt.Printf("DEBUG: scores after composite: %+v\n", scores)
	fmt.Printf("DEBUG: composite score: %f\n", scores["composite"])

	return scores
}
