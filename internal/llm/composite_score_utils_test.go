package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinNonNil(t *testing.T) {
	testCases := []struct {
		name     string
		values   map[string]*float64
		def      float64
		expected float64
	}{
		{
			name: "All values provided",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": floatPtr(0.1),
				"c": floatPtr(0.8),
			},
			def:      0.0,
			expected: 0.1,
		},
		{
			name: "Some nil values",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": nil,
				"c": floatPtr(0.1),
			},
			def:      0.0,
			expected: 0.1,
		},
		{
			name: "Single value",
			values: map[string]*float64{
				"a": floatPtr(-0.3),
			},
			def:      0.0,
			expected: -0.3,
		},
		{
			name:     "Empty map",
			values:   map[string]*float64{},
			def:      0.5,
			expected: 0.5,
		},
		{
			name: "All nil",
			values: map[string]*float64{
				"a": nil,
				"b": nil,
			},
			def:      0.5,
			expected: 0.5,
		},
		{
			name: "Negative values",
			values: map[string]*float64{
				"a": floatPtr(-0.3),
				"b": floatPtr(-0.1),
				"c": floatPtr(-0.8),
			},
			def:      0.0,
			expected: -0.8,
		},
		{
			name: "Mixed positive and negative",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": floatPtr(-0.2),
				"c": floatPtr(0.1),
			},
			def:      0.0,
			expected: -0.2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := minNonNil(tc.values, tc.def)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMaxNonNil(t *testing.T) {
	testCases := []struct {
		name     string
		values   map[string]*float64
		def      float64
		expected float64
	}{
		{
			name: "All values provided",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": floatPtr(0.1),
				"c": floatPtr(0.8),
			},
			def:      0.0,
			expected: 0.8,
		},
		{
			name: "Some nil values",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": nil,
				"c": floatPtr(0.1),
			},
			def:      0.0,
			expected: 0.5,
		},
		{
			name: "Single value",
			values: map[string]*float64{
				"a": floatPtr(-0.3),
			},
			def:      0.0,
			expected: -0.3,
		},
		{
			name:     "Empty map",
			values:   map[string]*float64{},
			def:      0.5,
			expected: 0.5,
		},
		{
			name: "All nil",
			values: map[string]*float64{
				"a": nil,
				"b": nil,
			},
			def:      0.5,
			expected: 0.5,
		},
		{
			name: "Negative values",
			values: map[string]*float64{
				"a": floatPtr(-0.3),
				"b": floatPtr(-0.1),
				"c": floatPtr(-0.8),
			},
			def:      0.0,
			expected: -0.1,
		},
		{
			name: "Mixed positive and negative",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": floatPtr(-0.2),
				"c": floatPtr(0.1),
			},
			def:      0.0,
			expected: 0.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := maxNonNil(tc.values, tc.def)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestScoreSpread(t *testing.T) {
	testCases := []struct {
		name     string
		values   map[string]*float64
		expected float64
	}{
		{
			name: "Standard spread",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": floatPtr(0.1),
				"c": floatPtr(0.8),
			},
			expected: 0.7, // 0.8 - 0.1 = 0.7
		},
		{
			name: "Some nil values",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": nil,
				"c": floatPtr(0.1),
			},
			expected: 0.4, // 0.5 - 0.1 = 0.4
		},
		{
			name: "Single value",
			values: map[string]*float64{
				"a": floatPtr(-0.3),
			},
			expected: 0.0, // No spread with a single value
		},
		{
			name:     "Empty map",
			values:   map[string]*float64{},
			expected: 0.0, // No spread with no values
		},
		{
			name: "All nil",
			values: map[string]*float64{
				"a": nil,
				"b": nil,
			},
			expected: 0.0, // No spread with no values
		},
		{
			name: "Negative values",
			values: map[string]*float64{
				"a": floatPtr(-0.3),
				"b": floatPtr(-0.1),
				"c": floatPtr(-0.8),
			},
			expected: 0.7, // -0.1 - (-0.8) = 0.7
		},
		{
			name: "Mixed positive and negative",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": floatPtr(-0.2),
				"c": floatPtr(0.1),
			},
			expected: 0.7, // 0.5 - (-0.2) = 0.7
		},
		{
			name: "Same values",
			values: map[string]*float64{
				"a": floatPtr(0.5),
				"b": floatPtr(0.5),
				"c": floatPtr(0.5),
			},
			expected: 0.0, // No spread with identical values
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := scoreSpread(tc.values)
			// Use delta comparison for floating point values
			assert.InDelta(t, tc.expected, result, 0.0001)
		})
	}
}

// Helper function to get a pointer to a float64
func floatPtr(v float64) *float64 {
	return &v
}
