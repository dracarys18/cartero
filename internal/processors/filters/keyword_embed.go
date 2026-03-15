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

const queryPrefix = "Retrieve technical articles about: "

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
		prefixed[i] = queryPrefix + kw.Context
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

func (k *KeywordFilterProcessor) processEmbedding(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	logger := st.GetLogger()

	if item.GetEmbedding() == nil {
		logger.Info("keyword_filter: rejected item — no embedding available", "processor", k.name, "item_id", item.ID, "title", item.GetTitle())
		return types.NewFilteredError(k.name, item.ID, "no embedding available for semantic matching").
			WithDetail("title", item.GetTitle())
	}

	results, err := k.embedCache.KNNSearch(ctx, 5, item.GetEmbedding())
	if err != nil {
		return fmt.Errorf("KNN search: %w", err)
	}

	logger.Debug("keyword_filter: top embedding matches", "item_id", item.ID, "title", item.GetTitle(), "top5", results)

	if len(results) == 0 || results[0].Score < cfg.EmbedThreshold {
		best := 0.0
		if len(results) > 0 {
			best = results[0].Score
		}
		logger.Info("keyword_filter: rejected item via embedding similarity", "processor", k.name, "item_id", item.ID, "title", item.GetTitle(), "best_score", best, "threshold", cfg.EmbedThreshold)
		return types.NewFilteredError(k.name, item.ID, "no keyword matched via embedding similarity").
			WithDetail("best_score", best).
			WithDetail("threshold", cfg.EmbedThreshold)
	}

	logger.Info("keyword_filter: matched via embedding similarity", "processor", k.name, "item_id", item.ID, "title", item.GetTitle(), "keyword", results[0].Keyword, "score", results[0].Score)
	item.SetMatchedKeywords(results[0].Keyword)
	return nil
}
