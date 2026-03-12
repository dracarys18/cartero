package processors

import (
	"context"

	"cartero/internal/components"
	procnames "cartero/internal/processors/names"
	"cartero/internal/types"

	"github.com/ollama/ollama/api"
)

type EmbedCategoryProcessor struct {
	name string
}

func NewEmbedCategoryProcessor(name string) *EmbedCategoryProcessor {
	return &EmbedCategoryProcessor{name: name}
}

func (e *EmbedCategoryProcessor) Name() string {
	return e.name
}

func (e *EmbedCategoryProcessor) Initialize(_ context.Context, _ types.StateAccessor) error {
	return nil
}

func (e *EmbedCategoryProcessor) DependsOn() []string {
	return []string{procnames.ExtractText}
}

func (e *EmbedCategoryProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[e.name].Settings.EmbedCategorySettings
	logger := st.GetLogger()

	var text string
	if article := item.GetArticle(); article != nil && article.Text != "" {
		text = article.Text
	} else {
		text = item.GetTitle()
	}

	if text == "" {
		return nil
	}

	limit := cfg.InputLimit
	if limit == 0 {
		limit = 1024
	}
	if len(text) > limit {
		text = text[:limit]
	}

	registry := st.GetRegistry()
	pc := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	ollamaClient := pc.OllamaPlatform(cfg.Model)

	resp, err := ollamaClient.Embed(ctx, &api.EmbedRequest{Input: text})
	if err != nil {
		logger.Warn("embed_category: failed to embed item", "processor", e.name, "item_id", item.ID, "error", err)
		return nil
	}

	if len(resp.Embeddings) == 0 {
		logger.Warn("embed_category: empty embeddings returned", "processor", e.name, "item_id", item.ID)
		return nil
	}

	item.SetEmbedding(resp.Embeddings[0])
	logger.Debug("embed_category: stored embedding", "processor", e.name, "item_id", item.ID, "dim", len(resp.Embeddings[0]))
	return nil
}
