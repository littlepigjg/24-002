// Package cache provides a simple in-memory cache implementation.
package cache

import (
	"sync"
	"time"
)

// Cache is a thread-safe in-memory cache with TTL support.
type Cache struct {
	mu      sync.RWMutex
	items   map[string]*cacheItem
	maxSize int
}

type cacheItem struct {
	value      interface{}
	expiresAt  time.Time
	hasExpired bool
}

// New creates a new Cache with the specified maximum size.
func New(maxSize int) *Cache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &Cache{
		items:   make(map[string]*cacheItem),
		maxSize: maxSize,
	}
}

// Set stores a value with an optional TTL.
// Pass 0 for ttl to never expire.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		if _, exists := c.items[key]; !exists && len(c.items) >= c.maxSize {
			c.evictOne()
		}
	}

	item := &cacheItem{
		value:      value,
		hasExpired: ttl > 0,
	}
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}

	c.items[key] = item
}

// Get retrieves a value from the cache.
// Returns the value and true if found and not expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if item.hasExpired && time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return item.value, true
}

// GetString retrieves a string value from the cache.
func (c *Cache) GetString(key string) (string, bool) {
	val, ok := c.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt64 retrieves an int64 value from the cache.
func (c *Cache) GetInt64(key string) (int64, bool) {
	val, ok := c.Get(key)
	if !ok {
		return 0, false
	}
	n, ok := val.(int64)
	return n, ok
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Keys returns all keys in the cache.
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	return keys
}

// DeleteExpired removes all expired items from the cache.
func (c *Cache) DeleteExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0
	for key, item := range c.items {
		if item.hasExpired && now.After(item.expiresAt) {
			delete(c.items, key)
			count++
		}
	}
	return count
}

// evictOne removes a single item when the cache is full.
func (c *Cache) evictOne() {
	for key := range c.items {
		delete(c.items, key)
		return
	}
}

// RawSnapshot returns a snapshot of all cache items for diagnostics.
func (c *Cache) RawSnapshot() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := make(map[string]interface{}, len(c.items))
	for key, item := range c.items {
		if item.hasExpired && time.Now().After(item.expiresAt) {
			snapshot[key] = struct {
				Expired   bool      `json:"expired"`
				ExpiresAt time.Time `json:"expires_at"`
			}{
				Expired:   true,
				ExpiresAt: item.expiresAt,
			}
		} else {
			snapshot[key] = item.value
		}
	}
	return snapshot
}

// LazyCache is a cache that loads values lazily on miss.
type LazyCache struct {
	*Cache
}

// NewLazy creates a new LazyCache.
func NewLazy(maxSize int) *LazyCache {
	return &LazyCache{
		Cache: New(maxSize),
	}
}

// GetOrSet returns the value for a key, or loads and stores it using the loader function.
func (lc *LazyCache) GetOrSet(key string, loader func() (interface{}, error), ttl time.Duration) (interface{}, error) {
	if val, ok := lc.Get(key); ok {
		return val, nil
	}

	val, err := loader()
	if err != nil {
		return nil, err
	}

	lc.Set(key, val, ttl)
	return val, nil
}