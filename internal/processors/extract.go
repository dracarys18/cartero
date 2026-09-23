package processors

import (
	"cartero/internal/config"
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils/batch"
	"context"
	"time"
)

const (
	defaultExtractConcurrency = 12
	defaultExtractTimeout     = 8 * time.Second
)

type ExtractText struct {
	settings  config.ExtractTextSettings
	extractor Extractor
}

func NewExtractProcessor(settings config.ExtractTextSettings) *ExtractText {
	return &ExtractText{settings: settings, extractor: newExtractor(settings)}
}

func (e *ExtractText) Name() string {
	return names.ExtractText
}

func (e *ExtractText) DependsOn() []string {
	return []string{
		names.Dedupe,
		names.ScoreFilter,
		names.PublishedAt,
	}
}

func (e *ExtractText) Process(ctx context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	concurrency := e.settings.Concurrency
	if concurrency <= 0 {
		concurrency = defaultExtractConcurrency
	}

	keep := make([]bool, len(items))
	idx := make([]int, len(items))
	for i := range idx {
		idx[i] = i
	}

	batch.Run(ctx, idx, concurrency, func(ctx context.Context, i int) {
		keep[i] = e.extract(ctx, st, items[i])
	})

	out := make([]*types.Item, 0, len(items))
	for i, item := range items {
		if keep[i] {
			out = append(out, item)
		}
	}
	return out, nil
}

func (e *ExtractText) extract(ctx context.Context, st types.StateAccessor, item *types.Item) bool {
	logger := st.GetLogger()

	rejected := st.GetRejected()
	if rejected != nil {
		if skip, err := rejected.Has(ctx, item.ID); err == nil && skip {
			return false
		}
	}

	u := item.GetURL()
	if u == nil || u.String() == "" {
		return true
	}

	timeout := time.Duration(e.settings.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultExtractTimeout
	}

	article, err := e.extractor.Extract(ctx, u, timeout)
	if err != nil {
		logger.Error("ExtractText processor failed to extract article text", "processor", names.ExtractText, "item_id", item.ID, "error", err)
		if rejected != nil {
			_ = rejected.Add(ctx, item.ID)
		}
		return true
	}

	if len(article.Text) >= e.settings.MinContentLength {
		item.SetArticle(article)
	}
	return true
}
