package rss

import (
	"testing"
)

// Test partisan cue detection
func TestDetectPartisanCues(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"The radical left is pushing new policies", true},
		{"A patriotic event was held downtown", true},
		{"Neutral news update on economy", false},
		{"Fake news spreads quickly", true},
		{"Discussion on social justice reforms", true},
		{"Completely unrelated topic", false},
	}

	for _, tt := range tests {
		cues := detectPartisanCues(tt.text)
		if (len(cues) > 0) != tt.expected {
			t.Errorf("detectPartisanCues(%q) = %v, want cues? %v", tt.text, cues, tt.expected)
		}
	}
}
