package feed

import (
	"cartero/internal/storage"
	"cartero/internal/types"
	"context"
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
	ID              string
	Item            FeedItem
	Source          string
	MatchedKeywords string
	PublishedAt     time.Time
}


func (f *FeedItem) From(item *types.Item) {
	if title := item.GetTitle(); title != "" {
		f.Title = title
	} else {
		f.Title = "Untitled"
	}

	f.Link = item.GetLink()
	f.Description = item.GetDescription()
	f.Content = item.GetFeedContent()
	f.Author = item.GetAuthor()
	f.ImageURL = item.GetImageURL()
	f.Published = item.Timestamp.Format(time.RFC3339)
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

func InsertIntoStore(ctx context.Context, store storage.EntryStore, entry FeedEntry) error {
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
		entry.MatchedKeywords,
		publishedAt,
	)
}
