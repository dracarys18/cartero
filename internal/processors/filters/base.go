package filters

import (
	"context"

	"cartero/internal/core"
)

type FilterProcessor struct {
	name     string
	filterFn func(*core.Item) bool
}

type FilterFunc func(*core.Item) bool

func NewFilterProcessor(name string, filterFn FilterFunc) *FilterProcessor {
	return &FilterProcessor{
		name:     name,
		filterFn: filterFn,
	}
}

func (f *FilterProcessor) Name() string {
	return f.name
}

func (f *FilterProcessor) Process(ctx context.Context, item *core.Item) (*core.ProcessedItem, error) {
	if f.filterFn != nil {
		if !f.filterFn(item) {
			// Filter rejected the item - return nil to signal filtering
			return nil, nil
		}
	}
	// Filter accepted the item - return it unchanged
	return &core.ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: item.Metadata,
	}, nil
}
