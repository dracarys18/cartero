package processors

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"cartero/internal/config"
	"cartero/internal/types"
	"cartero/internal/utils"
)

const defaultResolver = "1.1.1.1:53"

type Extractor interface {
	Extract(ctx context.Context, u *url.URL, timeout time.Duration) (*types.Article, error)
}

type ReadabilityExtractor struct {
	resolver string
}

func (r ReadabilityExtractor) Extract(ctx context.Context, u *url.URL, timeout time.Duration) (*types.Article, error) {
	return utils.GetArticle(ctx, u, timeout, r.resolver)
}

type TieredExtractor struct {
	primary  Extractor
	renderer Extractor
	minLen   int
}

func (t TieredExtractor) Extract(ctx context.Context, u *url.URL, timeout time.Duration) (*types.Article, error) {
	if u == nil || u.String() == "" {
		return nil, fmt.Errorf("URL is empty")
	}

	article, err := t.primary.Extract(ctx, u, timeout)
	if err == nil && article != nil && len(article.Text) >= t.minLen {
		return article, nil
	}

	var blocked *utils.AntibotError
	if t.renderer != nil && errors.As(err, &blocked) {
		rendered, rerr := t.renderer.Extract(ctx, u, timeout)
		if rerr == nil && rendered != nil {
			return rendered, nil
		}
	}

	if article != nil {
		return article, nil
	}
	return nil, err
}

func newExtractor(settings config.ExtractTextSettings) (Extractor, error) {
	var renderer Extractor
	if settings.FlareSolverrURL != "" {
		flare, err := NewFlareSolverrExtractor(
			settings.FlareSolverrURL,
			time.Duration(settings.FlareTimeoutMS)*time.Millisecond,
		)
		if err != nil {
			return nil, err
		}
		renderer = flare
	}
	resolver := settings.Resolver
	if resolver == "" {
		resolver = defaultResolver
	}
	return TieredExtractor{
		primary:  ReadabilityExtractor{resolver: resolver},
		renderer: renderer,
		minLen:   settings.MinContentLength,
	}, nil
}
