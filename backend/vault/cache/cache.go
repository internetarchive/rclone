// Package cache implements a minimal volatile cache.
package cache

import (
	"encoding/json"
	"fmt"
	"sync"
)

// New sets up a basic cache using a map.
func New() *Cache {
	return &Cache{
		m: make(map[string]interface{}),
		groupKeyFunc: func(k, g string) string {
			return fmt.Sprintf("%s-%s", k, g)
		},
	}
}

// Cache is a generic thread safe cache for local use.
type Cache struct {
	groupKeyFunc func(k, g string) string
	mu           sync.Mutex
	m            map[string]interface{}
}

// Reset clears the cache.
func (c *Cache) Reset() {
	c.mu.Lock()
	c.m = make(map[string]interface{})
	c.mu.Unlock()
}

// SetGroup set a key within a group.
func (c *Cache) SetGroup(k, group string, v interface{}) {
	c.Set(c.groupKeyFunc(k, group), v)
}

// GetGroup gets the value for a key within a group.
func (c *Cache) GetGroup(k, group string) interface{} {
	return c.Get(c.groupKeyFunc(k, group))
}

// Set value for a key.
func (c *Cache) Set(k string, v interface{}) {
	c.mu.Lock()
	c.m[k] = v
	c.mu.Unlock()
}

// Get value for a key.
func (c *Cache) Get(k string) interface{} {
	c.mu.Lock()
	result := c.m[k]
	c.mu.Unlock()
	return result
}

// Atos stringifies a value. Panics if the value cannot be marshalled.
func Atos(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("atos: %v", v))
	}
	return string(b)
}
