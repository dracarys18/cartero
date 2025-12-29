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

const Prompt = "You are a professional content summarizer. Please read the following article text carefully and summarize it into concise bullet points. Focus on high-impact information and key takeaways rather than jargon. Keep the summary brief and actionable.\n\nArticle text:\n"

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

	log.Printf("SummaryProcessor %s: fetched article content for item %s (%d chars)", d.name, item.ID, len(content))

	prompt := fmt.Sprintf("%s%s", Prompt, content)

	req := &api.GenerateRequest{
		Prompt: prompt,
		Stream: new(bool),
	}

	var summary string
	respFunc := func(resp api.GenerateResponse) error {
		summary += resp.Response
		return nil
	}

	err = d.ollamaClient.Generate(ctx, req, respFunc)
	if err != nil {
		log.Printf("SummaryProcessor %s: warning - couldn't generate summary, publishing without summary: %v", d.name, err)
		return processed, nil
	}

	summary = fmt.Sprintf("%s", summary)
	if len(summary) == 0 {
		log.Printf("SummaryProcessor %s: warning - generated empty summary for item %s, publishing without summary", d.name, item.ID)
		return processed, nil
	}

	processed.Metadata["summary"] = summary
	log.Printf("SummaryProcessor %s: generated summary for item %s: %s", d.name, item.ID, summary)

	return processed, nil
}
