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

func (e *ExtractText) DependsOn() []string {
	return []string{
		names.ScoreFilter,
	}
}

func (e *ExtractText) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[e.name].Settings.ExtractTextSettings
	limit := cfg.Limit
	minContentLength := cfg.MinContentLength
	logger := st.GetLogger()

	url, exists := item.GetMetadata("url")
	if !exists {
		return nil
	}

	urlStr, ok := url.(string)
	if !ok {
		logger.Info("ExtractText processor rejected item", "processor", e.name, "item_id", item.ID, "reason", "url metadata is not a string")
		return types.NewFilteredError(e.name, item.ID, "url metadata is not a string")
	}

	httpMod := utils.BrowserHeadersModifier()

	article, err := utils.GetArticle(urlStr, limit, httpMod)
	if err != nil {
		logger.Error("ExtractText processor failed to extract article text", "processor", e.name, "item_id", item.ID, "error", err)
		return nil
	}

	if len(article.Text) >= minContentLength {
		item.SetArticle(article)
	}

	return nil
}
