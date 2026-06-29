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

type KeywordFilterProcessor struct {
	name          string
	embedCache    *queue.EmbedCache
	embedReady    bool
	prefVec       []float32
	prefThreshold float64
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

	if hasKeywords {
		k.embedCache = queue.NewEmbedCache(st.GetRedisClient())
		if err := buildKeywordEmbeddings(ctx, embedder, k.embedCache, kwCfg.Keywords); err != nil {
			return err
		}
		if err := ensureIndexFromCache(ctx, k.embedCache); err != nil {
			return err
		}
		dims, err := k.embedCache.GetDims(ctx)
		if err != nil {
			return err
		}
		k.embedReady = dims > 0
		logger.Info("keyword_filter: keyword embeddings initialized", "processor", k.name, "count", len(kwCfg.Keywords), "embed_ready", k.embedReady)
	}

	if hasPreference {
		resp, err := embedder.Embed(ctx, &api.EmbedRequest{Input: []string{kwCfg.Preference}})
		if err != nil {
			return err
		}
		if len(resp.Embeddings) > 0 {
			k.prefVec = l2Normalize(resp.Embeddings[0])
		}

		if kwCfg.EmbedThreshold > 0 {
			k.prefThreshold = kwCfg.EmbedThreshold
		} else {
			k.prefThreshold = 0.55
		}

		logger.Info("keyword_filter: preference vector ready", "processor", k.name, "threshold", k.prefThreshold)
	}

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

	if len(k.prefVec) > 0 {
		if err := k.checkPreference(ctx, st, item); err != nil {
			return err
		}
	}

	if k.embedReady {
		k.labelItem(ctx, st, item)
	}

	return nil
}

func (k *KeywordFilterProcessor) checkPreference(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	logger := st.GetLogger()

	embeddings := item.GetEmbedding()
	if len(embeddings) == 0 {
		logger.Debug("keyword_filter: no embedding for preference check — passing through", "item_id", item.ID)
		return nil
	}

	titleVec := l2Normalize(embeddings[0])
	score := dotProduct(k.prefVec, titleVec)

	if score < k.prefThreshold {
		logger.Info("keyword_filter: rejected — preference mismatch", "processor", k.name, "item_id", item.ID, "title", item.GetTitle(), "score", score, "threshold", k.prefThreshold)
		return types.NewFilteredError(k.name, item.ID, "preference mismatch").
			WithDetail("score", score).
			WithDetail("threshold", k.prefThreshold)
	}

	logger.Debug("keyword_filter: preference match", "item_id", item.ID, "score", score)
	return nil
}

func l2Normalize(vec []float32) []float32 {
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return vec
	}
	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = float32(float64(v) / norm)
	}
	return result
}

func dotProduct(a, b []float32) float64 {
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}
