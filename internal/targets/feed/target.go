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
	title := "Untitled"
	if t, ok := item.Metadata["title"].(string); ok {
		title = t
	}

	link := ""
	if l, ok := item.Metadata["url"].(string); ok {
		link = l
	} else if l, ok := item.Metadata["link"].(string); ok {
		link = l
	}

	description := ""
	if d, ok := item.Metadata["description"].(string); ok {
		description = d
	}

	author := ""
	if a, ok := item.Metadata["author"].(string); ok {
		author = a
	}

	content := ""
	if s, ok := item.Metadata["summary"].(string); ok {
		content = s
	}

	err := t.feedStore.InsertEntry(ctx, item.ID, title, link, description, content, author, item.Source, item.Timestamp)
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
