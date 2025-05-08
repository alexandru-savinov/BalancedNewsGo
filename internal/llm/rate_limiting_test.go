package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

const (
	testBearerPrefix   = "Bearer "
	testArticleContent = "Test article content"
	testModelName      = "test-model"
	testPromptTemplate = "Analyze: {{.Content}}"
	rateLimitErrorMsg  = "rate limit"
)

// mockLLMService is a test implementation of LLMService to test rate limiting
type mockLLMService struct {
	primaryKey      string
	backupKey       string
	primaryFailRate bool
	backupFailRate  bool
	primaryUsed     bool
	backupUsed      bool
}

func newMockLLMService(backupKey string, primaryFailRate, backupFailRate bool) *mockLLMService {
	return &mockLLMService{
		primaryKey:      "primary-key", // Hardcode as it's always the same in tests
		backupKey:       backupKey,
		primaryFailRate: primaryFailRate,
		backupFailRate:  backupFailRate,
	}
}

// ScoreContent implements LLMService for testing rate limiting and fallback
func (m *mockLLMService) ScoreContent(ctx context.Context, pv PromptVariant, art *db.Article) (score float64, confidence float64, err error) {
	// Try primary key first
	m.primaryUsed = true
	if m.primaryFailRate {
		// If using backup and backup key exists
		if m.backupKey != "" {
			m.backupUsed = true
			if m.backupFailRate {
				return 0, 0, fmt.Errorf("rate limit exceeded for both keys: %w", ErrBothLLMKeysRateLimited)
			}
			// Backup key works
			return 0.5, 0.8, nil
		}
		return 0, 0, fmt.Errorf("rate limit exceeded for primary key and no backup key provided")
	}
	// Primary key works
	return 0.3, 0.9, nil
}

// TestRateLimitFallback verifies that when the primary key is rate-limited,
// the service falls back to the secondary key
func TestRateLimitFallback(t *testing.T) {
	// Create a mock LLM service where primary key fails with rate limit but backup succeeds
	mockService := newMockLLMService("backup-key", true, false)

	// Test article and prompt
	article := &db.Article{
		ID:      1,
		Content: testArticleContent,
	}

	promptVariant := PromptVariant{
		ID:       "default",
		Template: testPromptTemplate,
		Model:    testModelName,
	}

	// Call ScoreContent
	score, confidence, err := mockService.ScoreContent(context.Background(), promptVariant, article)

	// Verify fallback worked, no error, got expected score/confidence
	assert.NoError(t, err, "Expected no error when fallback succeeds")
	assert.Equal(t, 0.5, score, "Expected score of 0.5")
	assert.Equal(t, 0.8, confidence, "Expected confidence of 0.8")
	assert.True(t, mockService.primaryUsed, "Expected primary key to be used")
	assert.True(t, mockService.backupUsed, "Expected backup key to be used")
}

// TestBothKeysRateLimited verifies that when both primary and secondary
// keys are rate-limited, ErrBothLLMKeysRateLimited is returned
func TestBothKeysRateLimited(t *testing.T) {
	// Create a mock LLM service where both keys fail with rate limit
	mockService := newMockLLMService("backup-key", true, true)

	// Test article and prompt
	article := &db.Article{
		ID:      2,
		Content: testArticleContent,
	}

	promptVariant := PromptVariant{
		ID:       "default",
		Template: testPromptTemplate,
		Model:    testModelName,
	}

	// Call ScoreContent
	_, _, err := mockService.ScoreContent(context.Background(), promptVariant, article)

	// Verify we get the expected rate limit error
	assert.Error(t, err, "Expected error when both keys are rate-limited")
	assert.True(t, strings.Contains(err.Error(), rateLimitErrorMsg), "Expected rate limit error message")
	assert.True(t, mockService.primaryUsed, "Expected primary key to be used")
	assert.True(t, mockService.backupUsed, "Expected backup key to be used")
}

// TestPrimaryKeyWorking verifies that when the primary key works,
// the secondary key is not used
func TestPrimaryKeyWorking(t *testing.T) {
	// Create a mock LLM service where primary key succeeds
	mockService := newMockLLMService("backup-key", false, false)

	// Test article and prompt
	article := &db.Article{
		ID:      3,
		Content: testArticleContent,
	}

	promptVariant := PromptVariant{
		ID:       "default",
		Template: testPromptTemplate,
		Model:    testModelName,
	}

	// Call ScoreContent
	score, confidence, err := mockService.ScoreContent(context.Background(), promptVariant, article)

	// Verify only primary key was used and got expected results
	assert.NoError(t, err, "Expected no error with working primary key")
	assert.Equal(t, 0.3, score, "Expected score from primary key")
	assert.Equal(t, 0.9, confidence, "Expected confidence from primary key")
	assert.True(t, mockService.primaryUsed, "Expected primary key to be used")
	assert.False(t, mockService.backupUsed, "Expected secondary key NOT to be used")
}

// TestNoBackupKey verifies behavior when primary key fails and no backup key is provided
func TestNoBackupKey(t *testing.T) {
	// Create a mock LLM service with no backup key where primary key fails
	mockService := newMockLLMService("", true, false)

	// Test article and prompt
	article := &db.Article{
		ID:      4,
		Content: testArticleContent,
	}

	promptVariant := PromptVariant{
		ID:       "default",
		Template: testPromptTemplate,
		Model:    testModelName,
	}

	// Call ScoreContent
	_, _, err := mockService.ScoreContent(context.Background(), promptVariant, article)

	// Verify we get an error since no fallback is available
	assert.Error(t, err, "Expected error when primary key fails and no backup exists")
	assert.True(t, strings.Contains(err.Error(), rateLimitErrorMsg), "Expected rate limit error message")
	assert.True(t, mockService.primaryUsed, "Expected primary key to be used")
	assert.False(t, mockService.backupUsed, "Expected backup key NOT to be used")
}

// TestHTTPLLMServiceRateLimiting tests the actual HTTPLLMService implementation with a mock HTTP server
func TestHTTPLLMServiceRateLimiting(t *testing.T) {
	// Create test server that emulates rate limiting for primary key only
	primaryKey := "test-primary-key"
	backupKey := "test-backup-key"

	// Counter to track API calls
	var primaryKeyCalled, backupKeyCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// Check which key is being used
		if strings.Contains(authHeader, primaryKey) {
			primaryKeyCalled = true
			// Primary key is rate limited
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"Rate limit exceeded","type":"rate_limit_error"}}`))
		} else if strings.Contains(authHeader, backupKey) {
			backupKeyCalled = true
			// Backup key works - return valid response that can be parsed by the real implementation
			w.WriteHeader(http.StatusOK)
			// Format matches what the real service expects
			_, _ = w.Write([]byte(`{
				"choices": [
					{
						"message": {
							"content": "{\"score\": 0.5, \"explanation\": \"Test\", \"confidence\": 0.8}"
						}
					}
				]
			}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	// Create real HTTPLLMService with our test server
	client := resty.New()
	service := NewHTTPLLMService(client, primaryKey, backupKey, server.URL)

	// Test article and prompt
	article := &db.Article{
		ID:      1,
		Content: testArticleContent,
	}

	promptVariant := PromptVariant{
		ID:       "default",
		Template: testPromptTemplate,
		Model:    testModelName,
	}

	// Call ScoreContent
	score, confidence, err := service.ScoreContent(context.Background(), promptVariant, article)

	// Verify both keys were tried and the backup succeeded
	assert.NoError(t, err, "Expected no error when fallback succeeds")
	assert.True(t, primaryKeyCalled, "Expected primary key to be called")
	assert.True(t, backupKeyCalled, "Expected backup key to be called")
	assert.Equal(t, 0.5, score, "Expected score of 0.5")
	assert.Equal(t, 0.8, confidence, "Expected confidence of 0.8")
}

// TestBothHTTPKeysRateLimited tests that HTTPLLMService returns appropriate error when both keys are rate limited
func TestBothHTTPKeysRateLimited(t *testing.T) {
	// Create test server that emulates rate limiting for both keys
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return rate limited error
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"Rate limit exceeded","type":"rate_limit_error"}}`))
	}))
	defer server.Close()

	// Create real HTTPLLMService with our test server
	client := resty.New()
	service := NewHTTPLLMService(client, "primary-key", "backup-key", server.URL)

	// Test article and prompt
	article := &db.Article{
		ID:      2,
		Content: testArticleContent,
	}

	promptVariant := PromptVariant{
		ID:       "default",
		Template: testPromptTemplate,
		Model:    testModelName,
	}

	// Call ScoreContent
	_, _, err := service.ScoreContent(context.Background(), promptVariant, article)

	// Should get rate limit error
	assert.Error(t, err, "Expected error when both keys are rate-limited")
	assert.True(t, strings.Contains(err.Error(), "rate limit"), "Expected rate limit error message")
}

// TestLLMClientRateLimitFallback tests the LLMClient's ScoreWithModel method with rate limiting fallback
func TestLLMClientRateLimitFallback(t *testing.T) {
	// Create a mock service for the LLMClient
	mockService := newMockLLMService("backup-key", true, false)

	// Create LLMClient with mock service
	client := &LLMClient{
		llmService: mockService,
	}

	// Test article
	article := &db.Article{
		ID:      5,
		Content: testArticleContent,
	}

	// Call ScoreWithModel
	score, err := client.ScoreWithModel(article, testModelName)

	// Verify result
	assert.NoError(t, err, "Expected no error when LLMClient fallback succeeds")
	assert.Equal(t, 0.5, score, "Expected score from backup key")
}

// TestLLMClientBothKeysRateLimited tests the LLMClient with both keys rate limited
func TestLLMClientBothKeysRateLimited(t *testing.T) {
	// Create a mock service where both keys fail
	mockService := newMockLLMService("backup-key", true, true)

	// Create LLMClient with mock service
	client := &LLMClient{
		llmService: mockService,
	}

	// Test article
	article := &db.Article{
		ID:      6,
		Content: testArticleContent,
	}

	// Call ScoreWithModel
	_, err := client.ScoreWithModel(article, testModelName)

	// Verify error
	assert.Error(t, err, "Expected error when both keys are rate-limited")
	assert.Equal(t, ErrBothLLMKeysRateLimited, err, "Expected error to be ErrBothLLMKeysRateLimited")
}
