package utils

import (
	"fmt"
	"log"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	"codeberg.org/readeck/go-readability/v2/render"
)

func GetArticleText(u string, limit int, mod ...readability.RequestWith) (string, error) {
	if limit <= 0 {
		limit = 20000
	}
	if u == "" {
		return "", fmt.Errorf("URL is empty")
	}

	article, err := readability.FromURL(u, 30*time.Second)

	if err != nil || article.Node == nil {
		return "", fmt.Errorf("failed to extract content: %v", err)
	}

	textContent := render.InnerText(article.Node)

	if len(textContent) > limit {
		log.Printf("GetArticleText: truncating content from %d to 4000 characters", len(textContent))
		textContent = textContent[:limit] + "..."
	}

	log.Printf("GetArticleText: returning %d characters for URL '%s'", len(textContent), u)
	return textContent, nil
}
