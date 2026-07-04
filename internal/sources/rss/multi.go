package rss

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"strings"
	"sync"
	"time"

	"cartero/internal/types"
	"cartero/internal/utils/batch"
	strutils "cartero/internal/utils/string"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
)

const (
	maxConcurrentFeeds = 16
	feedFetchTimeout   = 20 * time.Second
	feedUserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

type MultiRSSSource struct {
	name   string
	feeds  []Feed
	parser *gofeed.Parser
}

func NewMultiRSSSource(name string, feeds []Feed) *MultiRSSSource {
	p := gofeed.NewParser()
	p.UserAgent = feedUserAgent
	return &MultiRSSSource{
		name:   name,
		feeds:  feeds,
		parser: p,
	}
}

func (m *MultiRSSSource) Name() string {
	return m.name
}

func (m *MultiRSSSource) Initialize(ctx context.Context) error {
	return nil
}

func (m *MultiRSSSource) Fetch(ctx context.Context, state types.StateAccessor) ([]*types.Item, error) {
	logger := state.GetLogger()

	logger.Info("MultiRSS source fetching feeds", "source", m.name, "count", len(m.feeds))

	var mu sync.Mutex
	var out []*types.Item

	batch.Run(ctx, m.feeds, maxConcurrentFeeds, func(ctx context.Context, feed Feed) {
		items := m.fetchFeed(ctx, feed, state)
		mu.Lock()
		out = append(out, items...)
		mu.Unlock()
	})

	logger.Info("MultiRSS source finished", "source", m.name)
	return out, nil
}

func (m *MultiRSSSource) fetchFeed(ctx context.Context, feed Feed, state types.StateAccessor) []*types.Item {
	logger := state.GetLogger()

	fctx, cancel := context.WithTimeout(ctx, feedFetchTimeout)
	defer cancel()

	parsedFeed, err := m.parser.ParseURLWithContext(feed.URL.String(), fctx)
	if err != nil {
		logger.Error("MultiRSS feed fetch error", "source", m.name, "feed", feed.Name, "url", feed.URL.String(), "error", err)
		return nil
	}

	logger.Debug("MultiRSS feed retrieved", "source", m.name, "feed", feed.Name, "items", len(parsedFeed.Items))

	limit := feed.MaxItems
	if limit > len(parsedFeed.Items) {
		limit = len(parsedFeed.Items)
	}

	var out []*types.Item
	for i := 0; i < limit; i++ {
		select {
		case <-ctx.Done():
			return out
		default:
		}

		item := m.convertToItem(parsedFeed.Items[i], feed.Name, feed.URL)
		out = append(out, item)
		logger.Debug("MultiRSS item published", "source", m.name, "feed", feed.Name, "item", i+1)
	}
	return out
}

func (m *MultiRSSSource) convertToItem(feedItem *gofeed.Item, feedName string, base *url.URL) *types.Item {
	link := base
	if abs, err := base.Parse(feedItem.Link); err == nil {
		link = abs
	}

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
	if len(feedItem.Content) > len(description) {
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
		"link":        link,
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
		Title:     feedItem.Title,
		URL:       link,
		Content:   feedItem,
		Source:    sourceName,
		Route:     m.name,
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

	return strutils.Truncate(id, 200)
}

var htmlStripper = bluemonday.StrictPolicy()

func stripHTML(s string) string {
	s = htmlStripper.Sanitize(s)
	s = html.UnescapeString(s)
	s = strings.TrimSpace(s)

	return strutils.Truncate(s, 500)
}
