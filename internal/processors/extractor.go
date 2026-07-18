package processors

import (
	"context"
	"net/url"
	"time"

	"cartero/internal/config"
	"cartero/internal/types"
	"cartero/internal/utils"
)

type Extractor interface {
	Extract(ctx context.Context, u *url.URL, limit int, timeout time.Duration) (*types.Article, error)
}

type ReadabilityExtractor struct{}

func (ReadabilityExtractor) Extract(ctx context.Context, u *url.URL, limit int, timeout time.Duration) (*types.Article, error) {
	return utils.GetArticle(ctx, u, limit, timeout)
}

type JinaExtractor struct {
	url string
}

func (j JinaExtractor) Extract(ctx context.Context, u *url.URL, limit int, timeout time.Duration) (*types.Article, error) {
	text, err := utils.GetArticleViaReader(ctx, j.url, u, limit, timeout)
	if err != nil {
		return nil, err
	}
	return &types.Article{Text: text}, nil
}

func newExtractor(settings config.ExtractTextSettings) Extractor {
	if settings.ExtractType == "jina" {
		return JinaExtractor{url: settings.ReaderURL}
	}
	return ReadabilityExtractor{}
}
