package filters

import (
	"context"
	"time"

	"cartero/internal/config"
	"cartero/internal/processors/names"
	"cartero/internal/types"
)

type PublishedAtFilterProcessor struct {
	name     string
	settings config.PublishedAtFilterSettings
}

func NewPublishedAtFilterProcessor(name string, settings config.PublishedAtFilterSettings) *PublishedAtFilterProcessor {
	return &PublishedAtFilterProcessor{
		name:     name,
		settings: settings,
	}
}

func (p *PublishedAtFilterProcessor) Name() string {
	return p.name
}

func (p *PublishedAtFilterProcessor) DependsOn() []string {
	return []string{
		names.Dedupe,
	}
}

func (p *PublishedAtFilterProcessor) Process(_ context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	logger := st.GetLogger()

	if p.settings.After == "" && p.settings.Before == "" {
		return items, nil
	}

	var after, before time.Time
	if p.settings.After != "" {
		t, err := time.Parse(time.RFC3339, p.settings.After)
		if err != nil {
			logger.Error("published_at: invalid 'after' time format", "processor", p.name, "after", p.settings.After, "error", err)
			return items, nil
		}
		after = t
	}
	if p.settings.Before != "" {
		t, err := time.Parse(time.RFC3339, p.settings.Before)
		if err != nil {
			logger.Error("published_at: invalid 'before' time format", "processor", p.name, "before", p.settings.Before, "error", err)
			return items, nil
		}
		before = t
	}

	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		ts := item.GetTimestamp()
		if !after.IsZero() && ts.Before(after) {
			logger.Debug("published_at: dropped item", "processor", p.name, "item_id", item.ID, "published_at", ts, "after", after)
			continue
		}
		if !before.IsZero() && ts.After(before) {
			logger.Debug("published_at: dropped item", "processor", p.name, "item_id", item.ID, "published_at", ts, "before", before)
			continue
		}
		out = append(out, item)
	}
	return out, nil
}
