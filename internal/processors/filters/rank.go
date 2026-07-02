package filters

import (
	"context"
	"sort"
	"time"

	"cartero/internal/storage"
)

type Interest struct {
	Vector  []float32
	Lexical string
}

type Options struct {
	WSemantic     float64
	WLexical      float64
	SemanticFloor float64
	MinScore      float64
	MMRLambda     float64
	Limit         int
	Lookback      time.Duration
}

func DefaultOptions() Options {
	return Options{
		WSemantic:     0.7,
		WLexical:      0.3,
		SemanticFloor: 0.45,
		MMRLambda:     0.7,
		Limit:         50,
	}
}

type Result struct {
	Entry    storage.FeedEntry
	Score    float64
	Semantic float64
	Lexical  float64
}

type Ranker struct {
	store storage.EntryStore
}

func NewRanker(store storage.EntryStore) *Ranker {
	return &Ranker{store: store}
}

func (r *Ranker) Rank(ctx context.Context, interests []Interest, opts Options) ([]Result, error) {
	if len(interests) == 0 {
		return nil, nil
	}

	ris := make([]storage.RankInterest, 0, len(interests))
	for _, in := range interests {
		ris = append(ris, storage.RankInterest{Text: in.Lexical, Vector: in.Vector})
	}

	var since time.Time
	if opts.Lookback > 0 {
		since = time.Now().Add(-opts.Lookback)
	}

	pool := opts.Limit * 5
	if pool < 100 {
		pool = 100
	}

	cands, err := r.store.RankCandidates(ctx, ris, since, pool)
	if err != nil {
		return nil, err
	}

	scored := make([]scoredCandidate, 0, len(cands))
	for _, c := range cands {
		normSem := normalizeSemantic(c.Semantic, opts.SemanticFloor)
		score := opts.WSemantic*normSem + opts.WLexical*c.Lexical
		if score < opts.MinScore {
			continue
		}
		scored = append(scored, scoredCandidate{
			result: Result{Entry: c.Entry, Score: score, Semantic: normSem, Lexical: c.Lexical},
			vector: c.Embedding,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool { return scored[i].result.Score > scored[j].result.Score })

	limit := opts.Limit
	if limit <= 0 {
		limit = len(scored)
	}
	return mmrRerank(scored, opts.MMRLambda, limit), nil
}

func normalizeSemantic(sem, floor float64) float64 {
	if floor <= 0 || floor >= 1 {
		return sem
	}
	if sem <= floor {
		return 0
	}
	return (sem - floor) / (1 - floor)
}
