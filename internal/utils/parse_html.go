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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:146.0) Gecko/20100101 Firefox/146.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

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

	log.Printf("GetArticleText: successfully extracted with char characters length %d", len(article.TextContent))
	return article.TextContent, nil
}
