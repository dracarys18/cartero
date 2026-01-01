package processors

import (
	"cartero/internal/components"
	"cartero/internal/core"
	"cartero/internal/platforms"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ollama/ollama/api"
)

const Prompt = "You are a professional content summarizer. Please read the following article text carefully and summarize it into concise bullet points. Focus on high-impact information and key takeaways rather than jargon. Keep the summary brief and actionable.\n\nArticle text:\n"

type SummaryProcessor struct {
	name         string
	model        string
	ollamaClient *platforms.OllamaPlatform
	mu           sync.RWMutex
}

func NewSummaryProcessor(name string, model string, registry *components.Registry) *SummaryProcessor {
	pc := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	return &SummaryProcessor{
		name:         name,
		model:        model,
		mu:           sync.RWMutex{},
		ollamaClient: pc.OllamaPlatform(model),
	}
}

func (d *SummaryProcessor) Name() string {
	return d.name
}

func (d *SummaryProcessor) Process(ctx context.Context, item *core.Item) error {
	content := item.GetTextContent()
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

	err := d.ollamaClient.Generate(ctx, req, respFunc)
	if err != nil {
		log.Printf("SummaryProcessor %s: warning - couldn't generate summary, publishing without summary: %v", d.name, err)
		return nil
	}

	if len(summary) == 0 {
		log.Printf("SummaryProcessor %s: warning - generated empty summary for item %s, publishing without summary", d.name, item.ID)
		return nil
	}

	if err := item.AddMetadata("summary", summary); err != nil {
		return err
	}
	log.Printf("SummaryProcessor %s: generated summary for item %s: %s", d.name, item.ID, summary)

	return nil
}
