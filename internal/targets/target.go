package targets

import (
	"cartero/internal/core"
	feedpkg "cartero/internal/targets/feed"
)

type FeedConfig struct {
	Port     string
	FeedSize int
	MaxItems int
}

func NewFeedTarget(name string, config FeedConfig) core.Target {
	return feedpkg.NewTarget(name, feedpkg.Config{
		Port:     config.Port,
		FeedSize: config.FeedSize,
		MaxItems: config.MaxItems,
	})
}
