package rss

import (
	"testing"

	"github.com/mmcdole/gofeed"
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

func TestIsValidItem(t *testing.T) {
	tests := []struct {
		title    string
		content  string
		desc     string
		expected bool
	}{
		{"Valid Title", "Valid Content", "", true},
		{"Valid Title", "", "Valid Description", true},
		{"Valid Title", "", "", false},       // No content or description
		{"", "Valid Content", "", false},     // No title
		{"Valid Title", "   ", "   ", false}, // Only whitespace content/desc
	}

	for _, tt := range tests {
		item := &gofeed.Item{
			Title:       tt.title,
			Content:     tt.content,
			Description: tt.desc,
			Link:        "http://example.com/article",
		}
		if got := isValidItem(item); got != tt.expected {
			t.Errorf("isValidItem(Title=%q, Content=%q, Description=%q) = %v, want %v", tt.title, tt.content, tt.desc, got, tt.expected)
		}
	}
}
