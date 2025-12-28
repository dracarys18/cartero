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

func (f *FilterProcessor) ShouldProcess(ctx context.Context, item *core.Item) (bool, error) {
	if f.filterFn != nil {
		return f.filterFn(item), nil
	}
	return true, nil
}
