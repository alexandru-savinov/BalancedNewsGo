package db

import (
	"testing"
	"time"
)

// TestBiasDetectionLLMOnly verifies that bias detection works solely through LLM scoring
// after removing keyword heuristics
func TestBiasDetectionLLMOnly(t *testing.T) {
	testCases := []struct {
		name           string
		compositeScore *float64
		expectedBias   string
	}{
		{"Left-leaning article", floatPtr(-0.5), "left"},
		{"Right-leaning article", floatPtr(0.5), "right"},
		{"Center article", floatPtr(0.05), "center"},
		{"Strong left article", floatPtr(-0.8), "left"},
		{"Strong right article", floatPtr(0.8), "right"},
		{"Article with no score", nil, "unknown"},
		{"Boundary left", floatPtr(-0.2), "left"},
		{"Boundary right", floatPtr(0.2), "right"},
		{"Exactly center", floatPtr(0.0), "center"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create an article with the test data
			article := Article{
				ID:             1,
				Source:         "test",
				PubDate:        time.Now(),
				URL:            "http://example.com/test",
				Title:          tc.name,
				Content:        "Test content",
				CompositeScore: tc.compositeScore,
				CreatedAt:      time.Now(),
			}

			// Calculate bias using the LLM-based method
			article.CalculateBias()

			// Verify the result
			if article.Bias != tc.expectedBias {
				t.Errorf("Expected bias %s, got %s for score %v",
					tc.expectedBias, article.Bias, tc.compositeScore)
			}
		})
	}
}

func floatPtr(f float64) *float64 {
	return &f
}
