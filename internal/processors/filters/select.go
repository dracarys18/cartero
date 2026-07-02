package filters

import (
	"context"
	"sort"

	"cartero/internal/platforms"
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils/keywords"
)

const (
	scoreKey    = "_score"
	interestKey = "_interest"

	mmrLambda    = 0.7
	publishLimit = 10
)

func setScore(item *types.Item, s float64) { item.AddMetadata(scoreKey, s) }

func getScore(item *types.Item) float64 {
	if v, ok := item.Metadata[scoreKey].(float64); ok {
		return v
	}
	return 0
}

func repVector(item *types.Item) []float32 {
	if e := item.GetEmbedding(); len(e) > 0 {
		return e[0]
	}
	return nil
}

// PublishedDedupeFilter drops items already delivered to all target(s), so the
// list stays "current stories minus what was already sent".
type PublishedDedupeFilter struct {
	targets []string
}

func NewPublishedDedupeFilter(targets []string) *PublishedDedupeFilter {
	return &PublishedDedupeFilter{targets: targets}
}

func (f *PublishedDedupeFilter) Name() string       { return "published_dedupe" }
func (f *PublishedDedupeFilter) DependsOn() []string { return nil }

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

// RankFilter is the cheap recall stage: it scores each item by its single
// best-matching interest (max cosine over chunks), tags that interest for the
// reranker, and sorts. Precision is left to the rerank filter.
type RankFilter struct {
	embedder  platforms.Embedder
	source    []keywords.KeywordWithContext
	interests []Interest
	ready     bool
}

func NewRankFilter(embedder platforms.Embedder, source []keywords.KeywordWithContext) *RankFilter {
	return &RankFilter{embedder: embedder, source: source}
}

func (f *RankFilter) Name() string       { return "rank" }
func (f *RankFilter) DependsOn() []string { return []string{names.EmbedText} }

func (f *RankFilter) Filter(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	if !f.ready {
		interests, err := BuildInterests(ctx, f.embedder, f.source)
		if err != nil {
			return nil, err
		}
		f.interests = interests
		f.ready = true
		state.GetLogger().Info("rank: interests ready", "count", len(interests))
	}
	if len(f.interests) == 0 {
		return items, nil
	}

	mean := batchMean(items)
	ivecs := make([][]float32, len(f.interests))
	for i, in := range f.interests {
		ivecs[i] = centered(in.Vector, mean)
	}

	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		raw := item.GetEmbedding()
		if len(raw) == 0 {
			continue
		}
		chunks := make([][]float32, len(raw))
		for i, ch := range raw {
			chunks[i] = centered(ch, mean)
		}

		best := -1.0
		bestIdx := 0
		for i, iv := range ivecs {
			for _, chunk := range chunks {
				if s := cosine(iv, chunk); s > best {
					best = s
					bestIdx = i
				}
			}
		}
		setScore(item, best)
		item.AddMetadata(interestKey, f.interests[bestIdx].Lexical)
		out = append(out, item)
	}

	sort.SliceStable(out, func(i, j int) bool { return getScore(out[i]) > getScore(out[j]) })
	return out, nil
}

func batchMean(items []*types.Item) []float32 {
	var sum []float32
	var n int
	for _, item := range items {
		for _, ch := range item.GetEmbedding() {
			if sum == nil {
				sum = make([]float32, len(ch))
			}
			if len(ch) != len(sum) {
				continue
			}
			for i, v := range ch {
				sum[i] += v
			}
			n++
		}
	}
	if n == 0 {
		return nil
	}
	for i := range sum {
		sum[i] /= float32(n)
	}
	return sum
}

func centered(v, mean []float32) []float32 {
	if len(mean) != len(v) {
		return v
	}
	out := make([]float32, len(v))
	for i := range v {
		out[i] = v[i] - mean[i]
	}
	return out
}

// DiversifyFilter reorders the ranked list with Maximal Marginal Relevance so
// the top isn't dominated by near-duplicate items.
type DiversifyFilter struct{}

func NewDiversifyFilter() *DiversifyFilter { return &DiversifyFilter{} }

func (f *DiversifyFilter) Name() string       { return "diversify" }
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

// LimitFilter keeps the top-N items.
type LimitFilter struct{}

func NewLimitFilter() *LimitFilter { return &LimitFilter{} }

func (f *LimitFilter) Name() string       { return "limit" }
func (f *LimitFilter) DependsOn() []string { return []string{"diversify"} }

func (f *LimitFilter) Filter(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	if len(items) > publishLimit {
		return items[:publishLimit], nil
	}
	return items, nil
}
