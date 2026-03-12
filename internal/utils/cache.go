package utils

import "sync"

type SyncCache[K comparable, V any] struct {
	mu   sync.RWMutex
	once sync.Once
	data map[K]V
	err  error
}

func (c *SyncCache[K, V]) Load(fn func() (map[K]V, error)) error {
	c.once.Do(func() {
		data, err := fn()
		c.mu.Lock()
		defer c.mu.Unlock()
		c.data, c.err = data, err
	})
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

func (c *SyncCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	return v, ok
}

func (c *SyncCache[K, V]) All() map[K]V {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

func (c *SyncCache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}
