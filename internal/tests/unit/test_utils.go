package unit

import (
	"encoding/json"
	"fmt"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// TestScore represents a score for testing with easy initialization
type TestScore struct {
	Model      string
	Score      float64
	Confidence float64
}

// ToLLMScore converts a TestScore to db.LLMScore
func (ts TestScore) ToLLMScore() db.LLMScore {
	metadata := map[string]interface{}{
		"confidence": ts.Confidence,
	}
	metadataBytes, _ := json.Marshal(metadata)

	return db.LLMScore{
		Model:    ts.Model,
		Score:    ts.Score,
		Metadata: string(metadataBytes),
	}
}

// GenerateScoreSet creates a set of test scores for a given base score
func GenerateScoreSet(baseScore float64, conf float64) []db.LLMScore {
	return []db.LLMScore{
		TestScore{Model: "left", Score: baseScore, Confidence: conf}.ToLLMScore(),
		TestScore{Model: "center", Score: baseScore, Confidence: conf}.ToLLMScore(),
		TestScore{Model: "right", Score: baseScore, Confidence: conf}.ToLLMScore(),
	}
}

// GenerateSteppedScores generates scores at regular intervals for testing
func GenerateSteppedScores(start, end, step float64, conf float64) [][]db.LLMScore {
	var result [][]db.LLMScore

	for score := start; score <= end; score += step {
		scores := GenerateScoreSet(score, conf)
		result = append(result, scores)
	}

	return result
}

// FormatTestName generates a descriptive test name
func FormatTestName(prefix string, score float64) string {
	return fmt.Sprintf("%s_%.3f", prefix, score)
}
