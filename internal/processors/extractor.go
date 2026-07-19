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
	strutils "cartero/internal/utils/string"
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
	req.Header.Set("X-Return-Format", "text")
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

	return &types.Article{Text: strutils.Truncate(string(body), limit)}, nil
}

func newExtractor(settings config.ExtractTextSettings) Extractor {
	if settings.ExtractType == "jina" {
		return JinaExtractor{url: settings.ReaderURL}
	}
	return ReadabilityExtractor{}
}
