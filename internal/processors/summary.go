package processors

import (
	"cartero/internal/components"
	procnames "cartero/internal/processors/names"
	"cartero/internal/types"
	"context"
	"fmt"
	"sync"

	"github.com/ollama/ollama/api"
)

const Prompt = "You are a professional content summarizer. Please read the following article text carefully and summarize it into concise bullet points. Focus on high-impact information and key takeaways rather than jargon. Keep the summary brief and actionable.\n\nArticle text:\n"

type SummaryProcessor struct {
	name string
	mu   sync.RWMutex
}

func NewSummaryProcessor(name string, registry *components.Registry) *SummaryProcessor {
	return &SummaryProcessor{
		name: name,
		mu:   sync.RWMutex{},
	}
}

func (d *SummaryProcessor) Name() string {
	return d.name
}

func (d *SummaryProcessor) DependsOn() []string {
	return []string{procnames.ScoreFilter, procnames.KeywordFilter, procnames.ExtractText}
}

func (d *SummaryProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[d.name].Settings.SummarySettings
	model := cfg.Model

	registry := st.GetRegistry()
	pc := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	ollamaClient := pc.OllamaPlatform(model)

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

	logger := st.GetLogger()
	err := ollamaClient.Generate(ctx, req, respFunc)
	if err != nil {
		logger.Warn("Couldn't generate summary, publishing without summary", "processor", d.name, "error", err)
		return nil
	}

	if len(summary) == 0 {
		logger.Warn("Generated empty summary for item, publishing without summary", "processor", d.name, "item_id", item.ID)
		return nil
	}

	if err := item.AddMetadata("summary", summary); err != nil {
		return err
	}
	logger.Debug("Generated summary for item", "processor", d.name, "item_id", item.ID, "summary", summary)

	return nil
}
