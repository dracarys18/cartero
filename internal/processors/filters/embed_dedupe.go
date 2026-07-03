package filters

import (
	"context"
	"time"

	"cartero/internal/config"
	"cartero/internal/processors/names"
	"cartero/internal/types"
)

type EmbedDedupeProcessor struct {
	name      string
	threshold float64
	window    time.Duration
}

func NewEmbedDedupeProcessor(name string, settings config.DedupeSettings) *EmbedDedupeProcessor {
	threshold := settings.EmbedThreshold
	if threshold == 0 {
		threshold = 0.9
	}

	windowStr := settings.EmbedWindow
	if windowStr == "" {
		windowStr = "168h"
	}
	window, err := time.ParseDuration(windowStr)
	if err != nil {
		window = 168 * time.Hour
	}

	return &EmbedDedupeProcessor{
		name:      name,
		threshold: threshold,
		window:    window,
	}
}

func (d *EmbedDedupeProcessor) Name() string {
	return d.name
}

func (d *EmbedDedupeProcessor) DependsOn() []string {
	return []string{names.EmbedText}
}

func (d *EmbedDedupeProcessor) Process(ctx context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	store := st.GetStorage().Entries()
	logger := st.GetLogger()
	since := time.Now().Add(-d.window)

	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		embeddings := item.GetEmbedding()
		if len(embeddings) == 0 {
			out = append(out, item)
			continue
		}

		similar, err := store.FindNearestEmbedding(ctx, embeddings[0], d.threshold, since)
		if err != nil {
			logger.Warn("embed_dedupe: check failed", "processor", d.name, "item_id", item.ID, "error", err)
			out = append(out, item)
			continue
		}

		if similar {
			logger.Debug("embed_dedupe: dropped item", "processor", d.name, "item_id", item.ID, "reason", "semantic duplicate")
			continue
		}

		out = append(out, item)
	}
	return out, nil
}
