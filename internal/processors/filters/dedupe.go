package filters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cartero/internal/types"
	"cartero/internal/utils/hash"
)

type DedupeProcessor struct {
	name      string
	seen      map[string]time.Time
	mu        sync.RWMutex
	cleanupCh chan struct{}
}

func NewDedupeProcessor(name string) *DedupeProcessor {
	d := &DedupeProcessor{
		name:      name,
		seen:      make(map[string]time.Time),
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

func (d *DedupeProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	hash := d.hashItem(item)
	store := st.GetStorage().Items()
	logger := st.GetLogger()

	exists, err := store.Exists(ctx, item.ID)
	if err != nil {
		return err
	}

	if exists {
		logger.Info("DedupeProcessor rejected item", "processor", d.name, "item_id", item.ID, "reason", "exists in storage")
		return fmt.Errorf("DedupeProcessor %s: duplicate item %s (exists in storage)", d.name, item.ID)
	}

	if lastSeen, exists := d.seen[hash]; exists {
		logger.Info("DedupeProcessor rejected item", "processor", d.name, "item_id", item.ID, "reason", "seen in current session", "first_seen", lastSeen)
		return fmt.Errorf("DedupeProcessor %s: duplicate item %s (first seen: %v)", d.name, item.ID, lastSeen)
	}

	d.seen[hash] = time.Now()
	return nil
}

func (d *DedupeProcessor) hashItem(item *types.Item) string {
	data, _ := json.Marshal(map[string]interface{}{
		"id":      item.ID,
		"source":  item.Source,
		"content": item.Content,
	})
	return hash.NewHash(data).ComputeHash()
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
				if now.Sub(timestamp) > 24*time.Hour {
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
