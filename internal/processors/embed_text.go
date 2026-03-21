package processors

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/textsplitter"

	"cartero/internal/components"
	procnames "cartero/internal/processors/names"
	"cartero/internal/types"

	"github.com/ollama/ollama/api"
)

const taskDescription = "Represent this sentence for searching relevant passages: "

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

	chunks := []string{item.GetTitle(), description}

	chunkSize := cfg.ChunkSize
	if chunkSize == 0 {
		chunkSize = 400
	}

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(chunkSize),
		textsplitter.WithChunkOverlap(chunkSize/8),
	)

	bodyChunk, err := splitter.SplitText(body)
	chunks = append(chunks, bodyChunk...)

	if err != nil || len(chunks) == 0 {
		logger.Warn("embed_text: failed to split text", "processor", e.name, "item_id", item.ID, "error", err)
		return nil
	}
	e.AppendToQuery(&chunks)

	registry := st.GetRegistry()
	pc := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	embedder := pc.Embedder()
	if embedder == nil {
		return nil
	}

	resp, err := embedder.Embed(ctx, &api.EmbedRequest{Input: chunks})
	if err != nil {
		logger.Warn("embed_text: failed to embed item", "processor", e.name, "item_id", item.ID, "error", err)
		return nil
	}

	if len(resp.Embeddings) == 0 {
		logger.Warn("embed_text: empty embeddings returned", "processor", e.name, "item_id", item.ID)
		return nil
	}

	item.SetEmbedding(resp.Embeddings)
	logger.Debug("embed_text: stored embedding", "processor", e.name, "item_id", item.ID, "chunks", len(chunks), "dim", len(resp.Embeddings[0]))
	return nil
}

func (e *EmbedTextProcessor) AppendToQuery(chunks *[]string) {
	for i, c := range *chunks {
		(*chunks)[i] = fmt.Sprintf("Instruct: %s\nQuery: %s", taskDescription, c)
	}
}
