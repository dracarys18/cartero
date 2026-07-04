package filters

import (
	"context"
	"sort"

	"cartero/internal/config"
	"cartero/internal/platforms"
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils/keywords"

	"github.com/viterin/vek/vek32"
)

const (
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
		end := min(start+batch, len(texts))
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
	cfg       config.InterestConfig
	interests []Interest
	ready     bool
	mean      []float32
	count     float64
}

func NewRankFilter(embedder platforms.Embedder, cfg config.InterestConfig) *RankFilter {
	return &RankFilter{embedder: embedder, cfg: cfg}
}

func (f *RankFilter) Name() string        { return filterRank }
func (f *RankFilter) DependsOn() []string { return []string{names.EmbedText} }

func (f *RankFilter) Process(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	if !f.ready {
		interests, err := BuildInterests(ctx, f.embedder, f.cfg.Keywords)
		if err != nil {
			return nil, err
		}
		f.interests = interests
		f.ready = true
		f.seedMean()
		state.GetLogger().Info("rank: interests ready", "count", len(interests))
	}
	if len(f.interests) == 0 {
		return items, nil
	}

	ivecs := make([][]float32, len(f.interests))
	for i, in := range f.interests {
		ivecs[i] = centered(in.Vector, f.mean)
	}

	logger := state.GetLogger()
	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		raw := item.GetEmbedding()
		if len(raw) == 0 {
			logger.Warn("rank: rejected", "reason", "no embedding", "item_id", item.ID, "title", item.GetTitle())
			continue
		}
		chunks := make([][]float32, len(raw))
		for i, ch := range raw {
			chunks[i] = centered(ch, f.mean)
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
		item.SetScore(best)
		item.AddMetadata(interestKey, f.interests[bestIdx].Lexical)
		if best < f.cfg.MinScore {
			logger.Info("rank: rejected", "score", best, "interest", f.interests[bestIdx].Lexical, "title", item.GetTitle())
			continue
		}
		item.SetMatchedKeywords(f.interests[bestIdx].Lexical)
		out = append(out, item)
	}

	for _, item := range items {
		for _, ch := range item.GetEmbedding() {
			f.foldMean(ch)
		}
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].GetScore() > out[j].GetScore() })

	for _, item := range out {
		logger.Info("rank: scored", "score", item.GetScore(), "interest", item.GetMatchedKeywords(), "title", item.GetTitle())
	}
	return out, nil
}

func (f *RankFilter) seedMean() {
	var sum []float32
	var n int
	for _, in := range f.interests {
		if sum == nil {
			sum = make([]float32, len(in.Vector))
		}
		if len(in.Vector) != len(sum) {
			continue
		}
		vek32.Add_Inplace(sum, in.Vector)
		n++
	}
	if n == 0 {
		return
	}
	vek32.DivNumber_Inplace(sum, float32(n))
	f.mean = sum
	f.count = float64(n)
}

func (f *RankFilter) foldMean(v []float32) {
	if len(f.mean) != len(v) {
		return
	}
	f.count++
	diff := vek32.Sub(v, f.mean)
	vek32.DivNumber_Inplace(diff, float32(f.count))
	vek32.Add_Inplace(f.mean, diff)
}

func centered(v, mean []float32) []float32 {
	if len(mean) != len(v) {
		return v
	}
	return vek32.Sub(v, mean)
}
