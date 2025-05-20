package balancer

import (
	"math"
	"testing"
)

func TestComputeWithMixedModelNames(t *testing.T) {
	tests := []struct {
		name                   string
		models                 []string
		expectedScores         map[string]float64
		expectedCompositeScore float64
	}{
		{
			name: "mixed model names",
			models: []string{
				"model1",
				"model2",
				"model3",
				"model4",
				"model5",
				"model6",
				"model7",
				"model8",
				"model9",
				"model10",
			},
			expectedScores: map[string]float64{
				"model1":  0.1,
				"model2":  0.2,
				"model3":  0.3,
				"model4":  0.4,
				"model5":  0.5,
				"model6":  0.6,
				"model7":  0.7,
				"model8":  0.8,
				"model9":  0.9,
				"model10": 1.0,
			},
			expectedCompositeScore: 0.55,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scores := ComputeWithMixedModelNames(test.models)
			for model, expectedScore := range test.expectedScores {
				actualScore := scores[model]
				if math.Abs(actualScore-expectedScore) > 1e-9 {
					t.Errorf("Expected score for %s to be %f, but got %f", model, expectedScore, actualScore)
				}
			}
			if scores["composite"] != test.expectedCompositeScore {
				t.Errorf("Expected composite score to be %f, but got %f", test.expectedCompositeScore, scores["composite"])
			}
		})
	}
}
