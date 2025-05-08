package llm

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/alexandru-savinov/BalancedNewsGo/internal/db"
)

// Cache provides a thread-safe in-memory cache
type Cache struct {
	m sync.Map
}

// NewCache creates a new empty cache instance
func NewCache() *Cache {
	return &Cache{}
}

// makeKey creates a composite key from content hash and model
func makeKey(contentHash, model string) string {
	return fmt.Sprintf("%s:%s", contentHash, model)
}

// Get retrieves a value from the cache
func (c *Cache) Get(contentHash, model string) (*db.LLMScore, bool) {
	v, ok := c.m.Load(makeKey(contentHash, model))
	if !ok {
		return nil, false
	}

	// Convert stored JSON string back to LLMScore
	var score db.LLMScore
	s, okAssert := v.(string)
	if !okAssert {
		// Optionally log an error here, e.g., log.Printf("Cache item for key %s was not a string", makeKey(contentHash, model))
		return nil, false
	}
	if err := json.Unmarshal([]byte(s), &score); err != nil {
		// Optionally log an error here, e.g., log.Printf("Failed to unmarshal cache item for key %s: %v", makeKey(contentHash, model), err)
		return nil, false
	}
	return &score, true
}

// Set stores a value in the cache
func (c *Cache) Set(contentHash, model string, score *db.LLMScore) {
	// Convert LLMScore to JSON string for storage
	data, err := json.Marshal(score)
	if err != nil {
		return
	}
	c.m.Store(makeKey(contentHash, model), string(data))
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.m.Delete(key)
}

// Remove removes a value from the cache by content hash and model
func (c *Cache) Remove(contentHash, model string) {
	c.m.Delete(makeKey(contentHash, model))
}
