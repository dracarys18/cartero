package filters

import (
	"context"

	"cartero/internal/platforms"
	"cartero/internal/types"
	"cartero/internal/utils"
	"cartero/internal/utils/keywords"

	"github.com/ollama/ollama/api"
)

const queryPrefix = "Retrieve technical articles about: "

func buildKeywordEmbeddings(ctx context.Context, client *platforms.OllamaPlatform, kws []string) (map[string][]float32, error) {
	prefixed := make([]string, len(kws))
	for i, kw := range kws {
		prefixed[i] = queryPrefix + kw
	}

	resp, err := client.Embed(ctx, &api.EmbedRequest{Input: prefixed})
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

	itemEmbedding := item.GetEmbedding()
	topK := keywords.NewTopK(5)

	for kw, kwEmb := range k.keywordCache.All() {
		topK.Add(kw, utils.CosineSimilarity(itemEmbedding, kwEmb))
	}

	logger.Debug("keyword_filter: top embedding matches", "item_id", item.ID, "top5", topK.Top(5))

	best, ok := topK.Best()
	if !ok || best.Score < cfg.EmbedThreshold {
		logger.Info("keyword_filter: rejected item via embedding similarity", "processor", k.name, "item_id", item.ID, "best_score", best.Score, "threshold", cfg.EmbedThreshold)
		return types.NewFilteredError(k.name, item.ID, "no keyword matched via embedding similarity").
			WithDetail("best_score", best.Score).
			WithDetail("threshold", cfg.EmbedThreshold)
	}

	logger.Info("keyword_filter: matched via embedding similarity", "processor", k.name, "item_id", item.ID, "keyword", best.Keyword, "score", best.Score)
	item.SetMatchedKeywords(best.Keyword)
	return nil
}
