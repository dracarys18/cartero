package filters

import (
	"context"
	"fmt"
	"log/slog"
	"math"

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
	if k.prefReady {
		prefVec, exists, err := k.embedCache.Get(ctx, prefKey)
		if err == nil && exists {
			for _, chunkVec := range item.GetEmbedding() {
				score := cosineSimilarity(prefVec, chunkVec)
				if score > bestPrefScore {
					bestPrefScore = score
				}
			}
		}
		if bestPrefScore < k.prefThreshold {
			logger.Info("keyword_filter: rejected — preference mismatch", "processor", k.name, "item_id", item.ID, "title", item.GetTitle(), "score", bestPrefScore, "threshold", k.prefThreshold)
			return types.NewFilteredError(k.name, item.ID, "preference mismatch").
				WithDetail("score", bestPrefScore).
				WithDetail("threshold", k.prefThreshold)
		}
	}

	var bestKwScore float64
	var bestKeyword string
	for _, chunkVec := range item.GetEmbedding() {
		results, err := k.embedCache.KNNSearch(ctx, 3, chunkVec)
		if err != nil {
			continue
		}
		for _, r := range results {
			if r.Keyword != prefKey && r.Score > bestKwScore {
				bestKwScore = r.Score
				bestKeyword = r.Keyword
			}
		}
	}

	if bestKeyword != "" {
		logger.Info("keyword_filter: labelled", "processor", k.name, "item_id", item.ID, "keyword", bestKeyword, "score", bestKwScore)
		item.SetMatchedKeywords(bestKeyword)
	}

	return nil
}

func cosineSimilarity(a, b []float32) float64 {
	var sum0, sum1, sum2, sum3 float64
	var normA0, normA1, normA2, normA3 float64
	var normB0, normB1, normB2, normB3 float64

	n := len(a) / 4 * 4
	for i := 0; i < n; i += 4 {
		sum0 += float64(a[i]) * float64(b[i])
		sum1 += float64(a[i+1]) * float64(b[i+1])
		sum2 += float64(a[i+2]) * float64(b[i+2])
		sum3 += float64(a[i+3]) * float64(b[i+3])
		normA0 += float64(a[i]) * float64(a[i])
		normA1 += float64(a[i+1]) * float64(a[i+1])
		normA2 += float64(a[i+2]) * float64(a[i+2])
		normA3 += float64(a[i+3]) * float64(a[i+3])
		normB0 += float64(b[i]) * float64(b[i])
		normB1 += float64(b[i+1]) * float64(b[i+1])
		normB2 += float64(b[i+2]) * float64(b[i+2])
		normB3 += float64(b[i+3]) * float64(b[i+3])
	}

	for i := n; i < len(a); i++ {
		sum0 += float64(a[i]) * float64(b[i])
		normA0 += float64(a[i]) * float64(a[i])
		normB0 += float64(b[i]) * float64(b[i])
	}

	dot := sum0 + sum1 + sum2 + sum3
	normA := normA0 + normA1 + normA2 + normA3
	normB := normB0 + normB1 + normB2 + normB3

	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
