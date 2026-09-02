package filters

import (
	"context"
	"sort"

	"cartero/internal/config"
	"cartero/internal/platforms"
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils/keywords"
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
}

func NewRankFilter(embedder platforms.Embedder, cfg config.InterestConfig) *RankFilter {
	if cfg.MinScore <= 0 {
		cfg.MinScore = 0.5
	}
	if cfg.Margin < 0 {
		cfg.Margin = 0
	}
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
		state.GetLogger().Info("rank: interests ready", "count", len(interests))
	}
	if len(f.interests) == 0 {
		return items, nil
	}

	logger := state.GetLogger()
	labelScores := make(map[string]float64, len(f.interests))
	out := make([]*types.Item, 0, len(items))

	for _, item := range items {
		doc := docVector(item)
		if len(doc) == 0 {
			logger.Warn("rank: rejected", "reason", "no embedding", "item_id", item.ID, "title", item.GetTitle())
			continue
		}

		// Score each category as the max of its facets so repeated
		// facets of the same label don't inflate the margin check.
		for k := range labelScores {
			delete(labelScores, k)
		}
		for _, in := range f.interests {
			if s := cosine(in.Vector, doc); s > labelScores[in.Lexical] {
				labelScores[in.Lexical] = s
			}
		}

		best, bestLabel, second := -1.0, "", -1.0
		for label, s := range labelScores {
			if s > best {
				second = best
				best, bestLabel = s, label
			} else if s > second {
				second = s
			}
		}

		item.SetScore(best)
		item.AddMetadata(interestKey, bestLabel)

		reason := ""
		switch {
		case best < f.cfg.MinScore:
			reason = "below_threshold"
		case second > -1 && best-second < f.cfg.Margin:
			reason = "ambiguous"
		}

		if reason != "" {
			f.reject(ctx, state, item, reason, best, bestLabel)
			continue
		}

		item.SetMatchedKeywords(bestLabel)
		out = append(out, item)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].GetScore() > out[j].GetScore() })

	for _, item := range out {
		logger.Info("rank: scored", "score", item.GetScore(), "interest", item.GetMatchedKeywords(), "title", item.GetTitle())
	}
	return out, nil
}

func (f *RankFilter) reject(ctx context.Context, state types.StateAccessor, item *types.Item, reason string, score float64, label string) {
	state.GetLogger().Info("rank: rejected", "reason", reason, "score", score, "interest", label, "item_id", item.ID, "title", item.GetTitle())
	if state.GetRejected() != nil {
		if err := state.GetRejected().Add(ctx, item.ID); err != nil {
			state.GetLogger().Warn("rank: failed to record rejection", "item_id", item.ID, "error", err)
		}
	}
}

// docVector returns the item's title embedding (the first chunk produced by
// embed_text). Classifying on the title instead of max-pooling over every
// body chunk stops a single noisy paragraph from steering the tag.
func docVector(item *types.Item) []float32 {
	if e := item.GetEmbedding(); len(e) > 0 {
		return e[0]
	}
	return nil
}
