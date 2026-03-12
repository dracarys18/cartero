package processors

import (
	"context"
	"strings"

	"cartero/internal/components"
	procnames "cartero/internal/processors/names"
	"cartero/internal/types"

	"github.com/ollama/ollama/api"
)

type EmbedTextProcessor struct {
	name string
}

func NewEmbedTextProcessor(name string) *EmbedTextProcessor {
	return &EmbedTextProcessor{name: name}
}

func (e *EmbedTextProcessor) Name() string {
	return e.name
}

func (e *EmbedTextProcessor) Initialize(_ context.Context, _ types.StateAccessor) error {
	return nil
}

func (e *EmbedTextProcessor) DependsOn() []string {
	return []string{procnames.ExtractText}
}

func (e *EmbedTextProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[e.name].Settings.EmbedTextSettings
	logger := st.GetLogger()

	var body string
	if article := item.GetArticle(); article != nil {
		body = article.Text
	}
	text := strings.TrimSpace(item.GetTitle() + " " + body)

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
		logger.Warn("embed_text: failed to embed item", "processor", e.name, "item_id", item.ID, "error", err)
		return nil
	}

	if len(resp.Embeddings) == 0 {
		logger.Warn("embed_text: empty embeddings returned", "processor", e.name, "item_id", item.ID)
		return nil
	}

	item.SetEmbedding(resp.Embeddings[0])
	logger.Debug("embed_text: stored embedding", "processor", e.name, "item_id", item.ID, "dim", len(resp.Embeddings[0]))
	return nil
}
