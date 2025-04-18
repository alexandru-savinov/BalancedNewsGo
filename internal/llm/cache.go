package llm

import "sync"

// Cache provides a thread-safe in-memory cache
type Cache struct {
	m sync.Map
}

// NewCache creates a new empty cache instance
func NewCache() *Cache {
	return &Cache{}
}

// Get retrieves a value from the cache
func (c *Cache) Get(k string) (string, bool) {
	v, ok := c.m.Load(k)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// Set stores a value in the cache
func (c *Cache) Set(k, v string) {
	c.m.Store(k, v)
}
