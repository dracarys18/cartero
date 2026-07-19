package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"cartero/internal/types"
	strutils "cartero/internal/utils/string"

	"github.com/enetx/surf"
	"github.com/markusmobius/go-trafilatura"
)

func GetArticle(ctx context.Context, u *url.URL, limit int, timeout time.Duration) (*types.Article, error) {
	if u == nil || u.String() == "" {
		return nil, fmt.Errorf("URL is empty")
	}

	surfClient := surf.NewClient().
		Builder().
		Impersonate().Firefox().
		Timeout(timeout).
		Session().
		Build().
		Unwrap()

	client := surfClient.Std()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	result, err := trafilatura.Extract(resp.Body, trafilatura.Options{
		OriginalURL:     u,
		EnableFallback:  true,
		ExcludeComments: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to extract content: %w", err)
	}

	return &types.Article{
		Text:        strutils.Truncate(result.ContentText, limit),
		Image:       result.Metadata.Image,
		Description: strutils.Clean(result.Metadata.Description),
	}, nil
}
