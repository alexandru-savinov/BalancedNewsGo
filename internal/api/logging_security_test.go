package api

import (
	"bytes"
	"errors"
	"log"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/apperrors"
	"github.com/alexandru-savinov/BalancedNewsGo/internal/llm"
	"github.com/gin-gonic/gin"
)

// TestSanitizeErrorMessage tests the sanitizeErrorMessage function
func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal message",
			input:    "This is a normal error message",
			expected: "This is a normal error message",
		},
		{
			name:     "Message with newlines",
			input:    "Error message\nwith newlines\r\nand carriage returns",
			expected: "Error message with newlines  and carriage returns",
		},
		{
			name:     "Message with tabs",
			input:    "Error\tmessage\twith\ttabs",
			expected: "Error message with tabs",
		},
		{
			name:     "Long message truncation",
			input:    strings.Repeat("a", 250),
			expected: strings.Repeat("a", 200) + "...",
		},
		{
			name:     "Potential log injection",
			input:    "Error: invalid JSON\n[FAKE] Injected log entry",
			expected: "Error: invalid JSON [FAKE] Injected log entry",
		},
		{
			name:     "Mixed dangerous characters",
			input:    "Error\nwith\tmixed\rdangerous\ncharacters",
			expected: "Error with mixed dangerous characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeErrorMessage(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeErrorMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestLogErrorSanitization tests that LogError properly sanitizes error messages
func TestLogErrorSanitization(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer func() {
		log.SetOutput(nil) // Reset to default
	}()

	// Create test context
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	tests := []struct {
		name             string
		err              error
		operation        string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:      "Generic error with newlines",
			err:       errors.New("JSON decode error\nmalicious log injection attempt"),
			operation: "test_operation",
			shouldContain: []string{
				"Operation=test_operation",
				"Type=GENERIC",
				"JSON decode error malicious log injection attempt",
			},
			shouldNotContain: []string{
				"malicious log injection attempt\nmalicious",
			},
		},
		{
			name: "LLM error with dangerous characters",
			err: llm.LLMAPIError{
				Message:    "API error\nwith\tdangerous\rcharacters",
				StatusCode: 429,
				ErrorType:  llm.ErrTypeRateLimit,
			},
			operation: "llm_operation",
			shouldContain: []string{
				"Operation=llm_operation",
				"Type=LLM_ERROR",
				"API error with dangerous characters",
			},
			shouldNotContain: []string{
				"dangerous\ncharacters",
				"dangerous\tcharacters",
				"dangerous\rcharacters",
			},
		},
		{
			name: "App error with injection attempt",
			err: &apperrors.AppError{
				Code:    "validation_error",
				Message: "Validation failed\n[FAKE] Admin logged in successfully",
			},
			operation: "validation_operation",
			shouldContain: []string{
				"Operation=validation_operation",
				"Type=APP_ERROR",
				"Validation failed [FAKE] Admin logged in successfully",
			},
			shouldNotContain: []string{
				"failed\n[FAKE]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear buffer
			logBuffer.Reset()

			// Call LogError
			LogError(c, tt.err, tt.operation)

			// Get logged output
			logOutput := logBuffer.String()

			// Check that required strings are present
			for _, should := range tt.shouldContain {
				if !strings.Contains(logOutput, should) {
					t.Errorf("Log output should contain %q, but got: %s", should, logOutput)
				}
			}

			// Check that dangerous strings are not present
			for _, shouldNot := range tt.shouldNotContain {
				if strings.Contains(logOutput, shouldNot) {
					t.Errorf("Log output should not contain %q, but got: %s", shouldNot, logOutput)
				}
			}
		})
	}
}

// TestLogErrorWithNilError tests that LogError handles nil errors correctly
func TestLogErrorWithNilError(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer func() {
		log.SetOutput(nil) // Reset to default
	}()

	// Create test context
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Call LogError with nil error
	LogError(c, nil, "test_operation")

	// Should not log anything
	logOutput := logBuffer.String()
	if logOutput != "" {
		t.Errorf("LogError with nil error should not log anything, but got: %s", logOutput)
	}
}
