package utils

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/go-shiori/go-readability"
)

func GetArticleText(u string) (string, error) {
	// Validate URL is not empty
	if u == "" {
		log.Printf("GetArticleText: error - URL is empty")
		return "", fmt.Errorf("URL is empty")
	}

	// Validate URL format
	urlParsed, err := url.Parse(u)
	if err != nil {
		log.Printf("GetArticleText: error - invalid URL format '%s': %v", u, err)
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate URL has scheme and host
	if urlParsed.Scheme == "" || urlParsed.Host == "" {
		log.Printf("GetArticleText: error - URL '%s' missing scheme or host", u)
		return "", fmt.Errorf("URL missing scheme or host")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Printf("GetArticleText: error - failed to create request for URL '%s': %v", u, err)
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GetArticleText: error - failed to fetch URL '%s': %v", u, err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("GetArticleText: error - HTTP %s for URL '%s'", resp.Status, u)
		return "", fmt.Errorf("failed to fetch page: %s", resp.Status)
	}

	article, err := readability.FromReader(resp.Body, urlParsed)
	if err != nil {
		log.Printf("GetArticleText: error - failed to extract content from URL '%s': %v", u, err)
		return "", fmt.Errorf("failed to extract content: %v", err)
	}

	return article.TextContent, nil
}
