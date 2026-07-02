package filters

import (
	"context"

	"cartero/internal/types"
)

const mmrLambda = 0.7

type DiversifyFilter struct{}

func NewDiversifyFilter() *DiversifyFilter { return &DiversifyFilter{} }

func (f *DiversifyFilter) Name() string        { return "diversify" }
func (f *DiversifyFilter) DependsOn() []string { return []string{"rank", "rerank"} }

func (f *DiversifyFilter) Filter(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	if len(items) < 2 {
		return items, nil
	}

	selected := make([]*types.Item, 0, len(items))
	remaining := make([]*types.Item, len(items))
	copy(remaining, items)

	for len(remaining) > 0 {
		bestIdx := 0
		bestMMR := -1e18
		for i, item := range remaining {
			var maxSim float64
			for _, sel := range selected {
				if sim := cosine(repVector(item), repVector(sel)); sim > maxSim {
					maxSim = sim
				}
			}
			mmr := mmrLambda*getScore(item) - (1-mmrLambda)*maxSim
			if mmr > bestMMR {
				bestMMR = mmr
				bestIdx = i
			}
		}
		selected = append(selected, remaining[bestIdx])
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}
	return selected, nil
}

func repVector(item *types.Item) []float32 {
	if e := item.GetEmbedding(); len(e) > 0 {
		return e[0]
	}
	return nil
}
