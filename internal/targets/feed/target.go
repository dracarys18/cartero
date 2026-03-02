package feed

import (
	"context"
	"time"

	"cartero/internal/components"
	"cartero/internal/storage"
	"cartero/internal/types"
)

type Target struct {
	name      string
	feedStore storage.FeedStore
}

func New(name string, registry *components.Registry) *Target {
	store := registry.Get(components.StorageComponentName).(*components.StorageComponent).Store()
	return &Target{
		name:      name,
		feedStore: store.Feed(),
	}
}

func (t *Target) Name() string {
	return t.name
}

func (t *Target) Initialize(ctx context.Context) error {
	return nil
}

func (t *Target) Publish(ctx context.Context, item *types.Item) (*types.PublishResult, error) {
	var feedItem FeedItem
	feedItem.From(item)

	entry := NewFeedEntry(item.ID, feedItem, item.Source, item.Timestamp)
	entry.MatchedKeywords = item.GetMatchedKeywords()

	err := InsertIntoStore(ctx, t.feedStore, entry)
	if err != nil {
		return nil, err
	}

	return &types.PublishResult{
		Success:   true,
		Target:    t.name,
		ItemID:    item.ID,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"feed_type": "rss/atom/json",
		},
	}, nil
}

func (t *Target) Shutdown(ctx context.Context) error {
	return nil
}
