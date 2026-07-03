package processors

import (
	"cartero/internal/components"
	"cartero/internal/config"
	procnames "cartero/internal/processors/names"
	"cartero/internal/types"
	"context"
	"fmt"

	"github.com/ollama/ollama/api"
)

const Prompt = "You are a professional content summarizer. Please read the following article text carefully and summarize it into concise bullet points. Focus on high-impact information and key takeaways rather than jargon. Keep the summary brief and actionable.\n\nArticle text:\n"

type SummaryProcessor struct {
	name     string
	settings config.SummarySettings
}

func NewSummaryProcessor(name string, settings config.SummarySettings) *SummaryProcessor {
	return &SummaryProcessor{
		name:     name,
		settings: settings,
	}
}

func (d *SummaryProcessor) Name() string {
	return d.name
}

func (d *SummaryProcessor) DependsOn() []string {
	return []string{procnames.ScoreFilter, procnames.ExtractText}
}

func (d *SummaryProcessor) Process(ctx context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	logger := st.GetLogger()
	pc := st.GetRegistry().Get(components.PlatformComponentName).(*components.PlatformComponent)
	ollamaClient := pc.OllamaPlatform(d.settings.Model)

	for _, item := range items {
		var content string
		if article := item.GetArticle(); article != nil {
			content = article.Text
		}

		req := &api.GenerateRequest{
			Prompt: fmt.Sprintf("%s%s", Prompt, content),
			Stream: new(bool),
		}

		var summary string
		respFunc := func(resp api.GenerateResponse) error {
			summary += resp.Response
			return nil
		}

		if err := ollamaClient.Generate(ctx, req, respFunc); err != nil {
			logger.Warn("summary: generation failed, publishing without summary", "processor", d.name, "item_id", item.ID, "error", err)
			continue
		}
		if len(summary) == 0 {
			logger.Warn("summary: empty summary, publishing without summary", "processor", d.name, "item_id", item.ID)
			continue
		}

		item.AddMetadata("summary", summary)
		logger.Debug("summary: generated", "processor", d.name, "item_id", item.ID)
	}

	return items, nil
}
