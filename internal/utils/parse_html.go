package utils

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"cartero/internal/types"
	strutils "cartero/internal/utils/string"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/enetx/surf"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
)

func GetArticle(ctx context.Context, u *url.URL, timeout time.Duration) (*types.Article, error) {
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

	var buf bytes.Buffer
	if err := html.Render(&buf, result.ContentNode); err != nil {
		return nil, fmt.Errorf("failed to render content: %w", err)
	}
	markdown, err := md.NewConverter(u.Hostname(), true, nil).ConvertString(buf.String())
	if err != nil {
		return nil, fmt.Errorf("failed to convert content to markdown: %w", err)
	}

	return &types.Article{
		Text:        markdown,
		Image:       result.Metadata.Image,
		Description: strutils.Clean(result.Metadata.Description),
	}, nil
}
