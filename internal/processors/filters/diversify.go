package filters

import (
	"context"

	"cartero/internal/types"
)

type DiversifyFilter struct{}

func NewDiversifyFilter() *DiversifyFilter { return &DiversifyFilter{} }

func (f *DiversifyFilter) Name() string        { return filterDiversify }
func (f *DiversifyFilter) DependsOn() []string { return []string{filterRank} }

func (f *DiversifyFilter) Process(_ context.Context, _ types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	return interleave(groupByTopic(items), len(items)), nil
}

func groupByTopic(items []*types.Item) [][]*types.Item {
	index := make(map[string]int)
	var groups [][]*types.Item
	for _, item := range items {
		topic := item.GetMatchedKeywords()
		i, ok := index[topic]
		if !ok {
			i = len(groups)
			index[topic] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], item)
	}
	return groups
}

func interleave(groups [][]*types.Item, total int) []*types.Item {
	longest := 0
	for _, group := range groups {
		longest = max(longest, len(group))
	}

	out := make([]*types.Item, 0, total)
	for round := range longest {
		for _, group := range groups {
			if round < len(group) {
				out = append(out, group[round])
			}
		}
	}
	return out
}
