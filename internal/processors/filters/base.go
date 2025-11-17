package filters

import (
	"context"

	"cartero/internal/core"
)

type FilterProcessor struct {
	name       string
	filterFn   func(*core.Item) bool
	skipOnFail bool
}

type FilterFunc func(*core.Item) bool

func NewFilterProcessor(name string, filterFn FilterFunc) *FilterProcessor {
	return &FilterProcessor{
		name:       name,
		filterFn:   filterFn,
		skipOnFail: true,
	}
}

func (f *FilterProcessor) Name() string {
	return f.name
}

func (f *FilterProcessor) Process(ctx context.Context, item *core.Item) (*core.ProcessedItem, error) {
	processed := &core.ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]interface{}),
		Skip:     false,
	}

	if f.filterFn != nil {
		if !f.filterFn(item) {
			processed.Skip = true
			processed.Metadata["filtered"] = true
			processed.Metadata["filter"] = f.name
		}
	}

	return processed, nil
}
