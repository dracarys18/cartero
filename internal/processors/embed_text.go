package processors

import (
	"context"
	"strings"

	"github.com/tmc/langchaingo/textsplitter"

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
	var description string

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

	chunkSize := cfg.ChunkSize
	if chunkSize == 0 {
		chunkSize = 400
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

	var cleanChunks []string
	for _, chunk := range chunks {
		trimmed := strings.TrimSpace(chunk)
		cleanChunks = append(cleanChunks, trimmed)
	}

	if len(cleanChunks) == 0 {
		return nil
	}

	registry := st.GetRegistry()
	pc := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	embedder := pc.Embedder()

	if embedder == nil {
		return nil
	}

	resp, err := embedder.Embed(ctx, &api.EmbedRequest{Input: cleanChunks})

	if err != nil {
		logger.Warn("embed_text: failed to embed item", "processor", e.name, "item_id", item.ID, "error", err)
		return nil
	}

	if len(resp.Embeddings) == 0 {
		logger.Warn("embed_text: empty embeddings returned", "processor", e.name, "item_id", item.ID)
		return nil
	}

	item.SetEmbedding(resp.Embeddings)
	logger.Debug("embed_text: stored embedding", "processor", e.name, "item_id", item.ID, "chunks", len(cleanChunks), "dim", len(resp.Embeddings[0]))
	return nil
}
