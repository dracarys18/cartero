package sources

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"cartero/internal/config"
	"cartero/internal/sources/rss"
	"cartero/internal/types"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
)

type RSSSource struct {
	name     string
	feedURL  string
	parser   *gofeed.Parser
	maxItems int
}

func NewRSSSource(name string, feedURL string, maxItems int) *RSSSource {
	if maxItems == 0 {
		maxItems = 50
	}

	return &RSSSource{
		name:     name,
		feedURL:  feedURL,
		parser:   gofeed.NewParser(),
		maxItems: maxItems,
	}
}

func (r *RSSSource) Name() string {
	return r.name
}

func (r *RSSSource) Initialize(ctx context.Context) error {
	return nil
}

func (r *RSSSource) Fetch(ctx context.Context, state types.StateAccessor) (<-chan *types.Item, <-chan error) {
	itemChan := make(chan *types.Item)
	errChan := make(chan error, 1)

	go func() {
		defer close(itemChan)
		defer close(errChan)

		logger := state.GetLogger()

		feed, err := r.parser.ParseURLWithContext(r.feedURL, ctx)
		if err != nil {
			logger.Error("RSS source error fetching feed", "source", r.name, "error", err)
			errChan <- fmt.Errorf("failed to parse feed: %w", err)
			return
		}

		logger.Debug("RSS source retrieved items", "source", r.name, "count", len(feed.Items))

		limit := r.maxItems
		if limit > len(feed.Items) {
			limit = len(feed.Items)
		}

		logger.Debug("RSS source processing items", "source", r.name, "limit", limit)

		for i := 0; i < limit; i++ {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				feedItem := feed.Items[i]
				item := r.convertToItem(feedItem)

				select {
				case itemChan <- item:
					logger.Debug("RSS source sent item", "source", r.name, "index", i+1, "limit", limit, "item_id", item.ID)
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				}
			}
		}

		logger.Debug("RSS source finished processing all items", "source", r.name)
	}()

	return itemChan, errChan
}

func (r *RSSSource) convertToItem(feedItem *gofeed.Item) *types.Item {
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
		itemID = fmt.Sprintf("rss_%s_%d", r.name, timestamp.Unix())
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
	}

	if len(categories) > 0 {
		metadata["categories"] = categories
		metadata["category"] = strings.Join(categories, ", ")
	}

	// Add comments link if available
	if feedItem.Custom != nil {
		if comments, ok := feedItem.Custom["comments"]; ok {
			metadata["comments"] = comments
		}
	}

	return &types.Item{
		ID:        fmt.Sprintf("rss_%s", sanitizeID(itemID)),
		Content:   feedItem,
		Source:    r.name,
		Timestamp: timestamp,
		Metadata:  metadata,
	}
}

func (r *RSSSource) Shutdown(ctx context.Context) error {
	return nil
}

func NewRSSSourceFromConfig(name string, cfg config.RSSSettings, maxItems int) (types.Source, error) {
	sourceConfig := rss.SourceConfig{
		Type:  cfg.From.Type,
		Kind:  cfg.From.Kind,
		Value: cfg.From.Value,
	}

	return rss.NewSource(name, sourceConfig, maxItems)
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
