package processors

import (
	"context"
	"strings"

	"github.com/tmc/langchaingo/textsplitter"

	"cartero/internal/components"
	"cartero/internal/config"
	procnames "cartero/internal/processors/names"
	"cartero/internal/types"
)

type EmbedTextProcessor struct {
	name     string
	settings config.EmbedTextSettings
}

func NewEmbedTextProcessor(name string, settings config.EmbedTextSettings) *EmbedTextProcessor {
	return &EmbedTextProcessor{name: name, settings: settings}
}

func (e *EmbedTextProcessor) Name() string {
	return e.name
}

func (e *EmbedTextProcessor) DependsOn() []string {
	return []string{procnames.ExtractText}
}

func (e *EmbedTextProcessor) Process(ctx context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	logger := st.GetLogger()

	pc := st.GetRegistry().Get(components.PlatformComponentName).(*components.PlatformComponent)
	embedder := pc.Embedder()
	if embedder == nil {
		return items, nil
	}

	chunkSize := e.settings.ChunkSize
	if chunkSize == 0 {
		chunkSize = 400
	}

	for _, item := range items {
		var body, description string
		if article := item.GetArticle(); article != nil {
			body = article.Text
			description = article.Description
		}

		chunks := []string{}
		if item.GetTitle() != "" {
			chunks = append(chunks, item.GetTitle())
		}
		if description != "" {
			chunks = append(chunks, description)
		}

		if body != "" {
			splitter := textsplitter.NewRecursiveCharacter(
				textsplitter.WithChunkSize(chunkSize),
				textsplitter.WithChunkOverlap(chunkSize/8),
			)
			bodyChunk, err := splitter.SplitText(body)
			if err != nil {
				logger.Warn("embed_text: failed to split text", "processor", e.name, "item_id", item.ID, "error", err)
			} else if len(bodyChunk) > 0 {
				chunks = append(chunks, bodyChunk...)
			}
		}

		cleanChunks := make([]string, 0, len(chunks))
		for _, chunk := range chunks {
			cleanChunks = append(cleanChunks, strings.TrimSpace(chunk))
		}
		if len(cleanChunks) == 0 {
			continue
		}

		embeddings, err := embedder.Embed(ctx, cleanChunks)
		if err != nil {
			logger.Warn("embed_text: failed to embed item", "processor", e.name, "item_id", item.ID, "error", err)
			continue
		}
		if len(embeddings) == 0 {
			logger.Warn("embed_text: empty embeddings returned", "processor", e.name, "item_id", item.ID)
			continue
		}

		item.SetEmbedding(embeddings)
		logger.Debug("embed_text: stored embedding", "processor", e.name, "item_id", item.ID, "chunks", len(cleanChunks), "dim", len(embeddings[0]))
	}

	return items, nil
}
