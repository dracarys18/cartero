package filters

import (
	"context"
	"time"

	"cartero/internal/config"
	"cartero/internal/types"
)

type EmbedDedupeProcessor struct {
	name       string
	threshold  float64
	window     time.Duration
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

func (d *EmbedDedupeProcessor) Initialize(_ context.Context, _ types.StateAccessor) error {
	return nil
}

func (d *EmbedDedupeProcessor) DependsOn() []string {
	return []string{"embed_text"}
}

func (d *EmbedDedupeProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	store := st.GetStorage().Entries()
	logger := st.GetLogger()

	embeddings := item.GetEmbedding()
	if len(embeddings) == 0 {
		return nil
	}

	since := time.Now().Add(-d.window)
	similar, err := store.FindNearestEmbedding(ctx, embeddings[0], d.threshold, since)
	if err != nil {
		logger.Warn("EmbedDedupeProcessor check failed", "processor", d.name, "item_id", item.ID, "error", err)
		return nil
	}

	if similar {
		logger.Info("EmbedDedupeProcessor rejected item", "processor", d.name, "item_id", item.ID, "reason", "semantic duplicate")
		return types.NewFilteredError(d.name, item.ID, "semantic duplicate").
			WithDetail("threshold", d.threshold)
	}

	return nil
}
