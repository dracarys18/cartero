package filters

import (
	"context"

	"cartero/internal/types"
)

const publishLimit = 10

type LimitFilter struct{}

func NewLimitFilter() *LimitFilter { return &LimitFilter{} }

func (f *LimitFilter) Name() string        { return "limit" }
func (f *LimitFilter) DependsOn() []string { return []string{"diversify"} }

func (f *LimitFilter) Filter(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	if len(items) > publishLimit {
		return items[:publishLimit], nil
	}
	return items, nil
}
