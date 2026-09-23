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
	interestKey            = "_interest"
	jevQuestion            = "interest"
	jevNoMatch             = "none"
	jevConcurrency         = 8
	jevMaxSummaryBytes     = 1000
	defaultOffTopic        = 0.4
	defaultRejectThreshold = 0.7
)

const (
	jevInstructions = "Which topic is this article primarily about? Judge by its main subject, not passing mentions."
	jevNoMatchDesc  = "None of the other topics is the main subject, or it is general news, business, politics or marketing with no technical focus."
)

type RankFilter struct {
	jev       *platforms.JevPlatform
	cfg       config.InterestConfig
	rules     []config.RejectRule
	questions map[string]platforms.JevQuestion
	enabled   bool
}

func NewRankFilter(jev *platforms.JevPlatform, cfg config.InterestConfig) *RankFilter {
	if cfg.OffTopicThreshold <= 0 {
		cfg.OffTopicThreshold = defaultOffTopic
	}
	criteria := buildCriteria(cfg.Keywords)
	questions := map[string]platforms.JevQuestion{jevQuestion: platforms.JevChoice(jevInstructions, criteria)}

	rules := make([]config.RejectRule, len(cfg.RejectIf))
	for i, rule := range cfg.RejectIf {
		if rule.Threshold <= 0 {
			rule.Threshold = defaultRejectThreshold
		}
		rules[i] = rule
		questions[rule.Name] = platforms.JevNoul(rule.Question)
	}

	return &RankFilter{
		jev:       jev,
		cfg:       cfg,
		rules:     rules,
		questions: questions,
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
	resp *platforms.JevResponse
	err  error
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
		resp, err := f.evaluate(ctx, items[i])
		verdicts[i] = verdict{resp: resp, err: err}
	})

	out := make([]*types.Item, 0, len(items))
	var firstErr error
	failed, inputTokens := 0, 0

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

		inputTokens += v.resp.Usage.InputTokens
		topic, offTopic := bestTopic(v.resp.Answers[jevQuestion])
		if offTopic >= f.cfg.OffTopicThreshold {
			reject(ctx, state, item, "off_topic", offTopic, topic)
			continue
		}

		if rule, p, flagged := f.flagged(v.resp.Answers); flagged {
			reject(ctx, state, item, rule, p, topic)
			continue
		}

		item.SetScore(1 - offTopic)
		item.AddMetadata(interestKey, topic)
		item.SetMatchedKeywords(topic)
		out = append(out, item)
	}

	logger.Info("rank: jev batch", "items", len(items), "failed", failed, "input_tokens", inputTokens)

	if failed == len(items) {
		return nil, fmt.Errorf("rank: jev failed for all %d items: %w", failed, firstErr)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].GetScore() > out[j].GetScore() })

	for _, item := range out {
		logger.Info("rank: scored", "score", item.GetScore(), "interest", item.GetMatchedKeywords(), "title", item.GetTitle())
	}
	return out, nil
}

func (f *RankFilter) evaluate(ctx context.Context, item *types.Item) (*platforms.JevResponse, error) {
	resp, err := f.jev.SystemOne(ctx, articleState(item), f.questions)
	if err != nil {
		return nil, err
	}
	for name := range f.questions {
		if _, ok := resp.Answers[name]; !ok {
			return nil, fmt.Errorf("jev: response missing %q answer", name)
		}
	}
	return resp, nil
}

func bestTopic(answer platforms.JevAnswer) (string, float64) {
	offTopic, ok := answer.Probabilities[jevNoMatch]
	if !ok && answer.Choice == jevNoMatch {
		offTopic = 1
	}

	topic, best := answer.Choice, -1.0
	for label, p := range answer.Probabilities {
		if label != jevNoMatch && p > best {
			topic, best = label, p
		}
	}
	return topic, offTopic
}

func (f *RankFilter) flagged(answers map[string]platforms.JevAnswer) (string, float64, bool) {
	for _, rule := range f.rules {
		if p := answers[rule.Name].Noul; p >= rule.Threshold {
			return rule.Name, p, true
		}
	}
	return "", 0, false
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
