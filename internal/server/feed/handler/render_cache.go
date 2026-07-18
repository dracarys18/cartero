package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type cacheEntry struct {
	html []byte
	etag string
	exp  time.Time
}

type pageCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

func newPageCache(ttl time.Duration) *pageCache {
	return &pageCache{entries: make(map[string]cacheEntry), ttl: ttl}
}

func (c *pageCache) get(key string) (cacheEntry, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.exp) {
		return cacheEntry{}, false
	}
	return e, true
}

func (c *pageCache) set(key string, html []byte) cacheEntry {
	sum := sha256.Sum256(html)
	e := cacheEntry{
		html: html,
		etag: `"` + hex.EncodeToString(sum[:8]) + `"`,
		exp:  time.Now().Add(c.ttl),
	}

	c.mu.Lock()
	if len(c.entries) > 128 {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.exp) {
				delete(c.entries, k)
			}
		}
	}
	c.entries[key] = e
	c.mu.Unlock()

	return e
}
