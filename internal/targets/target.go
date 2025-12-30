package targets

import (
	"cartero/internal/core"
	"cartero/internal/storage"
	feedpkg "cartero/internal/targets/feed"
)

type FeedConfig struct {
	Port     string
	FeedSize int
	MaxItems int
}

func NewFeedTarget(name string, config FeedConfig, feedStore storage.FeedStore) core.Target {
	return feedpkg.NewTarget(name, feedpkg.Config{
		Port:     config.Port,
		FeedSize: config.FeedSize,
		MaxItems: config.MaxItems,
	}, feedStore)
}
