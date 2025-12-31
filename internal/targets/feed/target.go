package feed

import (
	"context"
	"log"
	"time"

	"cartero/internal/components"
	"cartero/internal/core"
	"cartero/internal/storage"
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

func (t *Target) Publish(ctx context.Context, item *core.ProcessedItem) (*core.PublishResult, error) {
	title := "Untitled"
	if t, ok := item.Original.Metadata["title"].(string); ok {
		title = t
	}

	link := ""
	if l, ok := item.Original.Metadata["url"].(string); ok {
		link = l
	} else if l, ok := item.Original.Metadata["link"].(string); ok {
		link = l
	}

	description := ""
	if d, ok := item.Original.Metadata["description"].(string); ok {
		description = d
	}

	author := ""
	if a, ok := item.Original.Metadata["author"].(string); ok {
		author = a
	}

	content := ""
	if s, ok := item.Original.Metadata["summary"].(string); ok {
		content = s
	}

	err := t.feedStore.InsertEntry(ctx, item.Original.ID, title, link, description, content, author, item.Original.Source, item.Original.Timestamp)
	if err != nil {
		log.Printf("Feed target %s: failed to insert entry %s: %v", t.name, item.Original.ID, err)
		return nil, err
	}

	log.Printf("Feed target %s: inserted entry %s", t.name, item.Original.ID)

	return &core.PublishResult{
		Success:   true,
		Target:    t.name,
		ItemID:    item.Original.ID,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"feed_type": "rss/atom/json",
		},
	}, nil
}

func (t *Target) Shutdown(ctx context.Context) error {
	return nil
}
