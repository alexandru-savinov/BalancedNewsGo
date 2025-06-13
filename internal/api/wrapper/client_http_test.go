package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIClient_HTTPIntegration tests the API client with a mock HTTP server
func TestAPIClient_HTTPIntegration(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/articles":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			articles := []map[string]interface{}{
				{
					"article_id": 1,
					"Title":      "Test Article 1",
					"Content":    "Test content 1",
				},
				{
					"article_id": 2,
					"Title":      "Test Article 2",
					"Content":    "Test content 2",
				},
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": articles,
			})
		case "/api/articles/1":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			article := map[string]interface{}{
				"article_id": 1,
				"Title":      "Test Article 1",
				"Content":    "Test content 1",
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": article,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client pointing to test server
	client := NewAPIClient(server.URL)

	t.Run("GetArticles Integration", func(t *testing.T) {
		ctx := context.Background()
		params := ArticlesParams{Limit: 10}

		articles, err := client.GetArticles(ctx, params)
		require.NoError(t, err)
		assert.Len(t, articles, 2)
		assert.Equal(t, "Test Article 1", articles[0].Title)
	})

	t.Run("GetArticle Integration", func(t *testing.T) {
		ctx := context.Background()

		article, err := client.GetArticle(ctx, 1)
		require.NoError(t, err)
		assert.NotNil(t, article)
		assert.Equal(t, "Test Article 1", article.Title)
	})
}

// TestAPIClient_RetryWithHTTP tests retry logic with HTTP failures
func TestAPIClient_RetryWithHTTP(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Fail first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Succeed on 3rd attempt
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		articles := []map[string]interface{}{
			{"article_id": 1, "Title": "Success After Retry"},
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": articles,
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, WithRetryConfig(5, 10*time.Millisecond))

	ctx := context.Background()
	params := ArticlesParams{Limit: 10}

	articles, err := client.GetArticles(ctx, params)
	require.NoError(t, err)
	assert.Len(t, articles, 1)
	assert.Equal(t, "Success After Retry", articles[0].Title)
	assert.Equal(t, 3, attemptCount, "Should have made 3 attempts")
}

// TestAPIClient_CachingWithHTTP tests caching behavior with HTTP responses
func TestAPIClient_CachingWithHTTP(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		articles := []map[string]interface{}{
			{"article_id": 1, "Title": fmt.Sprintf("Request %d", requestCount)},
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": articles,
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, WithCacheTTL(100*time.Millisecond))

	ctx := context.Background()
	params := ArticlesParams{Limit: 10}

	// First request should hit server
	articles1, err := client.GetArticles(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, "Request 1", articles1[0].Title)

	// Second request should hit cache
	articles2, err := client.GetArticles(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, "Request 1", articles2[0].Title) // Same as first request (cached)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third request should hit server again
	articles3, err := client.GetArticles(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, "Request 2", articles3[0].Title)

	assert.Equal(t, 2, requestCount, "Should have made exactly 2 HTTP requests")
}

// TestAPIClient_ConcurrencyWithHTTP tests concurrent requests with HTTP server
func TestAPIClient_ConcurrencyWithHTTP(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		reqNum := requestCount
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		articles := []map[string]interface{}{
			{"article_id": reqNum, "Title": fmt.Sprintf("Concurrent Request %d", reqNum)},
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": articles,
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)

	const numGoroutines = 5
	var wg sync.WaitGroup
	results := make([][]Article, numGoroutines)
	errors := make([]error, numGoroutines)

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			params := ArticlesParams{Limit: 10, Offset: index}

			articles, err := client.GetArticles(ctx, params)
			results[index] = articles
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify all requests succeeded
	for i := 0; i < numGoroutines; i++ {
		require.NoError(t, errors[i], "Request %d should not have failed", i)
		assert.Len(t, results[i], 1, "Request %d should return 1 article", i)
	}

	assert.Equal(t, numGoroutines, requestCount, "Should have made %d HTTP requests", numGoroutines)
}

// BenchmarkAPIClient_HTTPPerformance benchmarks API client performance with HTTP
func BenchmarkAPIClient_HTTPPerformance(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		articles := []map[string]interface{}{
			{"article_id": 1, "Title": "Benchmark Article"},
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": articles,
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL)

	b.Run("GetArticles", func(b *testing.B) {
		ctx := context.Background()
		params := ArticlesParams{Limit: 10}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.GetArticles(ctx, params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetArticles Cached", func(b *testing.B) {
		client := NewAPIClient(server.URL, WithCacheTTL(1*time.Hour)) // Long cache
		ctx := context.Background()
		params := ArticlesParams{Limit: 10}

		// Prime the cache
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
