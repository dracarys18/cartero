package filters

import (
	"context"
	"fmt"
	"log/slog"

	"cartero/internal/platforms"
	"cartero/internal/queue"
	"cartero/internal/types"
	"cartero/internal/utils/keywords"

	"github.com/ollama/ollama/api"
)

func buildKeywordEmbeddings(ctx context.Context, client *platforms.OllamaPlatform, embedCache *queue.EmbedCache, kws []keywords.KeywordWithContext) error {
	return SeedKeywordEmbeddings(ctx, client, embedCache, kws, nil)
}

func SeedKeywordEmbeddings(ctx context.Context, client *platforms.OllamaPlatform, embedCache *queue.EmbedCache, kws []keywords.KeywordWithContext, logger *slog.Logger) error {
	var misses []keywords.KeywordWithContext
	for _, kw := range kws {
		_, ok, err := embedCache.Get(ctx, kw.Keyword)
		if err != nil {
			return fmt.Errorf("embed cache get %q: %w", kw.Keyword, err)
		}
		if !ok {
			misses = append(misses, kw)
		}
	}

	if logger != nil {
		logger.Info("embedding lookup complete", "total", len(kws), "cached", len(kws)-len(misses), "missing", len(misses))
	}

	if len(misses) == 0 {
		dims, err := embedCache.GetDims(ctx)
		if err != nil {
			return fmt.Errorf("embed cache get dims: %w", err)
		}
		if dims == 0 {
			return nil
		}
		return embedCache.EnsureIndex(ctx, dims)
	}

	prefixed := make([]string, len(misses))
	for i, kw := range misses {
		prefixed[i] = kw.Context
	}

	if logger != nil {
		logger.Info("calling ollama for missing embeddings", "count", len(misses))
	}

	resp, err := client.Embed(ctx, &api.EmbedRequest{Input: prefixed})
	if err != nil {
		return fmt.Errorf("ollama embed: %w", err)
	}

	for i, kw := range misses {
		if i >= len(resp.Embeddings) {
			break
		}
		if err := embedCache.Set(ctx, kw.Keyword, resp.Embeddings[i]); err != nil {
			return fmt.Errorf("embed cache set %q: %w", kw.Keyword, err)
		}
	}

	if logger != nil {
		logger.Info("stored embeddings in redis", "count", len(resp.Embeddings))
	}

	dims, err := embedCache.GetDims(ctx)
	if err != nil {
		return fmt.Errorf("embed cache get dims: %w", err)
	}
	if dims == 0 && len(resp.Embeddings) > 0 {
		dims = len(resp.Embeddings[0])
		if err := embedCache.SetDims(ctx, dims); err != nil {
			return fmt.Errorf("embed cache set dims: %w", err)
		}
	}

	if err := embedCache.EnsureIndex(ctx, dims); err != nil {
		return err
	}

	if logger != nil {
		logger.Info("HNSW index ready", "dims", dims)
	}

	return nil
}

func ensureIndexFromCache(ctx context.Context, embedCache *queue.EmbedCache) error {
	dims, err := embedCache.GetDims(ctx)
	if err != nil {
		return fmt.Errorf("embed cache get dims: %w", err)
	}
	if dims == 0 {
		return nil
	}
	return embedCache.EnsureIndex(ctx, dims)
}

func (k *KeywordFilterProcessor) processWithRedis(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	logger := st.GetLogger()

	if item.GetEmbedding() == nil {
		logger.Debug("keyword_filter: no embedding available", "item_id", item.ID)
		return types.NewFilteredError(k.name, item.ID, "no embedding available").
			WithDetail("title", item.GetTitle())
	}

	var bestPrefScore float64
	var bestKwScore float64
	var bestKeyword string

	for _, chunkVec := range item.GetEmbedding() {
		results, err := k.embedCache.KNNSearch(ctx, 20, chunkVec)
		if err != nil {
			logger.Warn("keyword_filter: KNN search failed", "item_id", item.ID, "error", err)
			continue
		}

		for _, r := range results {
			if r.Keyword == prefKey && r.Score > bestPrefScore {
				bestPrefScore = r.Score
			}
			if r.Keyword != prefKey && r.Score > bestKwScore {
				bestKwScore = r.Score
				bestKeyword = r.Keyword
			}
		}
	}

	if k.prefReady && bestPrefScore < k.prefThreshold {
		logger.Info("keyword_filter: rejected — preference mismatch", "processor", k.name, "item_id", item.ID, "title", item.GetTitle(), "score", bestPrefScore, "threshold", k.prefThreshold)
		return types.NewFilteredError(k.name, item.ID, "preference mismatch").
			WithDetail("score", bestPrefScore).
			WithDetail("threshold", k.prefThreshold)
	}

	if bestKeyword != "" {
		logger.Info("keyword_filter: labelled", "processor", k.name, "item_id", item.ID, "keyword", bestKeyword, "score", bestKwScore)
		item.SetMatchedKeywords(bestKeyword)
	}

	return nil
}
