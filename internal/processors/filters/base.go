package filters

import (
	"context"
	"fmt"

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

func (f *FilterProcessor) Process(ctx context.Context, item *core.Item) error {
	if f.filterFn != nil {
		if !f.filterFn(item) {
			// Filter rejected the item - return error to signal filtering
			return fmt.Errorf("filter %s rejected item", f.name)
		}
	}
	// Filter accepted the item - continue processing
	return nil
}
