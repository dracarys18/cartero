package utils

import (
	"fmt"
	"log"
	"time"

	"github.com/go-shiori/go-readability"
)

func GetArticleText(u string) (string, error) {
	if u == "" {
		return "", fmt.Errorf("URL is empty")
	}

	article, err := readability.FromURL(u, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %v", err)
	}

	if len(article.TextContent) > 10000 {
		log.Printf("GetArticleText: truncating content from %d to 10000 characters", len(article.TextContent))
		article.TextContent = article.TextContent[:10000] + "..."
	}

	log.Printf("GetArticleText: returning %d characters for URL '%s'", len(article.TextContent), u)
	return article.TextContent, nil
}
