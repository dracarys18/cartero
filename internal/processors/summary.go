package processors

import (
	"cartero/internal/core"
	"cartero/internal/platforms"
	"cartero/internal/state"
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
	model        string
	ollamaClient *platforms.OllamaPlatform
	mu           sync.RWMutex
}

func NewSummaryProcessor(name string, model string) *SummaryProcessor {
	return &SummaryProcessor{
		name:  name,
		model: model,
		mu:    sync.RWMutex{},
	}
}

func (d *SummaryProcessor) SetState(appState *state.State) {
	d.ollamaClient = appState.Platforms.OllamaPlatform(d.model)
	if d.ollamaClient == nil {
		// Log warning or panic? Original code panicked.
		// Since this is initialization phase (late), panic is acceptable if strictly required.
		log.Printf("SummaryProcessor %s: Warning - Ollama platform not found for model %s", d.name, d.model)
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
