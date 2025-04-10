package llm

import (
	"testing"
)

func TestParseBiasResult(t *testing.T) {
	// Set allowed categories for validation
	BiasCfg.Categories = []string{"Politics", "Economy", "Health", "Tech"}

	tests := []struct {
		name     string
		input    string
		expected BiasResult
	}{
		{
			name:     "Clean JSON with delimiters",
			input:    "```{\"category\":\"Politics\",\"confidence\":0.5}```",
			expected: BiasResult{Category: "Politics", Confidence: 0.5},
		},
		{
			name:     "Noisy output with JSON inside delimiters",
			input:    "Some explanation text.\n```{\"category\":\"Economy\",\"confidence\":0.8}```\nMore trailing text.",
			expected: BiasResult{Category: "Economy", Confidence: 0.8},
		},
		{
			name:     "No delimiters",
			input:    "{\"category\":\"Health\",\"confidence\":0.3}",
			expected: BiasResult{},
		},
		{
			name:     "Malformed JSON inside delimiters",
			input:    "```{\"category\":\"Tech\", \"confidence\":}```",
			expected: BiasResult{Category: "unknown", Confidence: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBiasResult(tt.input)
			if result.Category != tt.expected.Category || result.Confidence != tt.expected.Confidence {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}
