package filters

import (
	"context"
	"fmt"
	"log/slog"

	"cartero/internal/platforms"
	"cartero/internal/queue"
	"cartero/internal/types"
	"cartero/internal/utils/keywords"
)

func buildKeywordEmbeddings(ctx context.Context, client platforms.Embedder, embedCache *queue.EmbedCache, kws []keywords.KeywordWithContext) error {
	return SeedKeywordEmbeddings(ctx, client, embedCache, kws, nil)
}

func SeedKeywordEmbeddings(ctx context.Context, client platforms.Embedder, embedCache *queue.EmbedCache, kws []keywords.KeywordWithContext, logger *slog.Logger) error {
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
		logger.Info("calling embedder for missing keyword embeddings", "count", len(misses))
	}

	batchSize := 10
	totalEmbedded := 0
	var lastDim int
	for start := 0; start < len(prefixed); start += batchSize {
		end := start + batchSize
		if end > len(prefixed) {
			end = len(prefixed)
		}
		batch := prefixed[start:end]

		resp, err := client.Embed(ctx, batch)
		if err != nil {
			return fmt.Errorf("embed batch [%d:%d]: %w", start, end, err)
		}

		for i, kw := range misses[start:end] {
			if i >= len(resp) {
				break
			}
			if err := embedCache.Set(ctx, kw.Keyword, resp[i]); err != nil {
				return fmt.Errorf("embed cache set %q: %w", kw.Keyword, err)
			}
		}
		totalEmbedded += len(resp)
		if len(resp) > 0 {
			lastDim = len(resp[0])
		}
	}

	if logger != nil {
		logger.Info("stored embeddings in redis", "count", totalEmbedded)
	}

	dims, err := embedCache.GetDims(ctx)
	if err != nil {
		return fmt.Errorf("embed cache get dims: %w", err)
	}
	if dims == 0 && totalEmbedded > 0 {
		dims = lastDim
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

func (k *KeywordFilterProcessor) processEmbedding(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	logger := st.GetLogger()

	if item.GetEmbedding() == nil {
		logger.Info("keyword_filter: rejected item — no embedding available", "processor", k.name, "item_id", item.ID, "title", item.GetTitle())
		return types.NewFilteredError(k.name, item.ID, "no embedding available for semantic matching").
			WithDetail("title", item.GetTitle())
	}

	var bestScore float64
	var bestKeyword string
	var bestResults []queue.KNNResult

	for i, chunkVec := range item.GetEmbedding() {
		results, err := k.embedCache.KNNSearch(ctx, 5, chunkVec)
		if err != nil {
			logger.Warn("keyword_filter: KNN search failed for chunk", "item_id", item.ID, "chunk_index", i, "error", err)
			continue
		}

		if len(results) > 0 {
			if results[0].Score > bestScore || len(bestResults) == 0 {
				bestScore = results[0].Score
				bestKeyword = results[0].Keyword
				bestResults = results
			}
		}
	}

	logger.Debug("keyword_filter: top embedding matches", "item_id", item.ID, "title", item.GetTitle(), "top5", bestResults)

	if len(bestResults) == 0 || bestScore < cfg.EmbedThreshold {
		logger.Info("keyword_filter: rejected item via embedding similarity", "processor", k.name, "item_id", item.ID, "title", item.GetTitle(), "best_score", bestScore, "threshold", cfg.EmbedThreshold)
		return types.NewFilteredError(k.name, item.ID, "no keyword matched via embedding similarity").
			WithDetail("best_score", bestScore).
			WithDetail("threshold", cfg.EmbedThreshold)
	}

	logger.Info("keyword_filter: matched via embedding similarity", "processor", k.name, "item_id", item.ID, "title", item.GetTitle(), "keyword", bestKeyword, "score", bestScore)
	item.SetMatchedKeywords(bestKeyword)
	return nil
}
