package filters

import (
	"context"
	"time"

	"cartero/internal/processors/names"
	"cartero/internal/types"
)

type TimeFilter struct {
	cutoff time.Time
	name   string
}

func (t *TimeFilter) FilterAfter(itemTime time.Time, itemID string) error {
	if itemTime.Before(t.cutoff) {
		return types.NewFilteredError(t.name, itemID, "published before cutoff time").
			WithDetail("published_at", itemTime).
			WithDetail("after", t.cutoff)
	}
	return nil
}

func (t *TimeFilter) FilterBefore(itemTime time.Time, itemID string) error {
	if itemTime.After(t.cutoff) {
		return types.NewFilteredError(t.name, itemID, "published after cutoff time").
			WithDetail("published_at", itemTime).
			WithDetail("before", t.cutoff)
	}
	return nil
}

type PublishedAtFilterProcessor struct {
	name string
}

func NewPublishedAtFilterProcessor(name string) *PublishedAtFilterProcessor {
	return &PublishedAtFilterProcessor{
		name: name,
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

func (p *PublishedAtFilterProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[p.name].Settings.PublishedAtFilterSettings
	logger := st.GetLogger()

	if cfg.After == "" && cfg.Before == "" {
		return nil
	}

	itemTimestamp := item.GetTimestamp()

	if cfg.After != "" {
		afterTime, err := time.Parse(time.RFC3339, cfg.After)
		if err != nil {
			logger.Error("PublishedAtFilterProcessor: invalid 'after' time format", "processor", p.name, "after", cfg.After, "error", err)
			return nil
		}

		filter := TimeFilter{cutoff: afterTime, name: p.name}
		if err := filter.FilterAfter(itemTimestamp, item.ID); err != nil {
			logger.Info("PublishedAtFilterProcessor rejected item", "processor", p.name, "item_id", item.ID, "published_at", itemTimestamp, "after", afterTime)
			return err
		}
	}

	if cfg.Before != "" {
		beforeTime, err := time.Parse(time.RFC3339, cfg.Before)
		if err != nil {
			logger.Error("PublishedAtFilterProcessor: invalid 'before' time format", "processor", p.name, "before", cfg.Before, "error", err)
			return nil
		}

		filter := TimeFilter{cutoff: beforeTime, name: p.name}
		if err := filter.FilterBefore(itemTimestamp, item.ID); err != nil {
			logger.Info("PublishedAtFilterProcessor rejected item", "processor", p.name, "item_id", item.ID, "published_at", itemTimestamp, "before", beforeTime)
			return err
		}
	}

	return nil
}

func PublishedAtFilter(name string) *PublishedAtFilterProcessor {
	return NewPublishedAtFilterProcessor(name)
}
