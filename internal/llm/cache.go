package llm

import "sync"

type Cache struct {
	m sync.Map
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) Get(k string) (string, bool) {
	v, ok := c.m.Load(k)
	if !ok {
		return "", false
	}
	return v.(string), true
}

func (c *Cache) Set(k, v string) {
	c.m.Store(k, v)
}
