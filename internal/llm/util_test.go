package llm

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMin tests the min utility function
func TestMin(t *testing.T) {
	testCases := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "First value smaller",
			a:        5,
			b:        10,
			expected: 5,
		},
		{
			name:     "Second value smaller",
			a:        10,
			b:        5,
			expected: 5,
		},
		{
			name:     "Equal values",
			a:        7,
			b:        7,
			expected: 7,
		},
		{
			name:     "Negative values",
			a:        -10,
			b:        -5,
			expected: -10,
		},
		{
			name:     "Zero and positive",
			a:        0,
			b:        5,
			expected: 0,
		},
		{
			name:     "Negative and zero",
			a:        -5,
			b:        0,
			expected: -5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := min(tc.a, tc.b)
			assert.Equal(t, tc.expected, result, "min function returned incorrect value")
		})
	}
}

// TestHashContent tests the hashContent utility function
func TestHashContent(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:    "Empty string",
			content: "",
			// SHA-256 of empty string
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "Simple string",
			content: "hello world",
			// SHA-256 of "hello world"
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:    "Special characters",
			content: "!@#$%^&*()",
			// Generate the expected hash dynamically instead of hardcoding it
			expected: func() string {
				hash := sha256.Sum256([]byte("!@#$%^&*()"))
				return fmt.Sprintf("%x", hash)
			}(),
		},
		{
			name: "Long text",
			content: "This is a longer text that will be hashed to test the hashContent function " +
				"with more than just a few characters to ensure it works correctly with various input sizes.",
			// SHA-256 of the long text
			expected: func() string {
				hash := sha256.Sum256([]byte("This is a longer text that will be hashed to test the hashContent function " +
					"with more than just a few characters to ensure it works correctly with various input sizes."))
				return fmt.Sprintf("%x", hash)
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hashContent(tc.content)
			assert.Equal(t, tc.expected, result, "hashContent function returned incorrect hash")
		})
	}
}
