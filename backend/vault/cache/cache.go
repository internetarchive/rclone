package cache

import (
	"encoding/json"
	"fmt"
	"sync"
)

// New sets up a new in-memory cache.
func New() *Cache {
	return &Cache{
		m: make(map[string]interface{}),
		groupKeyFunc: func(k, g string) string {
			return fmt.Sprint("%s-%s", k, g)
		},
	}
}

// Cache is a generic thread safe cache for local use.
type Cache struct {
	groupKeyFunc func(k, g string) string
	mu           sync.Mutex
	m            map[string]interface{}
}

func (c *Cache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[string]interface{})
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
	defer c.mu.Unlock()
	c.m[k] = v
}

// Get value for a key.
func (c *Cache) Get(k string) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.m[k]
}

// Atos stringifies a value.
func Atos(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("atos: %v", v))
	}
	return string(b)
}
