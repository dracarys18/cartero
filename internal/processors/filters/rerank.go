package filters

import (
	"context"
	"sort"

	"cartero/internal/platforms"
	"cartero/internal/types"
)

const (
	shortlistK = 50
	docMaxLen  = 500
)

type RerankFilter struct {
	reranker platforms.Reranker
}

func NewRerankFilter(reranker platforms.Reranker) *RerankFilter {
	return &RerankFilter{reranker: reranker}
}

func (f *RerankFilter) Name() string        { return filterRerank }
func (f *RerankFilter) DependsOn() []string { return []string{filterRank} }

func (f *RerankFilter) Filter(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	if f.reranker == nil {
		return items, nil
	}
	logger := state.GetLogger()

	if len(items) > shortlistK {
		items = items[:shortlistK]
	}

	groups := map[string][]*types.Item{}
	for _, item := range items {
		interest, _ := item.Metadata[interestKey].(string)
		groups[interest] = append(groups[interest], item)
	}

	scored := make([]*types.Item, 0, len(items))
	for interest, group := range groups {
		if interest == "" {
			scored = append(scored, group...)
			continue
		}

		docs := make([]string, len(group))
		for i, item := range group {
			docs[i] = docText(item)
		}

		scores, err := f.reranker.Rerank(ctx, interest, docs)
		if err != nil {
			logger.Error("rerank: call failed, keeping group", "interest", interest, "error", err)
			scored = append(scored, group...)
			continue
		}

		for i, item := range group {
			if i < len(scores) {
				setScore(item, scores[i])
			}
			scored = append(scored, item)
		}
	}

	sort.SliceStable(scored, func(i, j int) bool { return getScore(scored[i]) > getScore(scored[j]) })

	for i, item := range scored {
		if i >= 12 {
			break
		}
		logger.Info("rerank: ranked", "pos", i+1, "score", getScore(item), "title", item.GetTitle())
	}
	return scored, nil
}

func docText(item *types.Item) string {
	text := item.GetTitle()
	if a := item.GetArticle(); a != nil && a.Text != "" {
		body := a.Text
		if len(body) > docMaxLen {
			body = body[:docMaxLen]
		}
		text += ". " + body
	} else if d := item.GetDescription(); d != "" {
		text += ". " + d
	}
	return text
}
