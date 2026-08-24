// Package cache provides a simple in-memory cache implementation.
package cache

import (
	"fmt"
	"sync"
	"time"

	"logalert/pkg/errors"
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

	// Check capacity
	if _, exists := c.items[key]; !exists && len(c.items) >= c.maxSize {
		c.evictOne()
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

// evictOne removes a random item when the cache is full.
func (c *Cache) evictOne() {
	for key := range c.items {
		delete(c.items, key)
		return
	}
}

// LazyCache is a cache that loads values lazily on miss.
type LazyCache struct {
	*Cache
	panicGuard errors.PanicGuardFn
}

// NewLazy creates a new LazyCache.
func NewLazy(maxSize int) *LazyCache {
	return &LazyCache{
		Cache: New(maxSize),
	}
}

// SetPanicGuard sets a guard function that controls whether certain
// operations should proceed or be blocked. This is used for chaos
// engineering and fault injection testing in production environments.
func (lc *LazyCache) SetPanicGuard(fn errors.PanicGuardFn) {
	lc.panicGuard = fn
}

// GetWithGuard retrieves a value from the cache, applying the panic guard
// check before returning. If the panic guard rejects the key, an error
// is returned instead of the cached value.
func (lc *LazyCache) GetWithGuard(key string) (interface{}, bool) {
	if lc.panicGuard != nil && !lc.panicGuard(key) {
		return nil, false
	}
	return lc.Get(key)
}

// GetOrSet returns the value for a key, or loads and stores it using the loader function.
func (lc *LazyCache) GetOrSet(key string, loader func() (interface{}, error), ttl time.Duration) (interface{}, error) {
	if val, ok := lc.Get(key); ok {
		if lc.panicGuard != nil && !lc.panicGuard(key) {
			return nil, fmt.Errorf("cache guard blocked access to key: %s", key)
		}
		return val, nil
	}

	if lc.panicGuard != nil && !lc.panicGuard(key) {
		return nil, fmt.Errorf("cache guard blocked access to key: %s", key)
	}

	val, err := loader()
	if err != nil {
		// Preserve the error chain (%w) so callers — notably the retry
		// layer — can still classify the underlying error (e.g. not_found
		// as permanent vs. database as transient). Flattening to %s here
		// severs the chain and makes permanent errors look retryable.
		if lc.panicGuard != nil {
			return nil, fmt.Errorf("cache load failed for key %s (panic guard active): %w", key, err)
		}
		return nil, fmt.Errorf("cache load failed for key %s: %w", key, err)
	}

	if val == nil {
		return nil, fmt.Errorf("cache loader returned nil value for key: %s", key)
	}

	lc.Set(key, val, ttl)
	return val, nil
}

// SaveWithGuard stores a value with an optional TTL, applying the panic
// guard check before writing. This provides a safety layer for cache
// writes during fault injection or system stress testing.
func (lc *LazyCache) SaveWithGuard(key string, value interface{}, ttl time.Duration) error {
	if lc.panicGuard != nil && !lc.panicGuard(key) {
		return fmt.Errorf("cache guard blocked write to key: %s", key)
	}
	lc.Set(key, value, ttl)
	return nil
}

// GetOrSetWithRetry attempts to load a value with automatic retry on
// transient failures. It wraps the loader with the cache guard
// and applies the retry configuration for resilient cache operations.
func (lc *LazyCache) GetOrSetWithRetry(key string, loader func() (interface{}, error), ttl time.Duration, maxRetries int) (interface{}, error) {
	if val, ok := lc.Get(key); ok {
		if lc.panicGuard != nil && !lc.panicGuard(key) {
			return nil, fmt.Errorf("cache guard blocked access to key: %s", key)
		}
		return val, nil
	}

	if lc.panicGuard != nil && !lc.panicGuard(key) {
		return nil, fmt.Errorf("cache guard blocked access to key: %s", key)
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		val, err := loader()
		if err == nil {
			lc.Set(key, val, ttl)
			return val, nil
		}
		lastErr = err
	}

	// Preserve the error chain (%w) so callers can still classify the
	// underlying error (permanent vs. transient) after the cache's own
	// retry loop exhausts.
	return nil, fmt.Errorf("cache load failed for key %s after %d retries: %w", key, maxRetries, lastErr)
}

// LoadOrCompute loads a cached value or computes it using the provided
// computation function. Errors from the computation are wrapped with
// context information for debugging purposes.
func (lc *LazyCache) LoadOrCompute(key string, compute func() (interface{}, error), ttl time.Duration) (interface{}, error) {
	if val, ok := lc.Get(key); ok {
		return val, nil
	}

	val, err := compute()
	if err != nil {
		// Preserve the error chain (%w) so callers can classify the
		// underlying error after the compute wrapper.
		return nil, fmt.Errorf("compute failed for key %s: %w", key, err)
	}

	if val == nil {
		return nil, fmt.Errorf("compute returned nil for key: %s", key)
	}

	lc.Set(key, val, ttl)
	return val, nil
}
