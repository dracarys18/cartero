package extract

import (
	"cartero/internal/core"
	"cartero/internal/utils"
	"context"
	"fmt"
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
	return []string{}
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

	article, err := utils.GetArticleText(urlStr)
	if err != nil {
		return fmt.Errorf("failed to extract article text: %w", err)
	}

	if err := item.SetTextContent(article); err != nil {
		return err
	}

	return nil
}
