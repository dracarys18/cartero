package processors

import (
	"cartero/internal/core"
	"cartero/internal/platforms"
	"context"
	"fmt"
	"github.com/ollama/ollama/api"
	"log"
	"sync"
)

type SummaryProcessor struct {
	name         string
	ollamaClient *platforms.OllamaPlatform
	mu           sync.RWMutex
}

func NewSummaryProcessor(name string, ollamaClient *platforms.OllamaPlatform) *SummaryProcessor {
	if ollamaClient == nil {
		panic("ollamaClient cannot be nil")
	}

	return &SummaryProcessor{
		name:         name,
		mu:           sync.RWMutex{},
		ollamaClient: ollamaClient,
	}
}

func (d *SummaryProcessor) Name() string {
	return d.name
}

func (d *SummaryProcessor) Process(ctx context.Context, item *core.Item) (*core.ProcessedItem, error) {
	processed := &core.ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: item.Metadata,
		Skip:     false,
	}

	req := &api.GenerateRequest{
		Model: "smollm2:135m",
		Prompt: fmt.Sprintf("Summarize this in one short sentence: %s", item.Content),
		Stream: new(bool),
	}

	respFunc := func(resp api.GenerateResponse) error {
		if resp.Done {
			processed.Metadata["summary"] = resp.Response
		}
		return nil
	}

	err := d.ollamaClient.Client().Generate(ctx, req, respFunc)
	if err != nil {
		log.Printf("SummaryProcessor %s: error generating summary: %v", d.name, err)
		return nil, err
	}
	return processed, nil
}
