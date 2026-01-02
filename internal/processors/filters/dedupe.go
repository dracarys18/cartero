package filters

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cartero/internal/core"
)

type DedupeProcessor struct {
	name      string
	seen      map[string]time.Time
	ttl       time.Duration
	mu        sync.RWMutex
	cleanupCh chan struct{}
}

func NewDedupeProcessor(name string, ttl time.Duration) *DedupeProcessor {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	d := &DedupeProcessor{
		name:      name,
		seen:      make(map[string]time.Time),
		ttl:       ttl,
		cleanupCh: make(chan struct{}),
	}

	go d.cleanup()

	return d
}

func (d *DedupeProcessor) Name() string {
	return d.name
}

func (d *DedupeProcessor) DependsOn() []string {
	return []string{}
}

// Process implements the Processor interface
func (d *DedupeProcessor) Process(ctx context.Context, item *core.Item) error {
	hash := d.hashItem(item)

	d.mu.Lock()
	defer d.mu.Unlock()

	if lastSeen, exists := d.seen[hash]; exists {
		// Item is a duplicate, filter it out
		return fmt.Errorf("DedupeProcessor %s: duplicate item %s (first seen: %v)", d.name, item.ID, lastSeen)
	}

	// Item is unique, mark as seen and allow processing
	d.seen[hash] = time.Now()
	return nil
}

func (d *DedupeProcessor) hashItem(item *core.Item) string {
	data, _ := json.Marshal(map[string]interface{}{
		"id":      item.ID,
		"content": item.Content,
	})
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func (d *DedupeProcessor) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.mu.Lock()
			now := time.Now()
			for hash, timestamp := range d.seen {
				if now.Sub(timestamp) > d.ttl {
					delete(d.seen, hash)
				}
			}
			d.mu.Unlock()
		case <-d.cleanupCh:
			return
		}
	}
}

func (d *DedupeProcessor) Stop() {
	close(d.cleanupCh)
}

type ContentDedupeProcessor struct {
	name      string
	seen      map[string]bool
	mu        sync.RWMutex
	fieldName string
}

func NewContentDedupeProcessor(name string, fieldName string) *ContentDedupeProcessor {
	return &ContentDedupeProcessor{
		name:      name,
		seen:      make(map[string]bool),
		fieldName: fieldName,
	}
}

func (c *ContentDedupeProcessor) Name() string {
	return c.name
}

func (c *ContentDedupeProcessor) DependsOn() []string {
	return []string{}
}

func (c *ContentDedupeProcessor) Process(ctx context.Context, item *core.Item) error {
	var content string
	if c.fieldName != "" {
		if val, ok := item.Metadata[c.fieldName]; ok {
			content = fmt.Sprintf("%v", val)
		}
	} else {
		data, _ := json.Marshal(item.Content)
		content = string(data)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.seen[hash] {
		return fmt.Errorf("ContentDedupeProcessor %s: duplicate content detected for item %s", c.name, item.ID)
	}

	c.seen[hash] = true
	return nil
}

func (c *ContentDedupeProcessor) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen = make(map[string]bool)
}
