package processors

import (
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils"
	"context"
)

type ExtractText struct {
	name string
}

func NewExtractProcessor(name string) *ExtractText {
	return &ExtractText{
		name: name,
	}
}

func (e *ExtractText) Name() string {
	return e.name
}

func (e *ExtractText) Initialize(_ context.Context, _ types.StateAccessor) error {
	return nil
}

func (e *ExtractText) DependsOn() []string {
	return []string{
		names.Dedupe,
		names.ScoreFilter,
	}
}

func (e *ExtractText) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[e.name].Settings.ExtractTextSettings
	limit := cfg.Limit
	minContentLength := cfg.MinContentLength
	logger := st.GetLogger()

	urlStr := item.GetURL()
	if urlStr == "" {
		return nil
	}

	article, err := utils.GetArticle(urlStr, limit)
	if err != nil {
		logger.Error("ExtractText processor failed to extract article text", "processor", e.name, "item_id", item.ID, "error", err)
		return nil
	}

	if len(article.Text) >= minContentLength {
		item.SetArticle(article)
	}

	return nil
}
