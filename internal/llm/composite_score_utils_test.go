package llm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := getConfigDir()
	require.NoError(t, err, "getConfigDir should not return an error")

	// Verify the directory exists
	_, err = os.Stat(dir)
	assert.NoError(t, err, "Directory returned by getConfigDir should exist")

	// Verify configs directory exists underneath it
	configsDir := filepath.Join(dir, "configs")
	_, err = os.Stat(configsDir)
	assert.NoError(t, err, "configs directory should exist under the returned directory")

	// Verify composite_score_config.json exists
	configFile := filepath.Join(configsDir, "composite_score_config.json")
	_, err = os.Stat(configFile)
	assert.NoError(t, err, "composite_score_config.json should exist in the configs directory")
}

func TestMinNonNil(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]float64
		default_ float64
		expected float64
	}{
		{
			name:     "empty map returns default",
			input:    map[string]float64{},
			default_: 5.0,
			expected: 5.0,
		},
		{
			name:     "single value returns that value",
			input:    map[string]float64{"a": 3.0},
			default_: 5.0,
			expected: 3.0,
		},
		{
			name:     "multiple values returns minimum",
			input:    map[string]float64{"a": 3.0, "b": 1.0, "c": 7.0},
			default_: 5.0,
			expected: 1.0,
		},
		{
			name:     "negative values handled correctly",
			input:    map[string]float64{"a": -3.0, "b": 1.0, "c": -7.0},
			default_: 0.0,
			expected: -7.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minNonNil(tt.input, tt.default_)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaxNonNil(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]float64
		default_ float64
		expected float64
	}{
		{
			name:     "empty map returns default",
			input:    map[string]float64{},
			default_: 5.0,
			expected: 5.0,
		},
		{
			name:     "single value returns that value",
			input:    map[string]float64{"a": 3.0},
			default_: 5.0,
			expected: 3.0,
		},
		{
			name:     "multiple values returns maximum",
			input:    map[string]float64{"a": 3.0, "b": 1.0, "c": 7.0},
			default_: 5.0,
			expected: 7.0,
		},
		{
			name:     "negative values handled correctly",
			input:    map[string]float64{"a": -3.0, "b": 1.0, "c": -7.0},
			default_: 0.0,
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxNonNil(tt.input, tt.default_)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScoreSpread(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]float64
		expected float64
	}{
		{
			name:     "empty map returns 0",
			input:    map[string]float64{},
			expected: 0.0,
		},
		{
			name:     "single value returns 0",
			input:    map[string]float64{"a": 3.0},
			expected: 0.0,
		},
		{
			name:     "multiple values returns max-min",
			input:    map[string]float64{"a": 3.0, "b": 1.0, "c": 7.0},
			expected: 6.0, // 7.0 - 1.0
		},
		{
			name:     "negative values handled correctly",
			input:    map[string]float64{"a": -3.0, "b": 1.0, "c": -7.0},
			expected: 8.0, // 1.0 - (-7.0)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scoreSpread(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
