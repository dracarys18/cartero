package utils

import (
	"fmt"
	"github.com/go-shiori/go-readability"
	"net/http"
	"net/url"
	"time"
)

func GetArticleText(u string) (string, error) {
	urlParsed, err := url.Parse(u)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch page: %s", resp.Status)
	}

	article, err := readability.FromReader(resp.Body, urlParsed)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %v", err)
	}

	if len(article.TextContent) > 4000 {
		article.TextContent = article.TextContent[:4000] + "..."
	}
	return article.TextContent, nil
}
