package filters

import (
	"context"
	"math"
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
	return &EmbedDedupeProcessor{
		name:      name,
		threshold: threshold,
		window:    config.ParseDuration(settings.EmbedWindow, 168*time.Hour),
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
	since := time.Now().Add(-d.window)

	out := make([]*types.Item, 0, len(items))
	var kept [][]float32
	for _, item := range items {
		model, vec := item.GetEmbedding()
		if len(vec) == 0 {
			out = append(out, item)
			continue
		}

		if d.matchesAny(vec, kept) {
			continue
		}

		similar, err := store.FindSimilarEntry(ctx, model, vec, d.threshold, since)
		if err != nil {
			st.GetLogger().Warn("embed_dedupe: check failed", "processor", d.name, "item_id", item.ID, "error", err)
		}
		if similar {
			continue
		}

		kept = append(kept, vec)
		out = append(out, item)
	}
	return out, nil
}

func (d *EmbedDedupeProcessor) matchesAny(vec []float32, kept [][]float32) bool {
	for _, k := range kept {
		if cosine(vec, k) >= d.threshold {
			return true
		}
	}
	return false
}

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / math.Sqrt(na*nb)
}
