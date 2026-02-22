package rss

import (
	"context"
	"fmt"
	"html"
	"strings"
	"sync"
	"time"

	"cartero/internal/types"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
)

type MultiRSSSource struct {
	name   string
	feeds  []Feed
	parser *gofeed.Parser
}

func NewMultiRSSSource(name string, feeds []Feed) *MultiRSSSource {
	return &MultiRSSSource{
		name:   name,
		feeds:  feeds,
		parser: gofeed.NewParser(),
	}
}

func (m *MultiRSSSource) Name() string {
	return m.name
}

func (m *MultiRSSSource) Initialize(ctx context.Context) error {
	return nil
}

func (m *MultiRSSSource) Fetch(ctx context.Context, state types.StateAccessor) (<-chan *types.Item, <-chan error) {
	itemChan := make(chan *types.Item, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(itemChan)
		defer close(errChan)

		logger := state.GetLogger()
		logger.Info("MultiRSS source fetching feeds", "source", m.name, "count", len(m.feeds))

		var wg sync.WaitGroup
		feedItemChan := make(chan *types.Item, 100)

		for _, feed := range m.feeds {
			wg.Add(1)
			go func(f Feed) {
				defer wg.Done()
				m.fetchFeed(ctx, f, feedItemChan, state)
			}(feed)
		}

		go func() {
			wg.Wait()
			close(feedItemChan)
		}()

		for item := range feedItemChan {
			select {
			case itemChan <- item:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}

		logger.Info("MultiRSS source finished", "source", m.name)
	}()

	return itemChan, errChan
}

func (m *MultiRSSSource) fetchFeed(ctx context.Context, feed Feed, itemChan chan<- *types.Item, state types.StateAccessor) {
	logger := state.GetLogger()

	parsedFeed, err := m.parser.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		logger.Error("MultiRSS feed fetch error", "source", m.name, "feed", feed.Name, "url", feed.URL, "error", err)
		return
	}

	logger.Debug("MultiRSS feed retrieved", "source", m.name, "feed", feed.Name, "items", len(parsedFeed.Items))

	limit := feed.MaxItems
	if limit > len(parsedFeed.Items) {
		limit = len(parsedFeed.Items)
	}

	for i := 0; i < limit; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			feedItem := parsedFeed.Items[i]
			item := m.convertToItem(feedItem, feed.Name)

			select {
			case itemChan <- item:
				logger.Debug("MultiRSS item sent", "source", m.name, "feed", feed.Name, "item", i+1)
			case <-ctx.Done():
				return
			}
		}
	}
}

func (m *MultiRSSSource) convertToItem(feedItem *gofeed.Item, feedName string) *types.Item {
	timestamp := time.Now()
	if feedItem.PublishedParsed != nil {
		timestamp = *feedItem.PublishedParsed
	} else if feedItem.UpdatedParsed != nil {
		timestamp = *feedItem.UpdatedParsed
	}

	itemID := feedItem.GUID
	if itemID == "" {
		itemID = feedItem.Link
	}
	if itemID == "" {
		itemID = fmt.Sprintf("rss_%s_%d", feedName, timestamp.Unix())
	}

	description := feedItem.Description
	if description == "" && feedItem.Content != "" {
		description = feedItem.Content
	}

	author := ""
	if feedItem.Author != nil {
		author = feedItem.Author.Name
		if author == "" {
			author = feedItem.Author.Email
		}
	}

	categories := []string{}
	if len(feedItem.Categories) > 0 {
		categories = feedItem.Categories
	}

	metadata := map[string]interface{}{
		"title":       feedItem.Title,
		"link":        feedItem.Link,
		"url":         feedItem.Link,
		"description": stripHTML(description),
		"author":      author,
		"published":   feedItem.Published,
		"updated":     feedItem.Updated,
		"feed_name":   feedName,
	}

	if len(categories) > 0 {
		metadata["categories"] = categories
		metadata["category"] = strings.Join(categories, ", ")
	}

	if feedItem.Custom != nil {
		if comments, ok := feedItem.Custom["comments"]; ok {
			metadata["comments"] = comments
		}
	}

	sourceName := fmt.Sprintf("%s_%s", m.name, feedName)

	return &types.Item{
		ID:        fmt.Sprintf("rss_%s", sanitizeID(itemID)),
		Content:   feedItem,
		Source:    sourceName,
		Timestamp: timestamp,
		Metadata:  metadata,
	}
}

func (m *MultiRSSSource) Shutdown(ctx context.Context) error {
	return nil
}

func sanitizeID(id string) string {
	id = strings.ReplaceAll(id, "://", "_")
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, "?", "_")
	id = strings.ReplaceAll(id, "&", "_")
	id = strings.ReplaceAll(id, "=", "_")
	id = strings.ReplaceAll(id, "#", "_")
	id = strings.ReplaceAll(id, " ", "_")

	if len(id) > 200 {
		id = id[:200]
	}

	return id
}

var htmlStripper = bluemonday.StrictPolicy()

func stripHTML(s string) string {
	s = htmlStripper.Sanitize(s)
	s = html.UnescapeString(s)
	s = strings.TrimSpace(s)

	if len(s) > 500 {
		s = s[:497] + "..."
	}

	return s
}
