package filters

import (
	"cmp"
	"context"
	"slices"

	"cartero/internal/platforms"
	"cartero/internal/types"
	"cartero/internal/utils"

	"github.com/ollama/ollama/api"
)

func buildKeywordEmbeddings(ctx context.Context, client *platforms.OllamaPlatform, kws []string) (map[string][]float32, error) {
	resp, err := client.Embed(ctx, &api.EmbedRequest{Input: kws})
	if err != nil {
		return nil, err
	}

	cache := make(map[string][]float32, len(kws))
	for i, kw := range kws {
		if i < len(resp.Embeddings) {
			cache[kw] = resp.Embeddings[i]
		}
	}
	return cache, nil
}

func (k *KeywordFilterProcessor) processEmbedding(st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	logger := st.GetLogger()

	type kwScore struct {
		kw    string
		score float64
	}

	itemEmbedding := item.GetEmbedding()
	var bestKeyword string
	var bestScore float64
	var all []kwScore

	for kw, kwEmb := range k.keywordCache.All() {
		score := utils.CosineSimilarity(itemEmbedding, kwEmb)
		all = append(all, kwScore{kw, score})
		if score > bestScore {
			bestScore = score
			bestKeyword = kw
		}
	}

	slices.SortFunc(all, func(a, b kwScore) int { return cmp.Compare(b.score, a.score) })
	if len(all) > 5 {
		all = all[:5]
	}
	logger.Debug("keyword_filter: top embedding matches", "item_id", item.ID, "top5", all)

	if bestScore >= cfg.Threshold {
		logger.Info("keyword_filter: matched via embedding similarity", "processor", k.name, "item_id", item.ID, "keyword", bestKeyword, "score", bestScore)
		item.SetMatchedKeywords(bestKeyword)
		return nil
	}

	logger.Info("keyword_filter: rejected item via embedding similarity", "processor", k.name, "item_id", item.ID, "best_score", bestScore, "threshold", cfg.Threshold)
	return types.NewFilteredError(k.name, item.ID, "no keyword matched via embedding similarity").
		WithDetail("best_score", bestScore).
		WithDetail("threshold", cfg.Threshold)
}
