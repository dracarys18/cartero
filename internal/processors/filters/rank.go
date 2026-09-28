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

const interestKey = "_interest"

const (
	topicQuestion   = "interest"
	qualityQuestion = "quality"
	noTopic         = "none"
)

const (
	evalConcurrency = 8
	maxSummaryBytes = 1000
	maxContentBytes = 12000
	minContentBytes = 1500
)

type RankFilter struct {
	jev       *platforms.JevPlatform
	cfg       config.InterestConfig
	prompts   config.Prompts
	questions map[string]platforms.JevQuestion
	enabled   bool
}

func NewRankFilter(jev *platforms.JevPlatform, cfg config.InterestConfig, prompts config.Prompts) *RankFilter {
	criteria := buildCriteria(cfg.Keywords, prompts.Topic.None)
	questions := map[string]platforms.JevQuestion{
		topicQuestion:   platforms.JevChoice(prompts.Topic.Instructions, criteria),
		qualityQuestion: platforms.JevScore(prompts.Quality.Instructions, prompts.Quality.Levels),
	}
	for _, flag := range prompts.Flags {
		questions[flag.Name] = platforms.JevNoul(flag.Question)
	}

	return &RankFilter{
		jev:       jev,
		cfg:       cfg,
		prompts:   prompts,
		questions: questions,
		enabled:   jev != nil && len(criteria) > 1,
	}
}

type topicSpec struct {
	covers   []string
	notFor   string
	examples []string
}

func buildCriteria(kws []keywords.KeywordWithContext, none string) map[string]any {
	topics := make(map[string]*topicSpec)
	for _, kw := range kws {
		label := kw.Keyword
		if label == "" {
			label = kw.Context
		}
		if label == "" {
			continue
		}
		t, ok := topics[label]
		if !ok {
			t = &topicSpec{}
			topics[label] = t
		}
		if kw.Context != "" && kw.Context != label {
			t.covers = append(t.covers, kw.Context)
		}
		if kw.NotFor != "" {
			t.notFor = kw.NotFor
		}
		t.examples = append(t.examples, kw.Examples...)
	}

	criteria := make(map[string]any, len(topics)+1)
	for label, t := range topics {
		criteria[label] = t.description()
	}
	criteria[noTopic] = none
	return criteria
}

func (t *topicSpec) description() any {
	var covers any
	switch len(t.covers) {
	case 0:
	case 1:
		covers = t.covers[0]
	default:
		covers = t.covers
	}
	if t.notFor == "" && len(t.examples) == 0 {
		return covers
	}

	desc := map[string]any{}
	if covers != nil {
		desc["covers"] = covers
	}
	if t.notFor != "" {
		desc["not_for"] = t.notFor
	}
	if len(t.examples) > 0 {
		desc["examples"] = t.examples
	}
	return desc
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

	batch.Run(ctx, idx, evalConcurrency, func(ctx context.Context, i int) {
		resp, err := f.evaluate(ctx, items[i])
		verdicts[i] = verdict{resp: resp, err: err}
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

		topic, offTopic := bestTopic(v.resp.Answers[topicQuestion])
		if offTopic >= f.cfg.OffTopicThreshold {
			reject(ctx, state, item, "off_topic", offTopic, topic)
			continue
		}

		if name, p, flagged := f.flagged(v.resp.Answers); flagged {
			reject(ctx, state, item, name, p, topic)
			continue
		}

		quality := v.resp.Answers[qualityQuestion].Score
		if quality < f.minQuality(item) {
			reject(ctx, state, item, "low_quality", quality, topic)
			continue
		}

		item.SetScore(quality)
		item.AddMetadata(interestKey, topic)
		item.SetMatchedKeywords(topic)
		out = append(out, item)
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

func (f *RankFilter) evaluate(ctx context.Context, item *types.Item) (*platforms.JevResponse, error) {
	resp, err := f.jev.SystemOne(ctx, f.articleState(item), f.questions)
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
	offTopic, ok := answer.Probabilities[noTopic]
	if !ok && answer.Choice == noTopic {
		offTopic = 1
	}

	topic, best := answer.Choice, -1.0
	for label, p := range answer.Probabilities {
		if label != noTopic && p > best {
			topic, best = label, p
		}
	}
	return topic, offTopic
}

func (f *RankFilter) flagged(answers map[string]platforms.JevAnswer) (string, float64, bool) {
	for _, flag := range f.prompts.Flags {
		if p := answers[flag.Name].Noul; p >= flag.Threshold {
			return flag.Name, p, true
		}
	}
	return "", 0, false
}

func (f *RankFilter) articleState(item *types.Item) map[string]any {
	st := map[string]any{"title": item.GetTitle()}
	if host := item.GetLink().Host; host != "" {
		st["site"] = host
	}

	summary := item.GetDescription()
	article := item.GetArticle()
	if article != nil && article.Description != "" {
		summary = article.Description
	}
	if summary != "" {
		st["summary"] = strutils.Truncate(summary, maxSummaryBytes)
	}
	if article != nil && article.Text != "" {
		st["content"] = strutils.Truncate(article.Text, maxContentBytes)
	}
	if contentIncomplete(item) {
		st["content_status"] = f.prompts.Quality.Incomplete
	}
	return st
}

func contentIncomplete(item *types.Item) bool {
	article := item.GetArticle()
	return article == nil || len(article.Text) < minContentBytes
}

func (f *RankFilter) minQuality(item *types.Item) float64 {
	if contentIncomplete(item) {
		return f.cfg.MinQualityIncomplete
	}
	return f.cfg.MinQuality
}

func reject(ctx context.Context, state types.StateAccessor, item *types.Item, reason string, score float64, label string) {
	state.GetLogger().Info("rank: rejected", "reason", reason, "score", score, "interest", label, "item_id", item.ID, "title", item.GetTitle())
	if state.GetRejected() != nil {
		if err := state.GetRejected().Add(ctx, item.ID); err != nil {
			state.GetLogger().Warn("rank: failed to record rejection", "item_id", item.ID, "error", err)
		}
	}
}
