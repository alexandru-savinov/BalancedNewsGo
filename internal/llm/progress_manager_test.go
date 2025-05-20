package llm

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewProgressManager(t *testing.T) {
	pm := NewProgressManager(time.Minute)
	assert.NotNil(t, pm)
	assert.NotNil(t, pm.progressMap)
}

func TestProgressManagerSetGet(t *testing.T) {
	pm := NewProgressManager(time.Minute)

	// Set and get for a new ID
	articleID := int64(123)
	progress := &models.ProgressState{
		Step:        "Analyzing content",
		Message:     "Processing article content",
		Percent:     50,
		Status:      "InProgress",
		LastUpdated: time.Now().Unix(),
	}

	pm.SetProgress(articleID, progress)

	// Get the progress
	retrieved := pm.GetProgress(articleID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, progress.Status, retrieved.Status)
	assert.Equal(t, progress.Step, retrieved.Step)
	assert.Equal(t, progress.Percent, retrieved.Percent)

	// Get non-existent progress
	retrieved = pm.GetProgress(int64(999))
	assert.Nil(t, retrieved)

	// Update existing progress
	updatedProgress := &models.ProgressState{
		Step:        "Analysis done",
		Message:     "Processing complete",
		Percent:     100,
		Status:      "Completed",
		LastUpdated: time.Now().Unix(),
	}
	pm.SetProgress(articleID, updatedProgress)

	// Get the updated progress
	retrieved = pm.GetProgress(articleID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, updatedProgress.Status, retrieved.Status)
	assert.Equal(t, updatedProgress.Step, retrieved.Step)
	assert.Equal(t, updatedProgress.Percent, retrieved.Percent)
}

// TestManualCleanup tests the cleanup functionality directly rather than
// waiting for the background routine to run
func TestManualCleanup(t *testing.T) {
	pm := NewProgressManager(time.Hour) // Long interval so auto-cleanup doesn't interfere

	// Add a completed progress entry (should be cleaned up)
	completedID := int64(1)
	completedProgress := &models.ProgressState{
		Step:        "Complete",
		Status:      "Success",
		Percent:     100,
		LastUpdated: time.Now().Add(-10 * time.Minute).Unix(), // Older than 5 minutes
	}
	pm.SetProgress(completedID, completedProgress)

	// Add an error progress entry (should be cleaned up)
	errorID := int64(2)
	errorProgress := &models.ProgressState{
		Step:        "Error",
		Status:      "Error",
		Percent:     50,
		LastUpdated: time.Now().Add(-10 * time.Minute).Unix(), // Older than 5 minutes
	}
	pm.SetProgress(errorID, errorProgress)

	// Add a stale in-progress entry (should be cleaned up)
	staleID := int64(3)
	staleProgress := &models.ProgressState{
		Step:        "Stalled",
		Status:      "InProgress",
		Percent:     25,
		LastUpdated: time.Now().Add(-120 * time.Minute).Unix(), // Older than 30 minutes
	}
	pm.SetProgress(staleID, staleProgress)

	// Add a recent in-progress entry (should remain)
	recentID := int64(4)
	recentProgress := &models.ProgressState{
		Step:        "Fresh",
		Status:      "InProgress",
		Percent:     75,
		LastUpdated: time.Now().Unix(), // Current
	}
	pm.SetProgress(recentID, recentProgress)

	// Manually trigger cleanup
	pm.cleanup()

	// Check what remains
	assert.Nil(t, pm.GetProgress(completedID), "Completed progress should be cleaned up")
	assert.Nil(t, pm.GetProgress(errorID), "Error progress should be cleaned up")
	assert.Nil(t, pm.GetProgress(staleID), "Stale progress should be cleaned up")
	assert.NotNil(t, pm.GetProgress(recentID), "Recent in-progress should remain")
}

func TestProgressManager_UpdateProgressWithLLMError(t *testing.T) {
	pm := NewProgressManager(time.Minute)

	// Test cases with different types of errors
	testCases := []struct {
		name            string
		err             error
		expectedType    string
		expectedStatus  int
		checkRetryAfter bool
		expectedRetry   int
	}{
		{
			name: "LLM Rate Limit Error",
			err: LLMAPIError{
				Message:      "Rate limit exceeded",
				StatusCode:   429,
				ResponseBody: "limit exceeded",
				ErrorType:    ErrTypeRateLimit,
				RetryAfter:   30,
			},
			expectedType:    string(ErrTypeRateLimit),
			expectedStatus:  429,
			checkRetryAfter: true,
			expectedRetry:   30,
		},
		{
			name: "LLM Authentication Error",
			err: LLMAPIError{
				Message:      "Invalid API key",
				StatusCode:   401,
				ResponseBody: "auth failed",
				ErrorType:    ErrTypeAuthentication,
			},
			expectedType:    string(ErrTypeAuthentication),
			expectedStatus:  401,
			checkRetryAfter: false,
		},
		{
			name: "LLM Credits Error",
			err: LLMAPIError{
				Message:      "Insufficient credits",
				StatusCode:   402,
				ResponseBody: "payment required",
				ErrorType:    ErrTypeCredits,
			},
			expectedType:    string(ErrTypeCredits),
			expectedStatus:  402,
			checkRetryAfter: false,
		},
		{
			name: "LLM Streaming Error",
			err: LLMAPIError{
				Message:      "Streaming failed",
				StatusCode:   503,
				ResponseBody: "streaming error",
				ErrorType:    ErrTypeStreaming,
			},
			expectedType:    string(ErrTypeStreaming),
			expectedStatus:  503,
			checkRetryAfter: false,
		},
		{
			name: "LLM Unknown Error",
			err: LLMAPIError{
				Message:      "Unknown error",
				StatusCode:   500,
				ResponseBody: "unknown error",
				ErrorType:    ErrTypeUnknown,
			},
			expectedType:    string(ErrTypeUnknown),
			expectedStatus:  500,
			checkRetryAfter: false,
		},
		{
			name:            "Normal Error",
			err:             errors.New("regular error"),
			expectedType:    "",
			expectedStatus:  0,
			checkRetryAfter: false,
		},
		{
			name:            "No Error",
			err:             nil,
			expectedType:    "",
			expectedStatus:  0,
			checkRetryAfter: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test article ID
			articleID := int64(12345)

			// Update progress with the test error
			pm.UpdateProgress(articleID, "test_step", 50, "error", tc.err)

			// Get the progress
			progress := pm.GetProgress(articleID)

			// Verify error is stored correctly
			if tc.err == nil {
				assert.Empty(t, progress.Error, "Error should be empty for nil error")
				assert.Empty(t, progress.ErrorDetails, "ErrorDetails should be empty for nil error")
			} else {
				assert.NotEmpty(t, progress.Error, "Error should not be empty")

				// For LLM API errors, verify error details
				if tc.expectedType != "" {
					assert.NotEmpty(t, progress.ErrorDetails, "ErrorDetails should not be empty for LLMAPIError")

					// Parse error details JSON
					var details map[string]interface{}
					err := json.Unmarshal([]byte(progress.ErrorDetails), &details)

					assert.NoError(t, err, "ErrorDetails should be valid JSON")
					assert.Equal(t, tc.expectedType, details["type"], "Error type should match")
					assert.Equal(t, float64(tc.expectedStatus), details["status_code"], "Status code should match")

					// Check RetryAfter if applicable
					if tc.checkRetryAfter {
						retryValue, exists := details["retry_after"]
						assert.True(t, exists, "RetryAfter should be in error details for rate limit errors")
						assert.Equal(t, float64(tc.expectedRetry), retryValue, "RetryAfter value should match")
					}

					// Make sure we have a message
					assert.NotEmpty(t, details["message"], "Error details should include message")
				} else {
					// For regular errors, just verify the error message
					assert.Equal(t, tc.err.Error(), progress.Error, "Error message should match")
				}
			}
		})
	}
}

// TestProgressManager_ClearProgress tests clearing progress using UpdateProgress with nil error
func TestProgressManager_ClearProgress(t *testing.T) {
	pm := NewProgressManager(time.Minute)

	// Set up some test progress with errors
	articleID := int64(12345)
	pm.UpdateProgress(articleID, "test_step", 50, "error", LLMAPIError{
		Message:    "Rate limit exceeded",
		StatusCode: 429,
		ErrorType:  ErrTypeRateLimit,
	})

	// Verify error is stored
	progress := pm.GetProgress(articleID)
	assert.NotEmpty(t, progress.Error)
	assert.NotEmpty(t, progress.ErrorDetails)

	// Clear by setting progress with nil error
	pm.UpdateProgress(articleID, "test_step", 0, "reset", nil)

	// Verify error is cleared
	progress = pm.GetProgress(articleID)
	assert.Empty(t, progress.Error)
	assert.Empty(t, progress.ErrorDetails)
}

// TestProgressManager_MultipleArticles tests managing progress for multiple articles
func TestProgressManager_MultipleArticles(t *testing.T) {
	pm := NewProgressManager(time.Minute)

	// Set up multiple article progress states with different errors
	pm.UpdateProgress(int64(1), "analyze", 30, "error", LLMAPIError{
		Message:    "Rate limit exceeded",
		StatusCode: 429,
		ErrorType:  ErrTypeRateLimit,
	})

	pm.UpdateProgress(int64(2), "fetch", 100, "completed", nil)

	pm.UpdateProgress(int64(3), "score", 50, "error", errors.New("regular error"))

	// Verify each article's progress
	progress1 := pm.GetProgress(int64(1))
	assert.Equal(t, "error", progress1.Status)
	assert.NotEmpty(t, progress1.ErrorDetails)

	progress2 := pm.GetProgress(int64(2))
	assert.Equal(t, "completed", progress2.Status)
	assert.Empty(t, progress2.Error)
	assert.Empty(t, progress2.ErrorDetails)

	progress3 := pm.GetProgress(int64(3))
	assert.Equal(t, "error", progress3.Status)
	assert.Equal(t, "regular error", progress3.Error)
	assert.Empty(t, progress3.ErrorDetails)
}

func TestProgressManager_ExportStateWithErrors(t *testing.T) {
	pm := NewProgressManager(time.Minute)

	// Set up progress with error
	articleID := int64(123)
	pm.UpdateProgress(articleID, "analyze", 75.0, "error", LLMAPIError{
		Message:    "Streaming failed",
		StatusCode: 503,
		ErrorType:  ErrTypeStreaming,
	})

	// Get progress state directly
	progress := pm.GetProgress(articleID)

	// Verify progress contains the error info
	assert.NotNil(t, progress, "Progress should exist for article ID")
	assert.Equal(t, "error", progress.Status)
	assert.NotEmpty(t, progress.Error)
	assert.NotEmpty(t, progress.ErrorDetails)

	// Parse error details
	var details map[string]interface{}
	err := json.Unmarshal([]byte(progress.ErrorDetails), &details)
	assert.NoError(t, err)
	assert.Equal(t, string(ErrTypeStreaming), details["type"])
}
