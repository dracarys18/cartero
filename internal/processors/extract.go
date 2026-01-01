package processors

import (
	"cartero/internal/core"
	"cartero/internal/utils"
	"context"
	"fmt"
	"net/http"
)

type ExtractText struct {
	name  string
	limit int
}

func NewExtractProcessor(name string, limit int) *ExtractText {
	return &ExtractText{
		name:  name,
		limit: limit,
	}
}

func (e *ExtractText) Name() string {
	return e.name
}

func (e *ExtractText) Process(ctx context.Context, item *core.Item) error {
	url, exists := item.GetMetadata("url")
	if !exists {
		return nil
	}

	urlStr, ok := url.(string)
	if !ok {
		return fmt.Errorf("url metadata is not a string")
	}

	httpMod := func(req *http.Request) {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:146.0) Gecko/20100101 Firefox/146.0")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "cross-site")
	}

	article, err := utils.GetArticleText(urlStr, e.limit, httpMod)
	if err != nil {
		return fmt.Errorf("failed to extract article text: %w", err)
	}

	if err := item.SetTextContent(article); err != nil {
		return err
	}

	return nil
}
