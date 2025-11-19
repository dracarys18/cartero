package sources

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"cartero/internal/core"

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
	log.Printf("RSS source %s: initializing (feed_url=%s, max_items=%d)", r.name, r.feedURL, r.maxItems)
	return nil
}

func (r *RSSSource) Fetch(ctx context.Context) (<-chan *core.Item, <-chan error) {
	itemChan := make(chan *core.Item)
	errChan := make(chan error, 1)

	go func() {
		defer close(itemChan)
		defer close(errChan)

		log.Printf("RSS source %s: fetching feed", r.name)
		feed, err := r.parser.ParseURLWithContext(r.feedURL, ctx)
		if err != nil {
			log.Printf("RSS source %s: error fetching feed: %v", r.name, err)
			errChan <- fmt.Errorf("failed to parse feed: %w", err)
			return
		}

		log.Printf("RSS source %s: retrieved %d items", r.name, len(feed.Items))

		limit := r.maxItems
		if limit > len(feed.Items) {
			limit = len(feed.Items)
		}

		log.Printf("RSS source %s: processing %d items", r.name, limit)

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
					log.Printf("RSS source %s: sent item %d/%d (id=%s)",
						r.name, i+1, limit, item.ID)
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				}
			}
		}

		log.Printf("RSS source %s: finished processing all items", r.name)
	}()

	return itemChan, errChan
}

func (r *RSSSource) convertToItem(feedItem *gofeed.Item) *core.Item {
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

	return &core.Item{
		ID:        fmt.Sprintf("rss_%s", sanitizeID(itemID)),
		Content:   feedItem,
		Source:    r.name,
		Timestamp: timestamp,
		Metadata:  metadata,
	}
}

func (r *RSSSource) Shutdown(ctx context.Context) error {
	log.Printf("RSS source %s: shutting down", r.name)
	return nil
}

// sanitizeID removes characters that might cause issues in item IDs
func sanitizeID(id string) string {
	// Replace problematic characters
	id = strings.ReplaceAll(id, "://", "_")
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, "?", "_")
	id = strings.ReplaceAll(id, "&", "_")
	id = strings.ReplaceAll(id, "=", "_")
	id = strings.ReplaceAll(id, "#", "_")
	id = strings.ReplaceAll(id, " ", "_")

	// Limit length
	if len(id) > 200 {
		id = id[:200]
	}

	return id
}

var htmlStripper = bluemonday.StrictPolicy()

// stripHTML removes HTML tags and decodes entities from text
func stripHTML(s string) string {
	// Strip all HTML tags
	s = htmlStripper.Sanitize(s)

	// Decode HTML entities
	s = html.UnescapeString(s)

	// Trim whitespace
	s = strings.TrimSpace(s)

	// Limit length to avoid extremely long descriptions
	if len(s) > 500 {
		s = s[:497] + "..."
	}

	return s
}
