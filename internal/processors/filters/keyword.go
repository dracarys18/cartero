package filters

import (
	"context"

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

		if kwCfg.EmbedThreshold > 0 {
			k.prefThreshold = kwCfg.EmbedThreshold
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

	if err := k.processWithRedis(ctx, st, item); err != nil {
		return err
	}

	return nil
}
