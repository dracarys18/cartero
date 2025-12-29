package utils

import (
	"fmt"
	"log"
	"time"

	"github.com/go-shiori/go-readability"
)

func GetArticleText(u string) (string, error) {
	// Validate URL is not empty
	if u == "" {
		log.Printf("GetArticleText: error - URL is empty")
		return "", fmt.Errorf("URL is empty")
	}

	article, err := readability.FromURL(u, 30*time.Second)
	if err != nil {
		log.Printf("GetArticleText: error - failed to extract content from URL '%s': %v", u, err)
		return "", fmt.Errorf("failed to extract content: %v", err)
	}

	log.Printf("GetArticleText: extraction complete for URL '%s'", u)
	log.Printf("GetArticleText: article title: '%s'", article.Title)
	log.Printf("GetArticleText: content length: %d characters", len(article.TextContent))

	if len(article.TextContent) > 0 {
		// Log first 500 chars of extracted content for debugging
		contentPreview := article.TextContent
		if len(contentPreview) > 500 {
			contentPreview = contentPreview[:500]
		}
		log.Printf("GetArticleText: first 500 chars of extracted content:\n%s\n---END PREVIEW---", contentPreview)
	}

	if len(article.TextContent) > 4000 {
		log.Printf("GetArticleText: truncating content from %d to 4000 characters", len(article.TextContent))
		article.TextContent = article.TextContent[:4000] + "..."
	}

	log.Printf("GetArticleText: returning %d characters for URL '%s'", len(article.TextContent), u)
	return article.TextContent, nil
}
