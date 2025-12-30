package targets

import (
	"cartero/internal/core"
	"cartero/internal/storage"
	feedpkg "cartero/internal/targets/feed"
)

func NewFeedTarget(name string, feedStore storage.FeedStore) core.Target {
	return feedpkg.New(name, feedStore)
}
