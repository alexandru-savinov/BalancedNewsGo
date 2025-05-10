package llm

import (
	"fmt"
	"math"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestDefaultScoreCalculator_CalculateScore(t *testing.T) {
	cfg := &CompositeScoreConfig{
		MinScore:         -1.0,
		MaxScore:         1.0,
		DefaultMissing:   0.0,
		HandleInvalid:    "ignore",
		Formula:          "average",
		ConfidenceMethod: "count_valid",
		MinConfidence:    0.0,
		MaxConfidence:    1.0,
	}
	// Initialize calculator without config
	calc := &DefaultScoreCalculator{}

	tests := []struct {
		name          string
		scores        []db.LLMScore
		expectedScore float64
		expectedConf  float64
		expectError   bool
		errorType     error
	}{
		{
			name: "all perspectives present",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8, Metadata: `{"confidence": 0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`},
				{Model: "right", Score: 0.6, Metadata: `{"confidence": 0.7}`},
			},
			expectedScore: -0.067, // (-0.8 + 0.0 + 0.6) / 3
			expectedConf:  0.8,    // (0.9 + 0.8 + 0.7) / 3
			expectError:   false,
		},
		{
			name: "missing center perspective",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8, Metadata: `{"confidence": 0.9}`},
				{Model: "right", Score: 0.6, Metadata: `{"confidence": 0.7}`},
			},
			expectedScore: -0.100, // (-0.8 + 0.6) / 2, center perspective missing
			expectedConf:  0.800,  // (0.9 + 0.7) / 2, missing confidence not counted
			expectError:   false,
		},
		{
			name: "score out of range",
			scores: []db.LLMScore{
				{Model: "left", Score: -2.0, Metadata: `{"confidence": 0.9}`}, // Will be ignored
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`},
				{Model: "right", Score: 1.5, Metadata: `{"confidence": 0.7}`}, // Will be ignored
			},
			expectedScore: 0.0, // Only center score is valid
			expectedConf:  0.8, // Only center confidence is valid
			expectError:   false,
		},
		{
			name:          "no scores",
			scores:        []db.LLMScore{},
			expectedScore: 0.0,
			expectedConf:  0.0,
			expectError:   true,
			errorType:     ErrAllPerspectivesInvalid,
		},
		{
			name: "all invalid scores",
			scores: []db.LLMScore{
				{Model: "left", Score: math.NaN(), Metadata: `{"confidence": 0.9}`},
				{Model: "center", Score: math.Inf(1), Metadata: `{"confidence": 0.8}`},
				{Model: "right", Score: -2.0, Metadata: `{"confidence": 0.7}`},
			},
			expectedScore: 0.0,
			expectedConf:  0.0,
			expectError:   true,
			errorType:     ErrAllPerspectivesInvalid,
		},
		{
			name: "all zero confidence",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.0, Metadata: `{"confidence": 0.0}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.0}`},
				{Model: "right", Score: 0.0, Metadata: `{"confidence": 0.0}`},
			},
			expectedScore: 0.0,
			expectedConf:  0.0,
			expectError:   true,
			errorType:     ErrAllPerspectivesInvalid,
		},
		{
			name: "invalid model names",
			scores: []db.LLMScore{
				{Model: "unknown", Score: 0.5, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 0.0,
			expectedConf:  0.0,
			expectError:   true,
			errorType:     ErrAllPerspectivesInvalid,
		},
		{
			name: "partial invalid metadata",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.5, Metadata: `invalid json`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`},
				{Model: "right", Score: 0.5, Metadata: `{"confidence": 0.7}`},
			},
			expectedScore: 0.25, // (0.0 + 0.5) / 2 (left is ignored due to invalid metadata)
			expectedConf:  0.75, // (0.8 + 0.7) / 2
			expectError:   false,
		},
		{
			name: "nil config",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.5, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 0.0,
			expectedConf:  0.0,
			expectError:   true,
			errorType:     fmt.Errorf("DefaultScoreCalculator: Config must not be nil"),
		},
		// Additional test cases
		{
			name: "all perspectives with equal scores",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.5, Metadata: `{"confidence": 0.9}`},
				{Model: "center", Score: 0.5, Metadata: `{"confidence": 0.9}`},
				{Model: "right", Score: 0.5, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 0.5,
			expectedConf:  0.9,
			expectError:   false,
		},
		{
			name: "extreme opposing scores",
			scores: []db.LLMScore{
				{Model: "left", Score: -1.0, Metadata: `{"confidence": 1.0}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 1.0}`},
				{Model: "right", Score: 1.0, Metadata: `{"confidence": 1.0}`},
			},
			expectedScore: 0.0,
			expectedConf:  1.0,
			expectError:   false,
		},
		{
			name: "all zeroes",
			scores: []db.LLMScore{
				{Model: "left", Score: 0.0, Metadata: `{"confidence": 0.5}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.5}`},
				{Model: "right", Score: 0.0, Metadata: `{"confidence": 0.5}`},
			},
			expectedScore: 0.0,
			expectedConf:  0.5,
			expectError:   false,
		},
		{
			name: "missing confidence fields",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.5, Metadata: `{}`},
				{Model: "center", Score: 0.0, Metadata: `{}`},
				{Model: "right", Score: 0.5, Metadata: `{}`},
			},
			expectedScore: 0.0,
			expectedConf:  0.0,
			expectError:   true,
			errorType:     ErrAllPerspectivesInvalid,
		},
		{
			name: "malformed metadata mixed with valid",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.5, Metadata: ``}, // Empty metadata
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`},
				{Model: "right", Score: 0.5, Metadata: `{]`}, // Invalid JSON
			},
			expectedScore: 0.0,
			expectedConf:  0.8, // Only center confidence is valid
			expectError:   false,
		},
		{
			name: "case insensitive model names",
			scores: []db.LLMScore{
				{Model: "LEFT", Score: -0.5, Metadata: `{"confidence": 0.7}`},
				{Model: "Center", Score: 0.0, Metadata: `{"confidence": 0.8}`},
				{Model: "RiGhT", Score: 0.5, Metadata: `{"confidence": 0.9}`},
			},
			expectedScore: 0.0,
			expectedConf:  0.8, // (0.7 + 0.8 + 0.9) / 3
			expectError:   false,
		},
		{
			name: "duplicate perspectives (should use last)",
			scores: []db.LLMScore{
				{Model: "left", Score: -0.8, Metadata: `{"confidence": 0.7}`},
				{Model: "left", Score: -0.5, Metadata: `{"confidence": 0.9}`},
				{Model: "center", Score: 0.0, Metadata: `{"confidence": 0.8}`},
				{Model: "right", Score: 0.5, Metadata: `{"confidence": 0.7}`},
			},
			expectedScore: -0.2,  // (-0.5 + 0.0 + 0.5 - 0.8) / 4
			expectedConf:  0.775, // (0.7 + 0.9 + 0.8 + 0.7) / 4
			expectError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var useCfg *CompositeScoreConfig
			if tc.name == "nil config" {
				useCfg = nil
			} else {
				useCfg = cfg
			}
			score, conf, err := calc.CalculateScore(tc.scores, useCfg)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorType != nil {
					assert.ErrorContains(t, err, tc.errorType.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tc.expectedScore, score, 0.001)
				assert.InDelta(t, tc.expectedConf, conf, 0.001)
			}
		})
	}
}

func TestDefaultScoreCalculator_GetPerspective(t *testing.T) {
	cfg := &CompositeScoreConfig{
		MinScore:       -1.0,
		MaxScore:       1.0,
		DefaultMissing: 0.0,
	}
	// Initialize calculator without config
	calc := &DefaultScoreCalculator{}

	tests := []struct {
		name          string
		model         string
		expectedValue string
	}{
		{"left lowercase", "left", "left"},
		{"right lowercase", "right", "right"},
		{"center lowercase", "center", "center"},
		{"left uppercase", "LEFT", "left"},
		{"right uppercase", "RIGHT", "right"},
		{"center mixed case", "CeNtEr", "center"},
		{"unknown model", "unknown", ""},
		{"empty string", "", ""},
		{"label constant left", LabelLeft, "left"},
		{"label constant right", LabelRight, "right"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass cfg to getPerspective
			result := calc.getPerspective(tt.model, cfg)
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

func TestDefaultScoreCalculator_ExtractConfidence(t *testing.T) {
	// Initialize calculator without config
	calc := &DefaultScoreCalculator{}

	tests := []struct {
		name          string
		metadata      string
		expectedValue float64
	}{
		{"valid confidence", `{"confidence": 0.8}`, 0.8},
		{"no confidence field", `{"other": 123}`, 0.0},
		{"invalid json", `{invalid}`, 0.0},
		{"empty string", "", 0.0},
		{"null confidence", `{"confidence": null}`, 0.0},
		{"string confidence", `{"confidence": "0.9"}`, 0.0},
		{"integer confidence", `{"confidence": 1}`, 1.0},
		{"nested confidence", `{"metadata": {"confidence": 0.7}}`, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.extractConfidence(tt.metadata)
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

func TestDefaultScoreCalculator_InitializeMaps(t *testing.T) {
	// Initialize calculator without config
	calc := &DefaultScoreCalculator{}

	scoreMap, confMap := calc.initializeMaps()

	// Check maps are initialized with correct keys
	expectedKeys := []string{"left", "center", "right"}
	for _, key := range expectedKeys {
		assert.Contains(t, scoreMap, key)
		assert.Contains(t, confMap, key)
		assert.Nil(t, scoreMap[key])
		assert.Nil(t, confMap[key])
	}

	// Check maps length
	assert.Len(t, scoreMap, 3)
	assert.Len(t, confMap, 3)
}
