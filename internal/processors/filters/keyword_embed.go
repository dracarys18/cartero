package filters

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"cartero/internal/components"
	"cartero/internal/types"
	"cartero/internal/utils"

	"github.com/ollama/ollama/api"
)

var errEmbedUnavailable = errors.New("embed unavailable")

func (k *KeywordFilterProcessor) processEmbedding(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	logger := st.GetLogger()

	registry := st.GetRegistry()
	pc := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	ollamaClient := pc.OllamaPlatform(cfg.EmbedModel)

	err := k.keywordCache.Load(func() (map[string][]float32, error) {
		seen := make(map[string]struct{})
		var allKeywords []string
		for kw := range slices.Values(cfg.Keywords) {
			if _, ok := seen[kw]; !ok {
				seen[kw] = struct{}{}
				allKeywords = append(allKeywords, kw)
			}
		}
		for kw := range slices.Values(cfg.ExactKeyword) {
			if _, ok := seen[kw]; !ok {
				seen[kw] = struct{}{}
				allKeywords = append(allKeywords, kw)
			}
		}

		resp, err := ollamaClient.Embed(ctx, &api.EmbedRequest{Input: allKeywords})
		if err != nil {
			return nil, err
		}

		cache := make(map[string][]float32, len(allKeywords))
		for i, kw := range allKeywords {
			if i < len(resp.Embeddings) {
				cache[kw] = resp.Embeddings[i]
			}
		}
		logger.Info("keyword_filter: keyword cache initialized", "processor", k.name, "count", len(cache))
		return cache, nil
	})

	if err != nil {
		logger.Warn("keyword_filter: keyword cache init failed", "processor", k.name, "error", err)
		return errEmbedUnavailable
	}

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
