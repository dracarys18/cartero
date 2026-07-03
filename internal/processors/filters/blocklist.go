package filters

import (
	"context"

	"cartero/internal/processors/names"
	"cartero/internal/types"
)

type BlocklistFilter struct{}

func NewBlocklistFilter() *BlocklistFilter {
	return &BlocklistFilter{}
}

func (f *BlocklistFilter) Name() string        { return names.Blocklist }
func (f *BlocklistFilter) DependsOn() []string { return nil }

func (f *BlocklistFilter) Process(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	bl := state.GetBlocklist()
	if bl == nil {
		return items, nil
	}

	logger := state.GetLogger()
	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		if bl.Blocked(ctx, item.GetLink()) {
			logger.Debug("blocklist: dropped item", "item_id", item.ID, "link", item.GetLink())
			continue
		}
		out = append(out, item)
	}
	return out, nil
}
