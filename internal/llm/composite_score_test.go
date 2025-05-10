package llm

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var now = time.Now()

func createScoreWithConfidence(model string, score float64, confidence float64, t ...time.Time) db.LLMScore {
	createdAt := now
	if len(t) > 0 {
		createdAt = t[0]
	}
	metadata := fmt.Sprintf(`{"confidence":%.2f}`, confidence)
	return db.LLMScore{Model: model, Score: score, Metadata: metadata, CreatedAt: createdAt}
}

// TestComputeCompositeScoreWithAllZeroResponses tests the critical edge case
// where all LLMs return empty or zero-value responses, including ensemble results.
func TestComputeCompositeScoreWithAllZeroResponses(t *testing.T) {
	// Create a specific test config
	testCfg := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "model1", Perspective: "left"},
			{ModelName: "model2", Perspective: "center"},
			{ModelName: "model3", Perspective: "right"},
			{ModelName: "ensemble", Perspective: "center"},
		},
		Formula: "average", DefaultMissing: 0.0, HandleInvalid: "default",
		MinScore: -1.0, MaxScore: 1.0, ConfidenceMethod: "count_valid",
		MinConfidence: 0.0, MaxConfidence: 1.0,
	}

	testCases := []struct {
		name          string
		scores        []db.LLMScore
		expectError   bool
		errorContains string
	}{
		{
			name: "All models return invalid scores",
			scores: []db.LLMScore{
				{Model: "model1", Score: math.NaN(), Metadata: `{"confidence": 0.0}`},
				{Model: "model2", Score: math.Inf(1), Metadata: `{"confidence": 0.0}`},
				{Model: "model3", Score: -2.0, Metadata: `{"confidence": 0.0}`}, // Out of bounds
			},
			expectError:   true,
			errorContains: ErrAllPerspectivesInvalid.Error(),
		},
		{
			name: "All models return zero confidence",
			scores: []db.LLMScore{
				{Model: "model1", Score: 0.1, Metadata: `{"confidence": 0.0}`},
				{Model: "model2", Score: 0.2, Metadata: `{"confidence": 0.0}`},
				{Model: "model3", Score: 0.3, Metadata: `{"confidence": 0.0}`},
			},
			expectError:   true,
			errorContains: ErrAllPerspectivesInvalid.Error(),
		},
		{
			name: "Only ensemble score with valid confidence",
			scores: []db.LLMScore{
				{
					Model:    "ensemble",
					Score:    0.7,
					Metadata: `{"all_sub_results":[{"model":"model1","score":0.1,"confidence":0.8},{"model":"model2","score":-0.1,"confidence":0.7}],"confidence":0.9,"final_aggregation":{"weighted_mean":0,"variance":0.1,"uncertainty_flag":false},"per_model_results":{},"per_model_aggregation":{},"timestamp":"2024-04-28T12:00:00Z"}`,
				},
			},
			expectError: false,
		},
		{
			name: "Only ensemble score with zero confidence",
			scores: []db.LLMScore{
				{
					Model:    "ensemble",
					Score:    0.0,
					Metadata: `{"all_sub_results":[{"model":"model1","score":0.1,"confidence":0.8},{"model":"model2","score":-0.1,"confidence":0.7}],"confidence":0,"final_aggregation":{"weighted_mean":0,"variance":1.0,"uncertainty_flag":true},"per_model_results":{},"per_model_aggregation":{},"timestamp":"2024-04-28T12:00:00Z"}`,
				},
			},
			expectError:   true,
			errorContains: ErrAllPerspectivesInvalid.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ComputeCompositeScoreWithConfidenceFixed(tc.scores, testCfg)
			if tc.expectError {
				assert.Error(t, err)
				if err != nil && tc.errorContains != "" {
					assert.ErrorIs(t, err, ErrAllPerspectivesInvalid)
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComputeCompositeScoreWithConfidence(t *testing.T) {
	// Create a specific test config
	testCfg := &CompositeScoreConfig{
		Formula: "average", DefaultMissing: 0.0, Models: []ModelConfig{{Perspective: "left", ModelName: "left"}, {Perspective: "center", ModelName: "center"}, {Perspective: "right", ModelName: "right"}}, MinScore: -1, MaxScore: 1, ConfidenceMethod: "count_valid", MinConfidence: 0, MaxConfidence: 1,
	}
	scores := []db.LLMScore{
		{Model: "left", Score: -1.0, Metadata: `{"confidence":0.9}`},
		{Model: "center", Score: 0.0, Metadata: `{"confidence":0.8}`},
		{Model: "right", Score: 1.0, Metadata: `{"confidence":0.85}`},
	}
	score, confidence, err := ComputeCompositeScoreWithConfidenceFixed(scores, testCfg) // Pass testCfg
	assert.NoError(t, err)
	assert.InDelta(t, 0.0, score, 1e-9)
	assert.InDelta(t, 1.0, confidence, 1e-9) // Assert confidence based on method
}

func TestComputeCompositeScoreEdgeCases(t *testing.T) {
	testCases := []struct {
		name             string
		scores           []db.LLMScore
		configOverride   *CompositeScoreConfig
		expectedScore    float64
		expectedConf     float64
		expectError      bool
		expectedErrorIs  error
		description      string
		customAssertions func(t *testing.T, score float64, conf float64, err error)
	}{
		{
			name:            "empty scores array",
			scores:          []db.LLMScore{},
			expectedScore:   0.0,
			expectedConf:    0.0,
			expectError:     true,
			expectedErrorIs: ErrAllPerspectivesInvalid,
			description:     "Empty scores array should result in an error and default/zero values",
		},
		{
			name: "values outside bounds",
			scores: []db.LLMScore{
				createScoreWithConfidence("left", -1.5, 0.9), // Outside default -1 to 1
				createScoreWithConfidence("center", 0.1, 0.8),
				createScoreWithConfidence("right", 1.5, 0.85), // Outside default -1 to 1
			},
			configOverride: &CompositeScoreConfig{HandleInvalid: "ignore", DefaultMissing: 0.0, MinScore: -1.0, MaxScore: 1.0, Formula: "average", ConfidenceMethod: "count_valid"},
			expectedScore:  0.1, // Only center score is valid
			expectedConf:   0.8, // Only center confidence is valid
			expectError:    false,
		},
		{
			name: "all zero confidence",
			scores: []db.LLMScore{
				createScoreWithConfidence("left", 0.0, 0.0),
				createScoreWithConfidence("center", 0.0, 0.0),
				createScoreWithConfidence("right", 0.0, 0.0),
			},
			configOverride:  &CompositeScoreConfig{HandleInvalid: "ignore", DefaultMissing: 0.0, MinScore: -1.0, MaxScore: 1.0, Formula: "average", ConfidenceMethod: "count_valid"},
			expectedScore:   0.0,
			expectedConf:    0.0,
			expectError:     true,
			expectedErrorIs: ErrAllPerspectivesInvalid,
		},
		{
			name: "extreme values within bounds",
			scores: []db.LLMScore{
				createScoreWithConfidence("left", -1.0, 0.9),
				createScoreWithConfidence("center", 0.0, 0.8),
				createScoreWithConfidence("right", 1.0, 0.85),
			},
			configOverride: &CompositeScoreConfig{HandleInvalid: "ignore", DefaultMissing: 0.0, MinScore: -1.0, MaxScore: 1.0, Formula: "average", ConfidenceMethod: "count_valid"},
			expectedScore:  0.0, // Avg(-1, 0, 1) = 0
			expectedConf:   1.0,
			expectError:    false,
		},
		{
			name: "non-standard model names",
			scores: []db.LLMScore{
				createScoreWithConfidence("custom-left-model", -0.5, 0.9),
				createScoreWithConfidence("custom-center-model", 0.0, 0.8),
				createScoreWithConfidence("custom-right-model", 0.5, 0.85),
			},
			configOverride: &CompositeScoreConfig{
				Models: []ModelConfig{
					{ModelName: "custom-left-model", Perspective: "left"},
					{ModelName: "custom-center-model", Perspective: "center"},
					{ModelName: "custom-right-model", Perspective: "right"},
				},
				Formula: "average", HandleInvalid: "ignore", DefaultMissing: 0.0, MinScore: -1.0, MaxScore: 1.0, ConfidenceMethod: "count_valid",
			},
			expectedScore: 0.0, // Avg(-0.5, 0, 0.5) = 0
			expectedConf:  1.0,
			expectError:   false,
		},
		{
			name: "case insensitive model names",
			scores: []db.LLMScore{
				createScoreWithConfidence("LeFt", -0.5, 0.9),
				createScoreWithConfidence("cEnTeR", 0.0, 0.8),
				createScoreWithConfidence("RIGHT", 0.5, 0.85),
			},
			configOverride: &CompositeScoreConfig{
				Models: []ModelConfig{
					{ModelName: "left", Perspective: "left"},
					{ModelName: "center", Perspective: "center"},
					{ModelName: "right", Perspective: "right"},
				},
				Formula: "average", HandleInvalid: "ignore", DefaultMissing: 0.0, MinScore: -1.0, MaxScore: 1.0, ConfidenceMethod: "count_valid",
			},
			expectedScore: 0.0, // Avg(-0.5, 0, 0.5) = 0
			expectedConf:  1.0,
			expectError:   false,
		},
		{
			name: "NaN values - ignored",
			scores: []db.LLMScore{
				createScoreWithConfidence("left", math.NaN(), 0.9),
				createScoreWithConfidence("center", 0.2, 0.8),
				createScoreWithConfidence("right", 0.4, 0.85),
			},
			configOverride: &CompositeScoreConfig{HandleInvalid: "ignore", DefaultMissing: 0.0, MinScore: -1.0, MaxScore: 1.0, Formula: "average", ConfidenceMethod: "count_valid"},
			expectedScore:  0.3,       // (0.2+0.4)/2 = 0.3. Left is ignored.
			expectedConf:   2.0 / 3.0, // 2/3 valid perspectives
			expectError:    false,
		},
		{
			name: "All Infinity values - ignored",
			scores: []db.LLMScore{
				createScoreWithConfidence("left", math.Inf(1), 0.9),
				createScoreWithConfidence("center", math.Inf(-1), 0.8),
				createScoreWithConfidence("right", math.Inf(1), 0.85),
			},
			configOverride:  &CompositeScoreConfig{HandleInvalid: "ignore", MinScore: -1.0, MaxScore: 1.0, Formula: "average", ConfidenceMethod: "count_valid"},
			expectedScore:   0.0,
			expectedConf:    0.0,
			expectError:     true,
			expectedErrorIs: ErrAllPerspectivesInvalid,
		},
		{
			name: "Out-of-range ignored with one valid score",
			scores: []db.LLMScore{
				createScoreWithConfidence("left", 2.0, 0.9, now.Add(time.Second*1)),    // Invalid, ignored
				createScoreWithConfidence("center", -2.0, 0.8, now.Add(time.Second*2)), // Invalid, ignored
				createScoreWithConfidence("right", 0.5, 0.85, now.Add(time.Second*3)),  // Valid
			},
			configOverride: &CompositeScoreConfig{HandleInvalid: "ignore", DefaultMissing: 0.0, MinScore: -1.0, MaxScore: 1.0, Formula: "average", ConfidenceMethod: "count_valid"},
			expectedScore:  0.5,       // Only right (0.5) is used. Avg(0.5)/1 = 0.5
			expectedConf:   1.0 / 3.0, // 1/3 valid perspectives
			expectError:    false,
		},
		{
			name: "weighted formula with config override",
			scores: []db.LLMScore{
				createScoreWithConfidence("left", 0.1, 0.9),
				createScoreWithConfidence("center", 0.2, 0.8),
				createScoreWithConfidence("right", 0.3, 0.85),
			},
			configOverride: &CompositeScoreConfig{
				Formula:          "weighted",
				Weights:          map[string]float64{"left": 1, "center": 2, "right": 3},
				HandleInvalid:    "default",
				DefaultMissing:   0.0,
				MinScore:         -1.0,
				MaxScore:         1.0,
				ConfidenceMethod: "count_valid",
			},
			expectedScore: (0.1*1 + 0.2*2 + 0.3*3) / (1 + 2 + 3), // (0.1 + 0.4 + 0.9) / 6 = 1.4 / 6 = 0.2333...
			expectedConf:  1.0,
			expectError:   false,
		},
		{
			name: "ignore invalid with config override", // Effectively all invalid and ignored
			scores: []db.LLMScore{
				createScoreWithConfidence("left", math.NaN(), 0.9),
				createScoreWithConfidence("center", math.Inf(1), 0.8),
				createScoreWithConfidence("right", -2.0, 0.85), // out of bound if min is -1
			},
			configOverride:  &CompositeScoreConfig{HandleInvalid: "ignore", DefaultMissing: 0.0, MinScore: -1.0, MaxScore: 1.0, Formula: "average", ConfidenceMethod: "count_valid"},
			expectedScore:   0.0, // No valid scores, should trigger ErrAllPerspectivesInvalid, score should be DefaultMissing from top func
			expectedConf:    0.0, // No valid scores
			expectError:     true,
			expectedErrorIs: ErrAllPerspectivesInvalid,
		},
		{
			name: "duplicate model scores - should use last one",
			scores: []db.LLMScore{
				createScoreWithConfidence("left", 0.1, 0.9, now.Add(time.Millisecond*100)), // Older
				createScoreWithConfidence("center", 0.2, 0.8, now.Add(time.Millisecond*200)),
				createScoreWithConfidence("right", 0.3, 0.85, now.Add(time.Millisecond*300)),
				createScoreWithConfidence("left", 0.4, 0.92, now.Add(time.Millisecond*400)),   // Newer left
				createScoreWithConfidence("center", 0.5, 0.82, now.Add(time.Millisecond*500)), // Newer center
				createScoreWithConfidence("right", 0.6, 0.88, now.Add(time.Millisecond*600)),  // Newer right
			},
			configOverride: &CompositeScoreConfig{HandleInvalid: "default", DefaultMissing: 0.0, Formula: "average", ConfidenceMethod: "count_valid", MinScore: -1, MaxScore: 1},
			expectedScore:  0.5, // Avg(0.4, 0.5, 0.6) = 1.5/3 = 0.5
			expectedConf:   1.0,
			expectError:    false,
		},
		{
			name: "custom min/max bounds",
			scores: []db.LLMScore{
				createScoreWithConfidence("left", -3.0, 0.9),
				createScoreWithConfidence("center", 0.0, 0.8),
				createScoreWithConfidence("right", 3.0, 0.85),
			},
			configOverride: &CompositeScoreConfig{HandleInvalid: "default", DefaultMissing: -100.0, MinScore: -5.0, MaxScore: 5.0, Formula: "average", ConfidenceMethod: "count_valid"},
			expectedScore:  0.0, // All scores are valid with these bounds. Avg(-3.0, 0.0, 3.0) = 0.0/3 = 0.0
			expectedConf:   1.0, // 3/3 valid
			expectError:    false,
		},
	}

	// Load default config for tests that don't override it
	defaultCfg, err := LoadCompositeScoreConfig()
	require.NoError(t, err, "Failed to load default test config")

	for _, tc := range testCases {
		t := t // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run test cases in parallel

			cfgToUse := defaultCfg
			if tc.configOverride != nil {
				// If specific fields aren't set in override, copy from defaultCfg to ensure a complete config.
				// This is a shallow copy, modify if deep copy is needed for nested structs/maps.
				tempCfg := *tc.configOverride // Corrected typo empCfg to tempCfg
				if tempCfg.Models == nil {    // Essential for perspective mapping
					tempCfg.Models = defaultCfg.Models
				}
				// Ensure MinScore/MaxScore are set if not in override, to prevent NaN comparisons with 0
				if tempCfg.MinScore == 0 && tempCfg.MaxScore == 0 && (defaultCfg.MinScore != 0 || defaultCfg.MaxScore != 0) {
					// Only copy if default has them and override doesn't (avoids overwriting intentional 0s)
					// For safety, if an override *doesn't* set them, use defaults.
					if tc.configOverride.MinScore == 0.0 && defaultCfg.MinScore != 0.0 {
						tempCfg.MinScore = defaultCfg.MinScore
					}
					if tc.configOverride.MaxScore == 0.0 && defaultCfg.MaxScore != 0.0 {
						tempCfg.MaxScore = defaultCfg.MaxScore
					}
				}
				cfgToUse = &tempCfg
			}

			score, conf, err := ComputeCompositeScoreWithConfidenceFixed(tc.scores, cfgToUse)

			if tc.expectError {
				assert.Error(t, err, fmt.Sprintf("%s: Expected error", tc.name))
				if tc.expectedErrorIs != nil {
					assert.True(t, errors.Is(err, tc.expectedErrorIs), fmt.Sprintf("%s: Expected error type %T, got %T (%v)", tc.name, tc.expectedErrorIs, err, err))
				}
				// When an error is expected (especially ErrAllPerspectivesInvalid), the returned score/conf might be defaults from the top-level function.
				// Assert these explicitly if they differ from typical non-error case expectations.
				assert.InDelta(t, tc.expectedScore, score, 1e-9, fmt.Sprintf("%s: Expected score %f, got %f with error", tc.name, tc.expectedScore, score))
				assert.InDelta(t, tc.expectedConf, conf, 1e-9, fmt.Sprintf("%s: Expected confidence %f, got %f with error", tc.name, tc.expectedConf, conf))
			} else {
				assert.NoError(t, err, fmt.Sprintf("%s: Expected no error, got %v", tc.name, err))
				assert.InDelta(t, tc.expectedScore, score, 1e-9, fmt.Sprintf("%s: Expected score %f, got %f", tc.name, tc.expectedScore, score))
				assert.InDelta(t, tc.expectedConf, conf, 1e-9, fmt.Sprintf("%s: Expected confidence %f, got %f", tc.name, tc.expectedConf, conf))
			}

			if tc.customAssertions != nil {
				tc.customAssertions(t, score, conf, err)
			}
		})
	}
}

func TestComputeCompositeScoreWeightedCalculation(t *testing.T) {
	// Create a base config for weighted tests
	baseCfg := &CompositeScoreConfig{
		Formula: "weighted", DefaultMissing: 0.0,
		Models:   []ModelConfig{{Perspective: "left", ModelName: "left"}, {Perspective: "center", ModelName: "center"}, {Perspective: "right", ModelName: "right"}},
		MinScore: -1, MaxScore: 1, ConfidenceMethod: "count_valid", MinConfidence: 0, MaxConfidence: 1,
	}

	t.Run("Equal weights", func(t *testing.T) {
		scores := []db.LLMScore{
			{Model: "left", Score: 0.1, Metadata: `{"confidence":0.9}`},
			{Model: "center", Score: 0.1, Metadata: `{"confidence":0.9}`},
			{Model: "right", Score: 0.1, Metadata: `{"confidence":0.9}`},
		}
		testCfg := *baseCfg // Copy base config
		testCfg.Weights = map[string]float64{"left": 1.0, "center": 1.0, "right": 1.0}
		score, _, err := ComputeCompositeScoreWithConfidenceFixed(scores, &testCfg) // Pass testCfg
		assert.NoError(t, err)
		assert.InDelta(t, 0.1, score, 0.001, "Equal weights should calculate average score")
	})

	t.Run("Unequal weights", func(t *testing.T) {
		scores := []db.LLMScore{
			{Model: "left", Score: 0.1, Metadata: `{"confidence":0.9}`},
			{Model: "center", Score: 0.2, Metadata: `{"confidence":0.9}`},
			{Model: "right", Score: 0.3, Metadata: `{"confidence":0.9}`},
		}
		testCfg := *baseCfg // Copy base config
		testCfg.Weights = map[string]float64{"left": 0.2, "center": 0.3, "right": 0.5}
		score, _, err := ComputeCompositeScoreWithConfidenceFixed(scores, &testCfg) // Pass testCfg
		assert.NoError(t, err)
		assert.InDelta(t, 0.23, score, 0.001, "Unequal weights should apply correct weighting") // (0.1*0.2 + 0.2*0.3 + 0.3*0.5) / 1.0 = 0.02 + 0.06 + 0.15 = 0.23
	})

	t.Run("Zero weight", func(t *testing.T) {
		scores := []db.LLMScore{
			{Model: "left", Score: 0.1, Metadata: `{"confidence":0.9}`},
			{Model: "center", Score: 0.2, Metadata: `{"confidence":0.9}`},
			{Model: "right", Score: 0.3, Metadata: `{"confidence":0.9}`},
		}
		testCfg := *baseCfg // Copy base config
		testCfg.Weights = map[string]float64{"left": 0.0, "center": 0.5, "right": 0.5}
		score, _, err := ComputeCompositeScoreWithConfidenceFixed(scores, &testCfg) // Pass testCfg
		assert.NoError(t, err)
		assert.InDelta(t, 0.25, score, 0.001, "Zero weight should exclude that perspective") // (0.2*0.5 + 0.3*0.5) / 1.0 = 0.1 + 0.15 = 0.25
	})
}
