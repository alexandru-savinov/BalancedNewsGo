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
	if err := json.Unmarshal([]byte(v.(string)), &score); err != nil {
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
