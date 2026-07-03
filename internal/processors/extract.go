package processors

import (
	"cartero/internal/config"
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils"
	"cartero/internal/utils/batch"
	"cartero/internal/utils/hash"
	"context"
	"time"
)

const (
	defaultExtractConcurrency = 12
	defaultExtractTimeout     = 8 * time.Second
)

type ExtractText struct {
	settings config.ExtractTextSettings
}

func NewExtractProcessor(settings config.ExtractTextSettings) *ExtractText {
	return &ExtractText{settings: settings}
}

func (e *ExtractText) Name() string {
	return names.ExtractText
}

func (e *ExtractText) DependsOn() []string {
	return []string{
		names.Dedupe,
		names.ScoreFilter,
	}
}

func (e *ExtractText) Process(ctx context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	concurrency := e.settings.Concurrency
	if concurrency <= 0 {
		concurrency = defaultExtractConcurrency
	}

	batch.Run(ctx, items, concurrency, func(ctx context.Context, item *types.Item) {
		e.extract(ctx, st, item)
	})

	return items, nil
}

func (e *ExtractText) extract(ctx context.Context, st types.StateAccessor, item *types.Item) {
	if seen := st.GetSeenStore(); seen != nil {
		_ = seen.Mark(ctx, hash.HashURL(item.GetLink()))
	}

	logger := st.GetLogger()

	urlStr := item.GetURL()
	if urlStr == "" {
		return
	}

	timeout := time.Duration(e.settings.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultExtractTimeout
	}

	article, err := utils.GetArticle(ctx, urlStr, e.settings.Limit, timeout)
	if err != nil {
		logger.Error("ExtractText processor failed to extract article text", "processor", names.ExtractText, "item_id", item.ID, "error", err)
		return
	}

	if len(article.Text) >= e.settings.MinContentLength {
		item.SetArticle(article)
	}
}
