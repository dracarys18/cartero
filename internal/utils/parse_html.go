package utils

import (
	"fmt"
	"time"

	"cartero/internal/types"
	readability "codeberg.org/readeck/go-readability/v2"
	"codeberg.org/readeck/go-readability/v2/render"
)

func GetArticle(u string, limit int, mod ...readability.RequestWith) (*types.Article, error) {
	if limit <= 0 {
		limit = 20000
	}
	if u == "" {
		return nil, fmt.Errorf("URL is empty")
	}

	article, err := readability.FromURL(u, 30*time.Second)

	if err != nil || article.Node == nil {
		return nil, fmt.Errorf("failed to extract content: %v", err)
	}

	textContent := render.InnerText(article.Node)

	if len(textContent) > limit {
		textContent = textContent[:limit] + "..."
	}

	res := &types.Article{
		Text:  textContent,
		Image: article.ImageURL(),
	}

	return res, nil
}
