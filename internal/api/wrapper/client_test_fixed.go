package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper to create a test client with mock HTTP server
func newTestClientWithMockServer(handler http.HandlerFunc) (*APIClient, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := NewAPIClient(server.URL)
	return client, server
}

func TestNewAPIClient(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		client := NewAPIClient("http://localhost:8080")

		assert.NotNil(t, client)
		assert.NotNil(t, client.cache)
		assert.Equal(t, "http://localhost:8080", client.cfg.BaseURL)
		assert.Equal(t, 30*time.Second, client.cfg.Timeout)
		assert.Equal(t, 30*time.Second, client.cfg.CacheTTL)
		assert.Equal(t, 3, client.cfg.MaxRetries)
		assert.Equal(t, time.Second, client.cfg.RetryDelay)
	})

	t.Run("Custom configuration", func(t *testing.T) {
		client := NewAPIClient("http://test:9000",
			WithTimeout(10*time.Second),
			WithCacheTTL(5*time.Minute),
			WithRetryConfig(5, 2*time.Second),
		)

		assert.Equal(t, "http://test:9000", client.cfg.BaseURL)
		assert.Equal(t, 10*time.Second, client.cfg.Timeout)
		assert.Equal(t, 5*time.Minute, client.cfg.CacheTTL)
		assert.Equal(t, 5, client.cfg.MaxRetries)
		assert.Equal(t, 2*time.Second, client.cfg.RetryDelay)
	})
}

func TestGetArticles(t *testing.T) {
	t.Run("Successful request", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/articles", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			// Check query parameters
			assert.Equal(t, "cnn", r.URL.Query().Get("source"))
			assert.Equal(t, "left", r.URL.Query().Get("leaning"))
			assert.Equal(t, "10", r.URL.Query().Get("limit"))
			assert.Equal(t, "0", r.URL.Query().Get("offset"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{
				"success": true,
				"data": [
					{
						"article_id": 123,
						"Title": "Test Article",
						"Content": "Test content",
						"Source": "cnn"
					}
				]
			}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		})

		client, server := newTestClientWithMockServer(handler)
		defer server.Close()

		ctx := context.Background()
		params := ArticlesParams{
			Source:  "cnn",
			Leaning: "left",
			Limit:   10,
			Offset:  0,
		}

		articles, err := client.GetArticles(ctx, params)
		require.NoError(t, err)
		assert.Len(t, articles, 1)
		assert.Equal(t, int64(123), articles[0].ArticleID)
		assert.Equal(t, "Test Article", articles[0].Title)
	})

	t.Run("API error response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte(`{
				"success": false,
				"error": {
					"code": "internal_error",
					"message": "Database connection failed"
				}
			}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		})

		client, server := newTestClientWithMockServer(handler)
		defer server.Close()

		ctx := context.Background()
		params := ArticlesParams{Limit: 10}

		_, err := client.GetArticles(ctx, params)
		assert.Error(t, err)
		apiErr, ok := err.(APIError)
		assert.True(t, ok)
		assert.Equal(t, 500, apiErr.StatusCode)
		assert.Contains(t, apiErr.Message, "Database connection failed")
	})

	t.Run("Network error with retries", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount < 3 {
				// Simulate network error
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			// Success on third try
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{
				"success": true,
				"data": [{"article_id": 1, "Title": "Success"}]
			}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		})

		client, server := newTestClientWithMockServer(handler)
		defer server.Close()

		// Set fast retry for testing
		client.cfg.RetryDelay = 10 * time.Millisecond

		ctx := context.Background()
		params := ArticlesParams{Limit: 1}

		articles, err := client.GetArticles(ctx, params)
		require.NoError(t, err)
		assert.Len(t, articles, 1)
		assert.Equal(t, 3, callCount) // Should have retried 2 times
	})
}

func TestGetArticle(t *testing.T) {
	t.Run("Successful request", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/articles/123", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{
				"success": true,
				"data": {
					"article_id": 123,
					"Title": "Single Article",
					"Content": "Article content",
					"Source": "test"
				}
			}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		})

		client, server := newTestClientWithMockServer(handler)
		defer server.Close()

		ctx := context.Background()
		article, err := client.GetArticle(ctx, 123)
		require.NoError(t, err)
		assert.Equal(t, int64(123), article.ArticleID)
		assert.Equal(t, "Single Article", article.Title)
	})

	t.Run("Article not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte(`{
				"success": false,
				"error": {
					"code": "not_found",
					"message": "Article not found"
				}
			}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		})

		client, server := newTestClientWithMockServer(handler)
		defer server.Close()

		ctx := context.Background()
		_, err := client.GetArticle(ctx, 999)
		assert.Error(t, err)
		apiErr, ok := err.(APIError)
		assert.True(t, ok)
		assert.Equal(t, 404, apiErr.StatusCode)
	})
}

func TestCaching(t *testing.T) {
	t.Run("Cache hit and miss", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{
				"success": true,
				"data": [{"article_id": 1, "Title": "Cached"}]
			}`)); err != nil {
				t.Errorf("Failed to write response: %v", err)
			}
		})

		client, server := newTestClientWithMockServer(handler)
		defer server.Close()

		ctx := context.Background()
		params := ArticlesParams{Limit: 1}

		// First call should hit the API
		articles1, err := client.GetArticles(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)

		// Second call should hit cache
		articles2, err := client.GetArticles(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount) // Should not increase
		assert.Equal(t, articles1[0].ArticleID, articles2[0].ArticleID)
	})

	t.Run("Cache expiration", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{
				"success": true,
				"data": [{"article_id": %d, "Title": "Call %d"}]
			}`, callCount, callCount)))
		})

		client, server := newTestClientWithMockServer(handler)
		defer server.Close()

		// Set very short cache TTL for testing
		client.cfg.CacheTTL = 50 * time.Millisecond

		ctx := context.Background()
		params := ArticlesParams{Limit: 1}

		// First call
		articles1, err := client.GetArticles(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int64(1), articles1[0].ArticleID)

		// Wait for expiration
		time.Sleep(100 * time.Millisecond)

		// Second call should hit API again
		articles2, err := client.GetArticles(ctx, params)
		require.NoError(t, err)
		assert.Equal(t, int64(2), articles2[0].ArticleID)
		assert.Equal(t, 2, callCount)
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("Concurrent requests", func(t *testing.T) {
		var callCount int
		var mu sync.Mutex

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			callCount++
			currentCall := callCount
			mu.Unlock()

			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{
				"success": true,
				"data": [{"article_id": %d, "Title": "Concurrent %d"}]
			}`, currentCall, currentCall)))
		})

		client, server := newTestClientWithMockServer(handler)
		defer server.Close()

		ctx := context.Background()
		const numGoroutines = 10
		results := make(chan []Article, numGoroutines)
		errors := make(chan error, numGoroutines)

		// Launch concurrent requests
		for i := 0; i < numGoroutines; i++ {
			go func(i int) {
				params := ArticlesParams{
					Limit:  1,
					Offset: i, // Different params to avoid cache hits
				}
				articles, err := client.GetArticles(ctx, params)
				if err != nil {
					errors <- err
				} else {
					results <- articles
				}
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numGoroutines; i++ {
			select {
			case articles := <-results:
				assert.Len(t, articles, 1)
				successCount++
			case err := <-errors:
				t.Errorf("Unexpected error: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("Test timeout")
			}
		}

		assert.Equal(t, numGoroutines, successCount)
		assert.Equal(t, numGoroutines, callCount)
	})
}

func TestContextCancellation(t *testing.T) {
	t.Run("Request canceled", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "data": []}`))
		})

		client, server := newTestClientWithMockServer(handler)
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		params := ArticlesParams{Limit: 1}
		_, err := client.GetArticles(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}

func TestErrorTranslation(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "Validation error",
			statusCode:     400,
			responseBody:   `{"success":false,"error":{"code":"validation_error","message":"Invalid parameters"}}`,
			expectedErrMsg: "Invalid parameters",
		},
		{
			name:           "Not found error",
			statusCode:     404,
			responseBody:   `{"success":false,"error":{"code":"not_found","message":"Resource not found"}}`,
			expectedErrMsg: "Resource not found",
		},
		{
			name:           "Server error",
			statusCode:     500,
			responseBody:   `{"success":false,"error":{"code":"internal_error","message":"Server error"}}`,
			expectedErrMsg: "Server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			})

			client, server := newTestClientWithMockServer(handler)
			defer server.Close()

			ctx := context.Background()
			_, err := client.GetArticles(ctx, ArticlesParams{Limit: 1})

			assert.Error(t, err)
			apiErr, ok := err.(APIError)
			assert.True(t, ok)
			assert.Equal(t, tt.statusCode, apiErr.StatusCode)
			assert.Contains(t, apiErr.Message, tt.expectedErrMsg)
		})
	}
}

func TestCacheKeyGeneration(t *testing.T) {
	tests := []struct {
		name     string
		params1  ArticlesParams
		params2  ArticlesParams
		sameCkey bool
	}{
		{
			name:     "Same parameters",
			params1:  ArticlesParams{Source: "cnn", Limit: 10},
			params2:  ArticlesParams{Source: "cnn", Limit: 10},
			sameCkey: true,
		},
		{
			name:     "Different source",
			params1:  ArticlesParams{Source: "cnn", Limit: 10},
			params2:  ArticlesParams{Source: "fox", Limit: 10},
			sameCkey: false,
		},
		{
			name:     "Different limit",
			params1:  ArticlesParams{Source: "cnn", Limit: 10},
			params2:  ArticlesParams{Source: "cnn", Limit: 20},
			sameCkey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := buildCacheKey("articles", tt.params1.Source, tt.params1.Leaning, tt.params1.Limit, tt.params1.Offset)
			key2 := buildCacheKey("articles", tt.params2.Source, tt.params2.Leaning, tt.params2.Limit, tt.params2.Offset)

			if tt.sameCkey {
				assert.Equal(t, key1, key2)
			} else {
				assert.NotEqual(t, key1, key2)
			}
		})
	}
}

// Benchmark tests
func BenchmarkGetArticles(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": true,
			"data": [
				{"article_id": 1, "Title": "Benchmark Article 1"},
				{"article_id": 2, "Title": "Benchmark Article 2"}
			]
		}`))
	})

	client, server := newTestClientWithMockServer(handler)
	defer server.Close()

	ctx := context.Background()
	params := ArticlesParams{Limit: 2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetArticles(ctx, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetArticlesWithCache(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": true,
			"data": [
				{"article_id": 1, "Title": "Cached Article 1"},
				{"article_id": 2, "Title": "Cached Article 2"}
			]
		}`))
	})

	client, server := newTestClientWithMockServer(handler)
	defer server.Close()

	ctx := context.Background()
	params := ArticlesParams{Limit: 2}

	// Prime the cache
	_, err := client.GetArticles(ctx, params)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetArticles(ctx, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}
