package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockArticlesResponse represents a mock API response for articles
// type MockArticlesResponse struct { // This struct is not strictly needed if handlers return []MockArticle directly
// 	Articles []MockArticle `json:"articles"`
// 	Total    int           `json:"total"`
// }

// MockArticle now mirrors the fields and JSON tags of rawclient.Article
// from d:\\\\Dev\\\\NBG\\\\internal\\\\api\\\\client\\\\models.go
type MockArticle struct {
	ArticleID      int64     `json:"article_id,omitempty"`     // This one is snake_case in rawclient, keep as is.
	Title          string    `json:"Title,omitempty"`          // Corrected to CamelCase
	Content        string    `json:"Content,omitempty"`        // Corrected to CamelCase
	URL            string    `json:"URL,omitempty"`            // Corrected to CamelCase
	Source         string    `json:"Source,omitempty"`         // Corrected to CamelCase
	PubDate        time.Time `json:"PubDate,omitempty"`        // Corrected to CamelCase
	CreatedAt      time.Time `json:"CreatedAt,omitempty"`      // Corrected to CamelCase
	CompositeScore float64   `json:"CompositeScore,omitempty"` // Corrected to CamelCase
	Confidence     float64   `json:"Confidence,omitempty"`     // Corrected to CamelCase
	ScoreSource    string    `json:"ScoreSource,omitempty"`    // Corrected to CamelCase
	BiasLabel      string    `json:"BiasLabel,omitempty"`      // Corrected to CamelCase
	AnalysisNotes  string    `json:"AnalysisNotes,omitempty"`  // Corrected to CamelCase
}

// TestAPIClientIntegration tests the API client with a mock server
func TestAPIClientIntegration(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/articles":
			// Simulate a delay if the source is specifically for the context cancellation test
			if r.URL.Query().Get("source") == "context_cancel_test_with_delay" {
				time.Sleep(200 * time.Millisecond) // Delay for context cancellation test
			}
			handleMockArticles(w, r)
		case "/api/articles/1": // Corrected path from /api/article/1
			handleMockArticle(w, r)
		default:
			// Log unexpected paths to help debug 500 errors
			t.Logf("Mock server received request for unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create API client pointing to mock server
	client := NewAPIClient(server.URL)

	t.Run("GetArticles with caching", func(t *testing.T) {
		ctx := context.Background()
		params := ArticlesParams{
			Source:  "test-source",
			Leaning: "left",
			Limit:   10,
			Offset:  0,
		}

		// First call - should hit the API
		articles1, err := client.GetArticles(ctx, params)
		if err != nil && strings.Contains(err.Error(), "500") {
			t.Logf("Mock server likely returned 500 for /api/articles. Params: %+v. Error: %v", params, err)
		}
		require.NoError(t, err, "GetArticles failed on first call")
		require.Len(t, articles1, 2, "Expected 2 articles on first call")

		// Assertions for Article 1
		assert.Equal(t, int64(1), articles1[0].ArticleID)
		assert.Equal(t, "Test Article 1", articles1[0].Title)
		assert.Equal(t, "This is test content for article 1.", articles1[0].Content)
		assert.Equal(t, "http://example.com/1", articles1[0].URL)
		assert.Equal(t, "test-source", articles1[0].Source)
		assert.WithinDuration(t, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), articles1[0].PubDate, time.Second)
		assert.WithinDuration(t, time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC), articles1[0].CreatedAt, time.Second)
		assert.Equal(t, 0.75, articles1[0].CompositeScore)
		assert.Equal(t, 0.9, articles1[0].Confidence)
		assert.Equal(t, "llm-model-x", articles1[0].ScoreSource)
		assert.Equal(t, "neutral", articles1[0].BiasLabel)
		assert.Equal(t, "Initial analysis complete.", articles1[0].AnalysisNotes)

		// Assertions for Article 2
		assert.Equal(t, int64(2), articles1[1].ArticleID)
		assert.Equal(t, "Test Article 2", articles1[1].Title)
		assert.Equal(t, "This is test content for article 2.", articles1[1].Content)
		assert.Equal(t, "http://example.com/2", articles1[1].URL)
		assert.Equal(t, "test-source-2", articles1[1].Source)
		assert.WithinDuration(t, time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC), articles1[1].PubDate, time.Second)
		assert.WithinDuration(t, time.Date(2024, 1, 2, 12, 5, 0, 0, time.UTC), articles1[1].CreatedAt, time.Second)
		assert.Equal(t, 0.85, articles1[1].CompositeScore)
		assert.Equal(t, 0.95, articles1[1].Confidence)
		assert.Equal(t, "llm-model-y", articles1[1].ScoreSource)
		assert.Equal(t, "left", articles1[1].BiasLabel)
		assert.Equal(t, "Further analysis needed.", articles1[1].AnalysisNotes)

		// Second call with same params - should hit cache
		articles2, err := client.GetArticles(ctx, params)
		require.NoError(t, err, "GetArticles failed on second call (cache hit)")
		require.Len(t, articles2, 2, "Expected 2 articles on cache hit")
		assert.Equal(t, articles1, articles2, "Cached articles should be identical to first call")
	})
	t.Run("GetArticle by ID", func(t *testing.T) {
		ctx := context.Background()

		article, err := client.GetArticle(ctx, 1)
		if err != nil && strings.Contains(err.Error(), "500") {
			t.Logf("Mock server likely returned 500 for /api/articles/1. Error: %v", err) // Corrected path in log
		}
		require.NoError(t, err, "GetArticle by ID failed")
		require.NotNil(t, article, "Article should not be nil")
		assert.Equal(t, int64(1), article.ArticleID)
		assert.Equal(t, "Test Article 1", article.Title)
		assert.Equal(t, "This is test content for article 1.", article.Content)
		assert.Equal(t, "http://example.com/1", article.URL)
		assert.Equal(t, "test-source", article.Source)
		assert.WithinDuration(t, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), article.PubDate, time.Second)
		assert.WithinDuration(t, time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC), article.CreatedAt, time.Second)
		assert.Equal(t, 0.75, article.CompositeScore)
		assert.Equal(t, 0.9, article.Confidence)
		assert.Equal(t, "llm-model-x", article.ScoreSource)
		assert.Equal(t, "neutral", article.BiasLabel)
		assert.Equal(t, "Initial analysis complete.", article.AnalysisNotes)
	})

	// Context cancellation test needs to be after other tests that might use the same base URL
	// or ensure its parameters are unique enough not to interfere with the delay mechanism.
	// The current setup with a unique source query param for delay is good.
	// The clientForCancellationTest is also separate.
	// It seems there was a copy-paste error in the previous context, where a GetArticles call was inside the GetArticle by ID test.
	// Removing that misplaced call.
	// The original context cancellation test is preserved below.

	// Context cancellation test (original, seems correct)
	t.Run("GetArticles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		params := ArticlesParams{
			Source:  "context_cancel_test_with_delay",
			Leaning: "left",
			Limit:   10,
			Offset:  0,
		}

		// This call should be delayed and then fail due to context cancellation
		_, err := client.GetArticles(ctx, params)

		require.Error(t, err, "Expected an error due to context cancellation/timeout")
		errMsg := err.Error()
		// The client's DefaultErrorHandler translates context deadline issues (which result in a client-side timeout)
		// into "API Error [408/timeout]: Request timed out". This is the expected behavior.
		assert.Contains(t, errMsg, "API Error [408/timeout]: Request timed out", "Expected a 408 timeout error from the client, got: %s", errMsg)
	})

	t.Run("API error handling", func(t *testing.T) {
		// Create a failing mock server
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}))
		defer failServer.Close()

		failClient := NewAPIClient(failServer.URL)

		ctx := context.Background()
		params := ArticlesParams{Limit: 10}

		_, err := failClient.GetArticles(ctx, params)
		assert.Error(t, err)
	})
}

// TestCacheInvalidation tests that cache invalidation works correctly
func TestCacheInvalidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleMockArticles(w, r)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, WithCacheTTL(100*time.Millisecond)) // Short TTL for testing
	ctx := context.Background()

	params := ArticlesParams{Limit: 10, Source: "cache_invalidation_test"}
	// First, populate cache
	_, err := client.GetArticles(ctx, params)
	if err != nil && strings.Contains(err.Error(), "500") {
		t.Logf("Mock server likely returned 500 for /api/articles during cache population. Params: %+v", params)
	}
	require.NoError(t, err, "Failed to populate cache")

	cacheKey := buildCacheKey("articles", params.Source, params.Leaning, params.Limit, params.Offset)
	_, found := client.getCached(cacheKey)
	assert.True(t, found, "Value should be in cache after first call")

	time.Sleep(150 * time.Millisecond) // Wait for cache to expire

	_, found = client.getCached(cacheKey)
	assert.False(t, found, "Value should be gone from cache after TTL expiry")

	// Call again, should hit API (and repopulate cache)
	_, err = client.GetArticles(ctx, params)
	if err != nil && strings.Contains(err.Error(), "500") {
		t.Logf("Mock server likely returned 500 for /api/articles after cache expiry. Params: %+v", params)
	}
	require.NoError(t, err, "Failed to get articles after cache expiry")
	_, found = client.getCached(cacheKey)
	assert.True(t, found, "Value should be back in cache after API call post-expiry")
}

// TestRetryLogic tests that retry logic works correctly
func TestRetryLogic(t *testing.T) {
	var requestCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			// Fail first 2 requests
			http.Error(w, "Temporary Error", http.StatusServiceUnavailable)
			return
		}
		// Succeed on 3rd request
		handleMockArticles(w, r)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, WithRetryConfig(3, 20*time.Millisecond)) // Adjusted retry delay
	ctx := context.Background()

	params := ArticlesParams{Limit: 10, Source: "retry_logic_test"}
	articles, err := client.GetArticles(ctx, params)
	if err != nil && strings.Contains(err.Error(), "500") {
		t.Logf("Mock server likely returned 500 for /api/articles during retry test. Params: %+v. Request count: %d", params, requestCount)
	}

	require.NoError(t, err, "GetArticles with retry failed")
	require.Len(t, articles, 2)
	assert.Equal(t, 3, requestCount) // Should have made 3 requests
}

// TestConcurrentAccess tests that the client is safe for concurrent use
func TestConcurrentAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add small delay to simulate real API
		time.Sleep(10 * time.Millisecond)
		handleMockArticles(w, r)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)

	const numGoroutines = 10
	errs := make(chan error, numGoroutines)
	var wg sync.WaitGroup // Use WaitGroup for better synchronization
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			params := ArticlesParams{
				Limit:  10,
				Offset: id * 10,
				Source: fmt.Sprintf("concurrent_test_%d", id),
			}

			_, err := client.GetArticles(ctx, params)
			if err != nil && strings.Contains(err.Error(), "500") {
				t.Logf("Mock server likely returned 500 for /api/articles during concurrent test. Goroutine ID: %d, Params: %+v", id, params)
			}
			errs <- err
		}(i)
	}

	go func() { // Goroutine to close channel once all workers are done
		wg.Wait()
		close(errs)
	}()

	// Wait for all goroutines to complete and check errors
	for err := range errs {
		assert.NoError(t, err, "Concurrent GetArticles call failed")
	}
}

// Helper functions for mock server responses

func handleMockArticles(w http.ResponseWriter, r *http.Request) {
	articles := []MockArticle{
		{
			ArticleID:      1,
			Title:          "Test Article 1",
			Content:        "This is test content for article 1.",
			URL:            "http://example.com/1",
			Source:         "test-source",
			PubDate:        time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			CreatedAt:      time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC),
			CompositeScore: 0.75,
			Confidence:     0.9,
			ScoreSource:    "llm-model-x",
			BiasLabel:      "neutral",
			AnalysisNotes:  "Initial analysis complete.",
		},
		{
			ArticleID:      2,
			Title:          "Test Article 2",
			Content:        "This is test content for article 2.",
			URL:            "http://example.com/2",
			Source:         "test-source-2",
			PubDate:        time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			CreatedAt:      time.Date(2024, 1, 2, 12, 5, 0, 0, time.UTC),
			CompositeScore: 0.85,
			Confidence:     0.95,
			ScoreSource:    "llm-model-y",
			BiasLabel:      "left",
			AnalysisNotes:  "Further analysis needed.",
		},
	}
	// Wrap articles in a structure matching rawclient.StandardResponse
	mockResponse := struct {
		Success bool          `json:"success"`
		Data    []MockArticle `json:"data"`
	}{
		Success: true,
		Data:    articles,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
		http.Error(w, "Failed to encode mock articles response", http.StatusInternalServerError)
	}
}

func handleMockArticle(w http.ResponseWriter, r *http.Request) {
	article := MockArticle{
		ArticleID:      1,
		Title:          "Test Article 1",
		Content:        "This is test content for article 1.",
		URL:            "http://example.com/1",
		Source:         "test-source",
		PubDate:        time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		CreatedAt:      time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC),
		CompositeScore: 0.75,
		Confidence:     0.9,
		ScoreSource:    "llm-model-x",
		BiasLabel:      "neutral",
		AnalysisNotes:  "Initial analysis complete.",
	}
	// Wrap article in a structure matching rawclient.StandardResponse
	mockResponse := struct {
		Success bool        `json:"success"`
		Data    MockArticle `json:"data"`
	}{
		Success: true,
		Data:    article,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
		http.Error(w, "Failed to encode mock article response", http.StatusInternalServerError)
	}
}

// BenchmarkAPIClient provides performance benchmarks
func BenchmarkAPIClient(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleMockArticles(w, r)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)
	ctx := context.Background()
	params := ArticlesParams{Limit: 10}

	b.Run("GetArticles", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.GetArticles(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetArticlesCached", func(b *testing.B) {
		// Warm up cache
		_, _ = client.GetArticles(ctx, params)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.GetArticles(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
