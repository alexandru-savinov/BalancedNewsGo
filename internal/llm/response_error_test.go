package llm

import (
	"context"
	"fmt"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
)

// TestInvalidContentJSON tests handling of valid outer JSON but invalid inner JSON content
func TestInvalidContentJSON(t *testing.T) {
	// Valid outer JSON structure but inner content is not valid JSON
	raw := `{"choices":[{"message":{"content":"This is not valid JSON"}}]}`

	_, _, _, err := parseNestedLLMJSONResponse(raw)

	// Should get an error about parsing inner content
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing inner content JSON")
}

// TestMalformedJSONWithBackticks tests handling of malformed JSON inside backticks
func TestMalformedJSONWithBackticks(t *testing.T) {
	// Valid outer JSON but backtick content is malformed
	raw := `{"choices":[{"message":{"content":"` +
		"```" + `json {\\\"score\\\":1.0,\\\"explanation\\\":\\\"test\\\",\\\"confidence\\\":INVALID} ` +
		"```" + `"}}]}`

	_, _, _, err := parseNestedLLMJSONResponse(raw)

	// Should get an error when parsing the inner JSON
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing inner content")
}

// TestMissingRequiredFields tests handling of JSON with missing required fields
func TestMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		expectError   bool
		expectedScore float64
		expectedConf  float64
		desc          string
	}{
		{
			name:          "missing score",
			raw:           `{"choices":[{"message":{"content":"{\"explanation\":\"test\",\"confidence\":0.9}"}}]}`,
			expectError:   false, // The parser doesn't validate required fields, it returns what it found
			expectedScore: 0,     // Default zero value when field is missing
			expectedConf:  0.9,
			desc:          "Missing score field should return zero value for score",
		},
		{
			name:          "missing explanation",
			raw:           `{"choices":[{"message":{"content":"{\"score\":0.5,\"confidence\":0.9}"}}]}`,
			expectError:   false,
			expectedScore: 0.5,
			expectedConf:  0.9,
			desc:          "Missing explanation field should return empty explanation",
		},
		{
			name:          "missing confidence",
			raw:           `{"choices":[{"message":{"content":"{\"score\":0.5,\"explanation\":\"test\"}"}}]}`,
			expectError:   false, // The parser doesn't validate confidence, it returns zero value
			expectedScore: 0.5,
			expectedConf:  0, // Default zero value when field is missing
			desc:          "Missing confidence field should return zero confidence",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, explanation, confidence, err := parseNestedLLMJSONResponse(tc.raw)

			if tc.expectError {
				assert.Error(t, err, tc.desc)
			} else {
				assert.NoError(t, err, tc.desc)
				assert.InDelta(t, tc.expectedScore, score, 0.001, "Score should match expected value")
				assert.InDelta(t, tc.expectedConf, confidence, 0.001, "Confidence should match expected value")
				if tc.name == "missing explanation" {
					assert.Equal(t, "", explanation, "Explanation should be empty")
				}
			}
		})
	}
}

// TestZeroConfidenceHandling tests how zero confidence values are handled
func TestZeroConfidenceHandling(t *testing.T) {
	// Valid JSON with zero confidence value
	raw := `{"choices":[{"message":{"content":"{\"score\":0.5,\"explanation\":\"test\",\"confidence\":0.0}"}}]}`

	_, _, confidence, err := parseNestedLLMJSONResponse(raw)

	// Parser should extract values successfully
	assert.NoError(t, err)
	assert.Equal(t, 0.0, confidence)

	// Note: The validation of zero confidence happens in callLLM, not in the parser
	// This test is just to verify the parser correctly extracts the zero value
}

// TestInvalidScoreValues tests handling of invalid score values
func TestInvalidScoreValues(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		score float64
	}{
		{
			name:  "out of range negative",
			raw:   `{"choices":[{"message":{"content":"{\"score\":-5.0,\"explanation\":\"test\",\"confidence\":0.9}"}}]}`,
			score: -5.0,
		},
		{
			name:  "out of range positive",
			raw:   `{"choices":[{"message":{"content":"{\"score\":5.0,\"explanation\":\"test\",\"confidence\":0.9}"}}]}`,
			score: 5.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, _, _, err := parseNestedLLMJSONResponse(tc.raw)

			// Parser should extract values successfully even if they're out of expected range
			assert.NoError(t, err)
			assert.Equal(t, tc.score, score)

			// Note: The validation of score range happens at a higher level
		})
	}
}

// TestParseWithExtraFields tests that extra fields in response don't cause issues
func TestParseWithExtraFields(t *testing.T) {
	raw := `{"choices":[{"message":{"content":"{\"score\":0.5,\"explanation\":\"test\",\"confidence\":0.9,\"extraField\":\"value\"}"}}]}`

	score, explanation, confidence, err := parseNestedLLMJSONResponse(raw)

	// Should ignore extra fields and parse successfully
	assert.NoError(t, err)
	assert.Equal(t, 0.5, score)
	assert.Equal(t, "test", explanation)
	assert.Equal(t, 0.9, confidence)
}

// TestEmptyContentResponse tests handling of empty content in API response
func TestEmptyContentResponse(t *testing.T) {
	// Valid outer structure but empty content
	raw := `{"choices":[{"message":{"content":""}}]}`

	_, _, _, err := parseNestedLLMJSONResponse(raw)

	// Should get an error when parsing the empty content
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing inner content")
}

// TestPartialResponse tests handling of truncated/partial responses
func TestPartialResponse(t *testing.T) {
	// Valid start of response but truncated
	raw := `{"choices":[{"message":{"content":"{\"score\":0.5,\"explanation\":`

	_, _, _, err := parseNestedLLMJSONResponse(raw)

	// Should fail at parsing outer JSON
	assert.Error(t, err)
}

// TestHTTPAPIErrorHandling tests how the service handles HTTP-level errors
func TestHTTPAPIErrorHandling(t *testing.T) {
	// Create a mock LLM service that returns various HTTP errors
	mockService := &mockErrorLLMService{
		errorType: "HTTP error",
		errorMsg:  "service unavailable",
	}

	// Create a client with our mock service
	client := &LLMClient{
		llmService: mockService,
	}

	// Test article
	article := &db.Article{
		ID:      1,
		Content: "Test content",
	}

	// Call ScoreWithModel to see how it handles HTTP errors
	_, err := client.ScoreWithModel(article, "test-model")

	// Should get an error
	assert.Error(t, err)
	// The actual error is wrapped in "llm_service_error: scoring with model test-model failed"
	assert.Contains(t, err.Error(), "llm_service_error")
}

// TestAPIErrorResponseHandling tests handling of API error responses in JSON
func TestAPIErrorResponseHandling(t *testing.T) {
	// Test parsing an API error response directly
	errorJSON := []byte(`{"error": {"message": "The server encountered an error", "type": "server_error", "code": 500}}`)

	responseStr, err := parseLLMAPIResponse(errorJSON)

	// Should extract error info properly
	assert.Error(t, err)
	assert.Empty(t, responseStr)
	// The actual error message contains "LLM service error response" rather than the raw error type
	assert.Contains(t, err.Error(), "LLM service error response")
}

// Mock service that always returns errors for testing error handling
type mockErrorLLMService struct {
	errorType string
	errorMsg  string
}

func (m *mockErrorLLMService) ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (float64, float64, error) {
	return 0, 0, fmt.Errorf("%s: %s", m.errorType, m.errorMsg)
}

// TestErrorRetryLogic tests that the retry logic exists and works
func TestErrorRetryLogic(t *testing.T) {
	// This test is more about validating that the retry infrastructure exists
	// than about testing the specific number of retries

	// Create a mock service
	mockService := &mockErrorLLMService{
		errorType: "temporary error",
		errorMsg:  "please retry",
	}

	// Create client with mock service
	client := &LLMClient{
		llmService: mockService,
	}

	// Test article
	article := &db.Article{
		ID:      1,
		Content: "Test content",
	}

	// Call ScoreWithModel - we expect it to fail after exhausting retries
	_, err := client.ScoreWithModel(article, "test-model")

	// Should eventually fail with error after retries
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llm_service_error")
}

// TestCompletelyEmptyResponse tests handling of a completely empty response from API
func TestCompletelyEmptyResponse(t *testing.T) {
	// Response with empty JSON object in content
	raw := `{"choices":[{"message":{"content":"{}"}}]}`

	score, explanation, confidence, err := parseNestedLLMJSONResponse(raw)

	// The current implementation doesn't treat missing fields as errors
	assert.NoError(t, err, "Parser shouldn't error on empty content JSON")
	assert.Equal(t, 0.0, score, "Score should be zero when missing")
	assert.Equal(t, "", explanation, "Explanation should be empty when missing")
	assert.Equal(t, 0.0, confidence, "Confidence should be zero when missing")

	// Also test with completely missing content
	rawMissingContent := `{"choices":[{"message":{"content":""}}]}`

	scoreMissing, explanationMissing, confidenceMissing, errMissing := parseNestedLLMJSONResponse(rawMissingContent)

	// This should produce an error since there's no JSON to parse
	assert.Error(t, errMissing, "Empty content string should cause an error")
	assert.Equal(t, 0.0, scoreMissing, "Score should be zero on error")
	assert.Equal(t, "", explanationMissing, "Explanation should be empty on error")
	assert.Equal(t, 0.0, confidenceMissing, "Confidence should be zero on error")
}
