package utils

import (
	"fmt"
	"net/url"
	"time"

	"cartero/internal/types"

	readability "codeberg.org/readeck/go-readability/v2"
	"codeberg.org/readeck/go-readability/v2/render"
	"github.com/enetx/surf"
)

func GetArticle(u string, limit int, mod ...readability.RequestWith) (*types.Article, error) {
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
		Timeout(30 * time.Second).
		Session().
		Build().
		Unwrap()

	client := surfClient.Std()
	resp, err := client.Get(u)

	defer resp.Body.Close()

	if err != nil {
		fmt.Println("Error reading body:", err)
	}

	article, err := readability.FromReader(resp.Body, parsedUrl)

	if err != nil || article.Node == nil {
		return nil, fmt.Errorf("failed to extract content: %v", err)
	}

	textContent := render.InnerText(article.Node)

	if len(textContent) > limit {
		textContent = textContent[:limit] + "..."
	}

	res := &types.Article{
		Text:        textContent,
		Image:       article.ImageURL(),
		Description: article.Excerpt(),
	}

	return res, nil
}
