package filters

import (
	"context"
	"math"

	"cartero/internal/components"
	"cartero/internal/processors/names"
	"cartero/internal/queue"
	"cartero/internal/types"

	"github.com/ollama/ollama/api"
)

const prefKey = "__pref__"

type KeywordFilterProcessor struct {
	name          string
	embedCache    *queue.EmbedCache
	embedReady    bool
	prefThreshold float64
	prefReady     bool
}

func NewKeywordFilterProcessor(name string) *KeywordFilterProcessor {
	return &KeywordFilterProcessor{name: name}
}

func (k *KeywordFilterProcessor) Name() string {
	return k.name
}

func (k *KeywordFilterProcessor) Initialize(ctx context.Context, st types.StateAccessor) error {
	kwCfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	logger := st.GetLogger()

	hasKeywords := len(kwCfg.Keywords) > 0
	hasPreference := kwCfg.Preference != ""

	if !hasKeywords && !hasPreference {
		return nil
	}

	registry := st.GetRegistry()
	pc := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	embedder := pc.Embedder()
	if embedder == nil {
		return nil
	}

	k.embedCache = queue.NewEmbedCache(st.GetRedisClient())

	if hasKeywords {
		if err := buildKeywordEmbeddings(ctx, embedder, k.embedCache, kwCfg.Keywords); err != nil {
			return err
		}
	}

	var prefResp *api.EmbedResponse
	if hasPreference {
		resp, err := embedder.Embed(ctx, &api.EmbedRequest{Input: []string{kwCfg.Preference}})
		if err != nil {
			return err
		}
		if len(resp.Embeddings) > 0 {
			if err := k.embedCache.Set(ctx, prefKey, resp.Embeddings[0]); err != nil {
				return err
			}
			k.prefReady = true
			prefResp = resp
		}

		dedupeCfg := st.GetConfig().Processors[k.name].Settings.DedupeSettings
		if dedupeCfg.EmbedThreshold > 0 {
			k.prefThreshold = dedupeCfg.EmbedThreshold
		} else {
			k.prefThreshold = 0.55
		}

		logger.Info("keyword_filter: preference stored in Redis", "processor", k.name, "threshold", k.prefThreshold)
	}

	dims, err := k.embedCache.GetDims(ctx)
	if err != nil {
		return err
	}
	if dims == 0 && hasPreference && prefResp != nil && len(prefResp.Embeddings) > 0 {
		dims = len(prefResp.Embeddings[0])
		if err := k.embedCache.SetDims(ctx, dims); err != nil {
			return err
		}
	}

	if err := ensureIndexFromCache(ctx, k.embedCache); err != nil {
		return err
	}

	k.embedReady = dims > 0
	logger.Info("keyword_filter: initialized", "processor", k.name, "keywords", len(kwCfg.Keywords), "pref_ready", k.prefReady, "embed_ready", k.embedReady)
	return nil
}

func (k *KeywordFilterProcessor) DependsOn() []string {
	return []string{names.ExtractText, names.EmbedText}
}

func (k *KeywordFilterProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings

	if len(cfg.Keywords) == 0 && cfg.Preference == "" {
		return nil
	}

	if err := k.process(ctx, st, item); err != nil {
		return err
	}

	return nil
}

func (k *KeywordFilterProcessor) process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
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
