package processors

import (
	"cartero/internal/core"
	"cartero/internal/platforms"
	"cartero/internal/utils"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ollama/ollama/api"
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
	}

	urlValue, exists := item.Metadata["url"]
	if !exists {
		log.Printf("SummaryProcessor %s: warning - no URL in metadata for item %s, publishing without summary", d.name, item.ID)
		return processed, nil
	}

	content, err := utils.GetArticleText(urlValue.(string))
	if err != nil {
		log.Printf("SummaryProcessor %s: warning - couldn't get article text for item %s, publishing without summary: %v", d.name, item.ID, err)
		return processed, nil
	}

	prompt := fmt.Sprintf(`<|im_start|>system
You are a professional news editor. Provide a single, information-dense sentence that summarizes the main event. Avoid fluff like "This article is about."<|im_end|>
<|im_start|>user
Article Content:
"""
%s
"""

Short Summary:<|im_end|>
<|im_start|>assistant`, content)

	req := &api.GenerateRequest{
		Model:  "qwen2.5:0.5b",
		Prompt: prompt,
		Stream: new(bool),
	}

	respFunc := func(resp api.GenerateResponse) error {
		if resp.Done {
			processed.Metadata["summary"] = resp.Response
		}
		return nil
	}

	err = d.ollamaClient.Client().Generate(ctx, req, respFunc)
	if err != nil {
		log.Printf("SummaryProcessor %s: warning - couldn't generate summary, publishing without summary: %v", d.name, err)
		return processed, nil
	}

	return processed, nil
}
