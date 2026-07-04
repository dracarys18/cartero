package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"cartero/internal/types"
	strutils "cartero/internal/utils/string"

	readability "codeberg.org/readeck/go-readability/v2"
	"codeberg.org/readeck/go-readability/v2/render"
	"github.com/enetx/surf"
)

func GetArticle(ctx context.Context, u string, limit int, timeout time.Duration, mod ...readability.RequestWith) (*types.Article, error) {
	if limit <= 0 {
		limit = 20000
	}
	if u == "" {
		return nil, fmt.Errorf("URL is empty")
	}

	parsedUrl, err := url.Parse(u)

	if err != nil {
		return nil, err
	}

	surfClient := surf.NewClient().
		Builder().
		Impersonate().Firefox().
		Timeout(timeout).
		Session().
		Build().
		Unwrap()

	client := surfClient.Std()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	article, err := readability.FromReader(resp.Body, parsedUrl)

	if err != nil || article.Node == nil {
		return nil, fmt.Errorf("failed to extract content: %v", err)
	}

	textContent := render.InnerText(article.Node)

	textContent = strutils.Truncate(textContent, limit)

	res := &types.Article{
		Text:        textContent,
		Image:       article.ImageURL(),
		Description: strutils.Clean(article.Excerpt()),
	}

	return res, nil
}
