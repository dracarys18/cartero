package filters

import (
	"context"
	"sort"

	"cartero/internal/platforms"
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils/keywords"

	"github.com/viterin/vek/vek32"
)

const (
	scoreKey    = "_score"
	interestKey = "_interest"
)

type Interest struct {
	Vector  []float32
	Lexical string
}

func BuildInterests(ctx context.Context, embedder platforms.Embedder, kws []keywords.KeywordWithContext) ([]Interest, error) {
	if embedder == nil || len(kws) == 0 {
		return nil, nil
	}

	texts := make([]string, len(kws))
	labels := make([]string, len(kws))
	for i, kw := range kws {
		text := kw.Context
		if text == "" {
			text = kw.Keyword
		}
		texts[i] = text
		labels[i] = kw.Keyword
		if labels[i] == "" {
			labels[i] = kw.Context
		}
	}

	const batch = 32
	var vectors [][]float32
	for start := 0; start < len(texts); start += batch {
		end := start + batch
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := embedder.Embed(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vecs...)
	}

	out := make([]Interest, 0, len(vectors))
	for i := range vectors {
		out = append(out, Interest{Vector: vectors[i], Lexical: labels[i]})
	}
	return out, nil
}

type RankFilter struct {
	embedder  platforms.Embedder
	source    []keywords.KeywordWithContext
	interests []Interest
	ready     bool
}

func NewRankFilter(embedder platforms.Embedder, source []keywords.KeywordWithContext) *RankFilter {
	return &RankFilter{embedder: embedder, source: source}
}

func (f *RankFilter) Name() string        { return filterRank }
func (f *RankFilter) DependsOn() []string { return []string{names.EmbedText} }

func (f *RankFilter) Process(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
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
		item.SetMatchedKeywords(f.interests[bestIdx].Lexical)
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
			vek32.Add_Inplace(sum, ch)
			n++
		}
	}
	if n == 0 {
		return nil
	}
	vek32.DivNumber_Inplace(sum, float32(n))
	return sum
}

func centered(v, mean []float32) []float32 {
	if len(mean) != len(v) {
		return v
	}
	return vek32.Sub(v, mean)
}

func setScore(item *types.Item, s float64) { item.AddMetadata(scoreKey, s) }

func getScore(item *types.Item) float64 {
	if v, ok := item.Metadata[scoreKey].(float64); ok {
		return v
	}
	return 0
}
