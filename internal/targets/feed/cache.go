package feed

import (
	"fmt"

	"cartero/internal/cache"
)

type CacheKey struct {
	TargetName string
	FeedType   string
}

func NewCacheKey(targetName, feedType string) CacheKey {
	return CacheKey{
		TargetName: targetName,
		FeedType:   feedType,
	}
}

func (k CacheKey) ToString() string {
	return fmt.Sprintf("%s:%s", k.TargetName, k.FeedType)
}

func NewCache(config cache.CacheConfig) *cache.Cache[CacheKey, string] {
	return cache.NewCache[CacheKey, string](config, func(k CacheKey) string {
		return k.ToString()
	})
}

const TypeRSS = "rss"
const TypeAtom = "atom"
const TypeJSON = "json"
