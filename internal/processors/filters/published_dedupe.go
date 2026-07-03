package filters

import (
	"context"

	"cartero/internal/types"
)

type PublishedDedupeFilter struct {
	targets []string
}

func NewPublishedDedupeFilter(targets []string) *PublishedDedupeFilter {
	return &PublishedDedupeFilter{targets: targets}
}

func (f *PublishedDedupeFilter) Name() string        { return filterPublishedDedupe }
func (f *PublishedDedupeFilter) DependsOn() []string { return []string{filterBlocklist} }

func (f *PublishedDedupeFilter) Filter(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	if len(f.targets) == 0 {
		return items, nil
	}
	store := state.GetStorage()
	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		delivered := true
		for _, target := range f.targets {
			published, _ := store.Entries().IsPublished(ctx, item.ID, target)
			if !published {
				delivered = false
				break
			}
		}
		if !delivered {
			out = append(out, item)
		}
	}
	return out, nil
}
