package api

import (
	"sync"
	"time"
)

// Cache implementation
type SimpleCache struct {
	cache map[string]cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

func NewSimpleCache() *SimpleCache {
	return &SimpleCache{
		cache: make(map[string]cacheEntry),
	}
}

func (c *SimpleCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiration) {
		delete(c.cache, key)
		return nil, false
	}

	return entry.value, true
}

func (c *SimpleCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

func (c *SimpleCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
}
