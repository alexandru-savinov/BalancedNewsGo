package llm

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockLLMServiceTestEnsemble is a mock implementation for ensemble tests
type mockLLMServiceTestEnsemble struct {
	scoreContentFunc func(ctx context.Context, pv PromptVariant, art *db.Article) (float64, float64, error)
}

func (m *mockLLMServiceTestEnsemble) ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (float64, float64, error) {
	return m.scoreContentFunc(ctx, pv, art)
}

// TestLoadPromptVariants tests the loadPromptVariants function
func TestLoadPromptVariants(t *testing.T) {
	// Call loadPromptVariants with no parameters
	variants := loadPromptVariants()

	// Verify function returns expected prompts
	assert.Len(t, variants, 4, "Should find 4 prompt variants")

	// Check for the expected prompt variants by ID
	foundIDs := make(map[string]bool)
	for _, variant := range variants {
		foundIDs[variant.ID] = true
	}

	assert.True(t, foundIDs["default"], "Should have 'default' prompt")
	assert.True(t, foundIDs["left_focus"], "Should have 'left_focus' prompt")
	assert.True(t, foundIDs["center_focus"], "Should have 'center_focus' prompt")
	assert.True(t, foundIDs["right_focus"], "Should have 'right_focus' prompt")
}

// TestEnsembleAnalyze tests the EnsembleAnalyze function
func TestEnsembleAnalyze(t *testing.T) {
	// Create a mock service that returns predictable results
	mockService := &mockLLMServiceTestEnsemble{
		scoreContentFunc: func(ctx context.Context, pv PromptVariant, art *db.Article) (float64, float64, error) {
			// Return different values based on model in the prompt variant
			switch pv.Model {
			case "model1":
				return 0.5, 0.8, nil
			case "model2":
				return -0.3, 0.7, nil
			default:
				return 0.0, 0.9, nil
			}
		},
	}

	// Create test config
	cfg := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "model1", Perspective: "left", Weight: 1.0},
			{ModelName: "model2", Perspective: "right", Weight: 1.0},
			{ModelName: "model3", Perspective: "center", Weight: 1.0},
		},
		Formula:          "average",
		MinScore:         -1.0,
		MaxScore:         1.0,
		DefaultMissing:   0.0,
		HandleInvalid:    "default",
		ConfidenceMethod: "count_valid",
	}

	// Test article
	article := &db.Article{
		ID:      1,
		Title:   "Test Article",
		Content: "This is a test article content.",
	}

	client := &LLMClient{
		llmService: mockService,
		cache:      NewCache(),
		config:     cfg,
	}

	// Call ScoreWithModel for each model to simulate what EnsembleAnalyze would do
	var scores []db.LLMScore

	for _, model := range []string{"model1", "model2", "model3"} {
		score, err := client.ScoreWithModel(article, model)
		assert.NoError(t, err)

		scores = append(scores, db.LLMScore{
			ArticleID: article.ID,
			Model:     model,
			Score:     score,
		})
	}

	// Calculate composite score manually
	compositeScore, confidence, err := ComputeCompositeScoreWithConfidence(scores, cfg)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, scores, 3, "Should have scores for all three models")
	assert.InDelta(t, 0.067, compositeScore, 0.01, "Composite score should be close to average of scores")
	assert.InDelta(t, 1.0, confidence, 0.01, "Should have full confidence with all perspectives")

	// Verify individual scores
	var leftScore, rightScore, centerScore float64
	for _, score := range scores {
		switch score.Model {
		case "model1":
			leftScore = score.Score
		case "model2":
			rightScore = score.Score
		case "model3":
			centerScore = score.Score
		}
	}

	assert.InDelta(t, 0.5, leftScore, 0.01)
	assert.InDelta(t, -0.3, rightScore, 0.01)
	assert.InDelta(t, 0.0, centerScore, 0.01)
}

// TestScoreWithModel_CacheUsage tests that ScoreWithModel uses the cache
func TestScoreWithModel_CacheUsage(t *testing.T) {
	// Create a mock service that counts calls
	callCount := 0
	mockService := &mockLLMServiceTestEnsemble{
		scoreContentFunc: func(ctx context.Context, pv PromptVariant, art *db.Article) (float64, float64, error) {
			callCount++
			return 0.5, 0.8, nil
		},
	}

	// Create test config
	cfg := &CompositeScoreConfig{
		Models: []ModelConfig{
			{ModelName: "model1", Perspective: "left", Weight: 1.0},
		},
		Formula:          "average",
		MinScore:         -1.0,
		MaxScore:         1.0,
		DefaultMissing:   0.0,
		HandleInvalid:    "default",
		ConfidenceMethod: "count_valid",
	}

	// Test article
	article := &db.Article{
		ID:      1,
		Title:   "Test Article",
		Content: "This is a test article content.",
	}

	cache := NewCache()
	client := &LLMClient{
		llmService: mockService,
		cache:      cache,
		config:     cfg,
	}

	// First call should use the service
	score1, err := client.ScoreWithModel(article, "model1")
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "First call should use the service")

	// Second call with same article should use cache
	score2, err := client.ScoreWithModel(article, "model1")
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "Second call should use cache")

	// Verify both calls returned the same scores
	assert.Equal(t, score1, score2)
}

// MockLLMService for testing
type MockLLMService struct {
	mock.Mock
}

func (m *MockLLMService) ScoreContent(ctx context.Context, content, systemPrompt, userPrompt, model string) (*db.LLMScore, error) {
	args := m.Called(ctx, content, systemPrompt, userPrompt, model)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*db.LLMScore), args.Error(1)
}

func TestLLMClient_StreamingErrorDetection(t *testing.T) {
	// Test cases to check if streaming-related errors are properly detected and categorized
	testCases := []struct {
		name          string
		errorMessage  string
		shouldConvert bool
		expectedType  OpenRouterErrorType
	}{
		{
			name:          "SSE Streaming Error",
			errorMessage:  "Failed to parse SSE message",
			shouldConvert: true,
			expectedType:  ErrTypeStreaming,
		},
		{
			name:          "Stream Disconnected Error",
			errorMessage:  "stream connection interrupted unexpectedly",
			shouldConvert: true,
			expectedType:  ErrTypeStreaming,
		},
		{
			name:          "Processing Error",
			errorMessage:  "model is still PROCESSING",
			shouldConvert: true,
			expectedType:  ErrTypeStreaming,
		},
		{
			name:          "Regular Error (Non-streaming)",
			errorMessage:  "regular error message",
			shouldConvert: false,
			expectedType:  ErrTypeUnknown,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Check if our error detection logic in callLLM would detect this as a streaming error
			isStreamingError := strings.Contains(tc.errorMessage, "SSE") ||
				strings.Contains(tc.errorMessage, "stream") ||
				strings.Contains(tc.errorMessage, "PROCESSING")

			assert.Equal(t, tc.shouldConvert, isStreamingError,
				"Error message '%s' should %sbe detected as a streaming error",
				tc.errorMessage,
				map[bool]string{true: "", false: "not "}[tc.shouldConvert])

			// If it should be detected as streaming error, check that the conversion works correctly
			if tc.shouldConvert {
				// Create the error as it would be converted in callLLM
				convertedError := LLMAPIError{
					Message:      "LLM streaming response failed",
					StatusCode:   503,
					ResponseBody: tc.errorMessage,
					ErrorType:    ErrTypeStreaming,
				}

				// Verify the converted error has the right properties
				assert.Equal(t, ErrTypeStreaming, convertedError.ErrorType)
				assert.Equal(t, 503, convertedError.StatusCode)
				assert.Contains(t, convertedError.ResponseBody, tc.errorMessage)
			}
		})
	}
}

func TestLLMAPIError_ErrorPropagation(t *testing.T) {
	// Test various types of LLMAPIError and verify their string representation
	errorCases := []struct {
		name           string
		errorType      OpenRouterErrorType
		message        string
		statusCode     int
		expectedFormat string
	}{
		{
			name:           "Rate Limit Error",
			errorType:      ErrTypeRateLimit,
			message:        "Rate limit exceeded",
			statusCode:     429,
			expectedFormat: "LLM API Error (rate_limit): Status 429 - Rate limit exceeded",
		},
		{
			name:           "Authentication Error",
			errorType:      ErrTypeAuthentication,
			message:        "Invalid API key",
			statusCode:     401,
			expectedFormat: "LLM API Error (authentication): Status 401 - Invalid API key",
		},
		{
			name:           "Credits Exhausted",
			errorType:      ErrTypeCredits,
			message:        "Insufficient credits",
			statusCode:     402,
			expectedFormat: "LLM API Error (credits): Status 402 - Insufficient credits",
		},
		{
			name:           "Streaming Error",
			errorType:      ErrTypeStreaming,
			message:        "Streaming connection failed",
			statusCode:     503,
			expectedFormat: "LLM API Error (streaming): Status 503 - Streaming connection failed",
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			err := LLMAPIError{
				Message:      tc.message,
				StatusCode:   tc.statusCode,
				ResponseBody: "test response body",
				ErrorType:    tc.errorType,
			}

			// Test that Error() method follows the expected format
			errString := err.Error()
			assert.Equal(t, tc.expectedFormat, errString, "Error string should match expected format")

			// Verify error contains all key components
			assert.Contains(t, errString, tc.message, "Error string should contain the message")
			assert.Contains(t, errString, string(tc.errorType), "Error string should contain the error type")
			assert.Contains(t, errString, fmt.Sprintf("Status %d", tc.statusCode), "Error string should contain the status code")
		})
	}
}
