// Package cache implements a minimal in-memory cache.
package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const defaultCacheTTL = 5 * time.Minute

// New sets up a basic cache using a map.
func New() *Cache {
	return &Cache{
		m:   make(map[string]entry),
		ttl: defaultCacheTTL,
		groupKeyFunc: func(k, g string) string {
			return fmt.Sprintf("%s-%s", k, g)
		},
	}
}

// entry wraps the value with an expiration time
type entry struct {
	val    any
	expiry time.Time
}

type Cache struct {
	groupKeyFunc func(k, g string) string
	mu           sync.Mutex
	m            map[string]entry
	ttl          time.Duration
}

// Reset clears the cache.
func (c *Cache) Reset() {
	c.mu.Lock()
	c.m = make(map[string]entry)
	c.mu.Unlock()
}

// SetGroup set a key within a group.
func (c *Cache) SetGroup(k, group string, v any) {
	c.Set(c.groupKeyFunc(k, group), v)
}

// GetGroup gets the value for a key within a group.
func (c *Cache) GetGroup(k, group string) any {
	return c.Get(c.groupKeyFunc(k, group))
}

// Set value for a key with default TTL
func (c *Cache) Set(k string, v any) {
	c.mu.Lock()
	c.m[k] = entry{
		val:    v,
		expiry: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Get value for a key, returning nil if expired or not found
func (c *Cache) Get(k string) any {
	c.mu.Lock()
	e, ok := c.m[k]
	defer c.mu.Unlock()

	if !ok {
		return nil
	}
	if time.Now().After(e.expiry) {
		c.Delete(k)
		return nil
	}
	return e.val
}

// Helper to clean up specific keys
func (c *Cache) Delete(k string) {
	c.mu.Lock()
	delete(c.m, k)
	c.mu.Unlock()
}

// Atos stringifies a value. Panics if the value cannot be marshalled.
func Atos(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("atos: %v", v))
	}
	return string(b)
}
