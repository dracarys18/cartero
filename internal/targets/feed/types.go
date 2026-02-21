package feed

import (
	"cartero/internal/storage"
	"cartero/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type FeedItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	Author      string `json:"author,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	Published   string `json:"published"`
}

type FeedEntry struct {
	ID          string
	Item        FeedItem
	Source      string
	PublishedAt time.Time
}

func (f *FeedItem) TryFrom(templateOutput []byte) error {
	if err := json.Unmarshal(templateOutput, f); err != nil {
		return fmt.Errorf("feed: failed to unmarshal template output to FeedItem: %w", err)
	}
	return nil
}

func (f *FeedItem) From(item *types.Item) {
	if title, ok := item.Metadata["title"].(string); ok {
		f.Title = title
	} else {
		f.Title = "Untitled"
	}

	if link, ok := item.Metadata["url"].(string); ok {
		f.Link = link
	} else if link, ok := item.Metadata["link"].(string); ok {
		f.Link = link
	}

	if description, ok := item.Metadata["description"].(string); ok {
		f.Description = description
	}

	if author, ok := item.Metadata["author"].(string); ok {
		f.Author = author
	}

	if summary, ok := item.Metadata["summary"].(string); ok {
		f.Content = summary
	}

	if item.TextContent != nil && item.TextContent.Image != "" {
		f.ImageURL = item.TextContent.Image
	}

	f.Published = item.Timestamp.Format(time.RFC3339)
}

func (e *FeedEntry) Into() storage.FeedEntry {
	publishedAt, _ := time.Parse(time.RFC3339, e.Item.Published)

	return storage.FeedEntry{
		ID:          e.ID,
		Title:       e.Item.Title,
		Link:        e.Item.Link,
		Description: e.Item.Description,
		Content:     e.Item.Content,
		Author:      e.Item.Author,
		Source:      e.Source,
		ImageURL:    e.Item.ImageURL,
		PublishedAt: publishedAt,
		CreatedAt:   time.Now(),
	}
}

func NewFeedEntry(id string, item FeedItem, source string, publishedAt time.Time) FeedEntry {
	if item.Published == "" {
		item.Published = publishedAt.Format(time.RFC3339)
	}

	return FeedEntry{
		ID:          id,
		Item:        item,
		Source:      source,
		PublishedAt: publishedAt,
	}
}

func InsertIntoStore(ctx context.Context, store storage.FeedStore, entry FeedEntry) error {
	publishedAt := entry.PublishedAt
	if publishedAt.IsZero() {
		var err error
		publishedAt, err = time.Parse(time.RFC3339, entry.Item.Published)
		if err != nil {
			publishedAt = time.Now()
		}
	}

	return store.InsertEntry(
		ctx,
		entry.ID,
		entry.Item.Title,
		entry.Item.Link,
		entry.Item.Description,
		entry.Item.Content,
		entry.Item.Author,
		entry.Source,
		entry.Item.ImageURL,
		publishedAt,
	)
}
