package processors

import (
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils"
	"context"
	"net/http"
)

type ExtractText struct {
	name string
}

func NewExtractProcessor(name string) *ExtractText {
	return &ExtractText{
		name: name,
	}
}

func (e *ExtractText) Name() string {
	return e.name
}

func (e *ExtractText) DependsOn() []string {
	return []string{
		names.ScoreFilter,
	}
}

func (e *ExtractText) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[e.name].Settings.ExtractTextSettings
	limit := cfg.Limit
	logger := st.GetLogger()

	url, exists := item.GetMetadata("url")
	if !exists {
		return nil
	}

	urlStr, ok := url.(string)
	if !ok {
		logger.Info("ExtractText processor rejected item", "processor", e.name, "item_id", item.ID, "reason", "url metadata is not a string")
		return types.NewFilteredError(e.name, item.ID, "url metadata is not a string")
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

	article, err := utils.GetArticleText(urlStr, limit, httpMod)
	if err != nil {
		logger.Error("ExtractText processor failed to extract article text", "processor", e.name, "item_id", item.ID, "error", err)
		return nil
	}

	if err := item.SetTextContent(article); err != nil {
		return err
	}

	return nil
}
