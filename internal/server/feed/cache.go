package feed

import (
	"fmt"

	"cartero/internal/cache"
)

type CacheKey struct {
	Name string
	Type string
}

func NewCacheKey(name, feedType string) CacheKey {
	return CacheKey{
		Name: name,
		Type: feedType,
	}
}

func (k CacheKey) ToString() string {
	return fmt.Sprintf("%s:%s", k.Name, k.Type)
}

func NewCache(config cache.CacheConfig) *cache.Cache[CacheKey, string] {
	return cache.NewCache[CacheKey, string](config, func(k CacheKey) string {
		return k.ToString()
	})
}

const TypeRSS = "rss"
const TypeAtom = "atom"
const TypeJSON = "json"
