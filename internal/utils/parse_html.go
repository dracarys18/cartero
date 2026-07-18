package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cartero/internal/types"
	strutils "cartero/internal/utils/string"

	readability "codeberg.org/readeck/go-readability/v2"
	"codeberg.org/readeck/go-readability/v2/render"
	"github.com/enetx/surf"
)

func GetArticle(ctx context.Context, u *url.URL, limit int, timeout time.Duration, mod ...readability.RequestWith) (*types.Article, error) {
	if limit <= 0 {
		limit = 20000
	}
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

	article, err := readability.FromReader(resp.Body, u)

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

func GetArticleViaReader(ctx context.Context, readerURL string, u *url.URL, limit int, timeout time.Duration) (string, error) {
	if readerURL == "" {
		return "", fmt.Errorf("reader URL is empty")
	}
	if u == nil || u.String() == "" {
		return "", fmt.Errorf("URL is empty")
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	endpoint := strings.TrimRight(readerURL, "/") + "/" + u.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Return-Format", "text")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("reader request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reader returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strutils.Truncate(string(body), limit), nil
}
