package filters

import (
	"context"
	"fmt"
	"sort"

	"cartero/internal/config"
	"cartero/internal/platforms"
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils/batch"
	"cartero/internal/utils/keywords"
	strutils "cartero/internal/utils/string"
)

const (
	interestKey          = "_interest"
	jevQuestion          = "interest"
	jevNoMatch           = "none"
	jevConcurrency       = 8
	jevMaxSummaryBytes   = 1000
	defaultMinConfidence = 0.5
)

const (
	jevInstructions = "Which topic is this article primarily about? Judge by its main subject, not passing mentions."
	jevNoMatchDesc  = "None of the other topics is the main subject, or it is general news, business, politics or marketing with no technical focus."
)

type RankFilter struct {
	jev       *platforms.JevPlatform
	cfg       config.InterestConfig
	questions map[string]platforms.JevQuestion
	enabled   bool
}

func NewRankFilter(jev *platforms.JevPlatform, cfg config.InterestConfig) *RankFilter {
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = defaultMinConfidence
	}
	criteria := buildCriteria(cfg.Keywords)
	return &RankFilter{
		jev:       jev,
		cfg:       cfg,
		questions: map[string]platforms.JevQuestion{jevQuestion: platforms.JevChoice(jevInstructions, criteria)},
		enabled:   jev != nil && len(criteria) > 1,
	}
}

func buildCriteria(kws []keywords.KeywordWithContext) map[string]any {
	facets := make(map[string][]string)
	for _, kw := range kws {
		label := kw.Keyword
		if label == "" {
			label = kw.Context
		}
		if label == "" {
			continue
		}
		if kw.Context != "" && kw.Context != label {
			facets[label] = append(facets[label], kw.Context)
		} else if _, ok := facets[label]; !ok {
			facets[label] = nil
		}
	}

	criteria := make(map[string]any, len(facets)+1)
	for label, descs := range facets {
		if len(descs) == 0 {
			criteria[label] = nil
			continue
		}
		criteria[label] = descs
	}
	criteria[jevNoMatch] = jevNoMatchDesc
	return criteria
}

func (f *RankFilter) Name() string        { return filterRank }
func (f *RankFilter) DependsOn() []string { return []string{names.ExtractText} }

type verdict struct {
	answer *platforms.JevAnswer
	err    error
}

func (f *RankFilter) Process(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	if !f.enabled || len(items) == 0 {
		return items, nil
	}

	logger := state.GetLogger()
	verdicts := make([]verdict, len(items))
	idx := make([]int, len(items))
	for i := range idx {
		idx[i] = i
	}

	batch.Run(ctx, idx, jevConcurrency, func(ctx context.Context, i int) {
		answer, err := f.evaluate(ctx, items[i])
		verdicts[i] = verdict{answer: answer, err: err}
	})

	out := make([]*types.Item, 0, len(items))
	var firstErr error
	failed := 0

	for i, item := range items {
		v := verdicts[i]
		if v.err != nil {
			failed++
			if firstErr == nil {
				firstErr = v.err
			}
			logger.Warn("rank: jev failed, will retry next run", "item_id", item.ID, "title", item.GetTitle(), "error", v.err)
			continue
		}

		choice, confidence := v.answer.Choice, v.answer.Confidence
		switch {
		case choice == jevNoMatch:
			reject(ctx, state, item, "no_match", confidence, choice)
		case confidence < f.cfg.MinConfidence:
			reject(ctx, state, item, "low_confidence", confidence, choice)
		default:
			item.SetScore(confidence)
			item.AddMetadata(interestKey, choice)
			item.SetMatchedKeywords(choice)
			out = append(out, item)
		}
	}

	if failed == len(items) {
		return nil, fmt.Errorf("rank: jev failed for all %d items: %w", failed, firstErr)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].GetScore() > out[j].GetScore() })

	for _, item := range out {
		logger.Info("rank: scored", "score", item.GetScore(), "interest", item.GetMatchedKeywords(), "title", item.GetTitle())
	}
	return out, nil
}

func (f *RankFilter) evaluate(ctx context.Context, item *types.Item) (*platforms.JevAnswer, error) {
	resp, err := f.jev.SystemOne(ctx, articleState(item), f.questions)
	if err != nil {
		return nil, err
	}
	answer, ok := resp.Answers[jevQuestion]
	if !ok {
		return nil, fmt.Errorf("jev: response missing %q answer", jevQuestion)
	}
	return &answer, nil
}

func articleState(item *types.Item) map[string]any {
	st := map[string]any{"title": item.GetTitle()}
	if host := item.GetLink().Host; host != "" {
		st["site"] = host
	}

	summary := item.GetDescription()
	if article := item.GetArticle(); article != nil {
		if article.Description != "" {
			summary = article.Description
		} else if summary == "" {
			summary = article.Text
		}
	}
	if summary != "" {
		st["summary"] = strutils.Truncate(summary, jevMaxSummaryBytes)
	}
	return st
}

func reject(ctx context.Context, state types.StateAccessor, item *types.Item, reason string, score float64, label string) {
	state.GetLogger().Info("rank: rejected", "reason", reason, "score", score, "interest", label, "item_id", item.ID, "title", item.GetTitle())
	if state.GetRejected() != nil {
		if err := state.GetRejected().Add(ctx, item.ID); err != nil {
			state.GetLogger().Warn("rank: failed to record rejection", "item_id", item.ID, "error", err)
		}
	}
}
