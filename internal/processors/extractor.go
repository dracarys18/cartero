package processors

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cartero/internal/config"
	"cartero/internal/types"
	"cartero/internal/utils"
)

type Extractor interface {
	Extract(ctx context.Context, u *url.URL, timeout time.Duration) (*types.Article, error)
}

type ReadabilityExtractor struct{}

func (ReadabilityExtractor) Extract(ctx context.Context, u *url.URL, timeout time.Duration) (*types.Article, error) {
	return utils.GetArticle(ctx, u, timeout)
}

type JinaExtractor struct {
	url string
}

func (j JinaExtractor) Extract(ctx context.Context, u *url.URL, timeout time.Duration) (*types.Article, error) {
	if j.url == "" {
		return nil, fmt.Errorf("reader URL is empty")
	}
	if u == nil || u.String() == "" {
		return nil, fmt.Errorf("URL is empty")
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	endpoint := strings.TrimRight(j.url, "/") + "/" + u.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Return-Format", "markdown")
	req.Header.Set("X-Retain-Images", "none")
	if timeout > 0 {
		req.Header.Set("X-Timeout", strconv.Itoa(int(timeout.Seconds())))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reader request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reader returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &types.Article{Text: string(body)}, nil
}

type TieredExtractor struct {
	primary  Extractor
	fallback Extractor
	minLen   int
}

func (t TieredExtractor) Extract(ctx context.Context, u *url.URL, timeout time.Duration) (*types.Article, error) {
	article, err := t.primary.Extract(ctx, u, timeout)
	if err == nil && article != nil && len(article.Text) >= t.minLen {
		return article, nil
	}

	if t.fallback != nil {
		if rendered, rerr := t.fallback.Extract(ctx, u, timeout); rerr == nil && rendered != nil && len(rendered.Text) > 0 {
			return rendered, nil
		}
	}

	if article != nil {
		return article, nil
	}
	return nil, err
}

func newExtractor(settings config.ExtractTextSettings) Extractor {
	var fallback Extractor
	if settings.ReaderURL != "" {
		fallback = JinaExtractor{url: settings.ReaderURL}
	}
	return TieredExtractor{
		primary:  ReadabilityExtractor{},
		fallback: fallback,
		minLen:   settings.MinContentLength,
	}
}
