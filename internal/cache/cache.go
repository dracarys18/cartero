package cache

import (
	"log"
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

type Cache[K comparable, V any] struct {
	cache       *gocache.Cache
	mu          sync.RWMutex
	keyToString func(K) string
}

type CacheConfig struct {
	TTL time.Duration
}

func NewCache[K comparable, V any](config CacheConfig, keyToString func(K) string) *Cache[K, V] {
	if config.TTL == 0 {
		config.TTL = 1 * time.Hour
	}

	goCacheInstance := gocache.New(config.TTL, config.TTL/2)
	log.Printf("Cache: Initialized with TTL=%v", config.TTL)

	return &Cache[K, V]{
		cache:       goCacheInstance,
		keyToString: keyToString,
	}
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stringKey := c.keyToString(key)
	value, found := c.cache.Get(stringKey)
	if !found {
		var zero V
		return zero, false
	}

	if typedValue, ok := value.(V); ok {
		return typedValue, true
	}

	var zero V
	return zero, false
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stringKey := c.keyToString(key)
	c.cache.Set(stringKey, value, gocache.DefaultExpiration)
	log.Printf("Cache: Stored key=%s (type=%T)", stringKey, value)
}

func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stringKey := c.keyToString(key)
	c.cache.Set(stringKey, value, ttl)
	log.Printf("Cache: Stored key=%s (type=%T, ttl=%v)", stringKey, value, ttl)
}

func (c *Cache[K, V]) InvalidateKey(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stringKey := c.keyToString(key)
	c.cache.Delete(stringKey)
	log.Printf("Cache: Invalidated key=%s", stringKey)
}

func (c *Cache[K, V]) InvalidatePattern(patternPrefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	items := c.cache.Items()
	for key := range items {
		if len(key) >= len(patternPrefix) && key[:len(patternPrefix)] == patternPrefix {
			c.cache.Delete(key)
		}
	}
	log.Printf("Cache: Invalidated pattern=%s", patternPrefix)
}

func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Flush()
	log.Printf("Cache: Cleared all entries")
}

func (c *Cache[K, V]) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Flush()
	log.Printf("Cache: Shut down")
	return nil
}

func (c *Cache[K, V]) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	items := c.cache.Items()
	return map[string]interface{}{
		"size":  len(items),
		"items": len(items),
	}
}
