package processors

import (
	"context"

	"cartero/internal/components"
	procnames "cartero/internal/processors/names"
	"cartero/internal/types"
)

const embedBatchSize = 64

type EmbedTextProcessor struct {
	name string
}

func NewEmbedTextProcessor(name string) *EmbedTextProcessor {
	return &EmbedTextProcessor{name: name}
}

func (e *EmbedTextProcessor) Name() string {
	return e.name
}

func (e *EmbedTextProcessor) DependsOn() []string {
	return []string{procnames.Dedupe, procnames.ScoreFilter, procnames.PublishedAt}
}

func (e *EmbedTextProcessor) Process(ctx context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	pc := st.GetRegistry().Get(components.PlatformComponentName).(*components.PlatformComponent)
	embedder := pc.Embedder()
	if embedder == nil {
		return items, nil
	}

	var pending []*types.Item
	for _, item := range items {
		if item.GetTitle() != "" {
			pending = append(pending, item)
		}
	}

	for start := 0; start < len(pending); start += embedBatchSize {
		chunk := pending[start:min(start+embedBatchSize, len(pending))]
		titles := make([]string, len(chunk))
		for i, item := range chunk {
			titles[i] = item.GetTitle()
		}

		vecs, err := embedder.Embed(ctx, titles)
		if err != nil || len(vecs) != len(chunk) {
			st.GetLogger().Warn("embed_text: embedding failed", "processor", e.name, "error", err)
			return items, nil
		}
		for i, item := range chunk {
			item.SetEmbedding(embedder.Model(), vecs[i])
		}
	}
	return items, nil
}
