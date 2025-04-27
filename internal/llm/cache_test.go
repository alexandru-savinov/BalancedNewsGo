package llm

import (
	"testing"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestNewCache(t *testing.T) {
	cache := NewCache()
	assert.NotNil(t, cache)
}

func TestMakeKey(t *testing.T) {
	cases := []struct {
		name     string
		hash     string
		model    string
		expected string
	}{
		{
			name:     "Simple key",
			hash:     "abc123",
			model:    "gpt-4",
			expected: "abc123:gpt-4",
		},
		{
			name:     "Empty hash",
			hash:     "",
			model:    "gpt-4",
			expected: ":gpt-4",
		},
		{
			name:     "Empty model",
			hash:     "abc123",
			model:    "",
			expected: "abc123:",
		},
		{
			name:     "Both empty",
			hash:     "",
			model:    "",
			expected: ":",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := makeKey(c.hash, c.model)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestCacheGetSet(t *testing.T) {
	cache := NewCache()

	// Create test score
	score1 := &db.LLMScore{
		ArticleID: 1,
		Model:     "gpt-4",
		Score:     0.75,
		Metadata:  `{"text":"test metadata"}`,
	}

	// Test setting and getting a value
	cache.Set("hash1", "gpt-4", score1)
	val, ok := cache.Get("hash1", "gpt-4")
	assert.True(t, ok)
	assert.Equal(t, score1.Score, val.Score)
	assert.Equal(t, score1.Metadata, val.Metadata)

	// Test getting a non-existent key
	val, ok = cache.Get("non-existent", "gpt-4")
	assert.False(t, ok)
	assert.Nil(t, val)

	// Test overwriting a value
	score2 := &db.LLMScore{
		ArticleID: 1,
		Model:     "gpt-4",
		Score:     0.85,
		Metadata:  `{"text":"updated metadata"}`,
	}
	cache.Set("hash1", "gpt-4", score2)
	val, ok = cache.Get("hash1", "gpt-4")
	assert.True(t, ok)
	assert.Equal(t, score2.Score, val.Score)
	assert.Equal(t, score2.Metadata, val.Metadata)
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache()

	// Create test scores
	score1 := &db.LLMScore{
		ArticleID: 1,
		Model:     "gpt-4",
		Score:     0.75,
	}

	score2 := &db.LLMScore{
		ArticleID: 2,
		Model:     "gpt-4",
		Score:     0.25,
	}

	// Add some items
	cache.Set("hash1", "gpt-4", score1)
	cache.Set("hash2", "gpt-4", score2)

	// Delete using direct key
	cache.Delete(makeKey("hash1", "gpt-4"))

	// Check it's gone
	_, ok := cache.Get("hash1", "gpt-4")
	assert.False(t, ok)

	// Other key should still be there
	val, ok := cache.Get("hash2", "gpt-4")
	assert.True(t, ok)
	assert.Equal(t, score2.Score, val.Score)
}

func TestCacheRemove(t *testing.T) {
	cache := NewCache()

	// Create test scores
	score1 := &db.LLMScore{
		ArticleID: 1,
		Model:     "gpt-4",
		Score:     0.75,
	}

	score2 := &db.LLMScore{
		ArticleID: 2,
		Model:     "llama",
		Score:     0.25,
	}

	// Add some items
	cache.Set("hash1", "gpt-4", score1)
	cache.Set("hash1", "llama", score2)

	// Remove specific model for hash1
	cache.Remove("hash1", "gpt-4")

	// Check specific model is gone
	_, ok := cache.Get("hash1", "gpt-4")
	assert.False(t, ok)

	// Other model for same hash should still be there
	val, ok := cache.Get("hash1", "llama")
	assert.True(t, ok)
	assert.Equal(t, score2.Score, val.Score)
}
