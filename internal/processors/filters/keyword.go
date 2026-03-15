package filters

import (
	"context"

	"cartero/internal/components"
	"cartero/internal/processors/names"
	"cartero/internal/queue"
	"cartero/internal/types"
)

type KeywordFilterProcessor struct {
	name       string
	embedCache *queue.EmbedCache
	embedReady bool
}

func NewKeywordFilterProcessor(name string) *KeywordFilterProcessor {
	return &KeywordFilterProcessor{name: name}
}

func (k *KeywordFilterProcessor) Name() string {
	return k.name
}

func (k *KeywordFilterProcessor) Initialize(ctx context.Context, st types.StateAccessor) error {
	kwCfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings

	if len(kwCfg.Keywords) == 0 {
		return nil
	}

	registry := st.GetRegistry()
	pc := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	embedder := pc.Embedder()
	if embedder == nil {
		return nil
	}

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

	st.GetLogger().Info("keyword_filter: keyword embeddings initialized", "processor", k.name, "count", len(kwCfg.Keywords), "embed_ready", k.embedReady)
	return nil
}

func (k *KeywordFilterProcessor) DependsOn() []string {
	return []string{names.ExtractText, names.EmbedText}
}

func (k *KeywordFilterProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings

	if len(cfg.Keywords) == 0 && len(cfg.ExactKeyword) == 0 {
		return nil
	}

	if k.embedReady {
		return k.processEmbedding(ctx, st, item)
	}

	logger := st.GetLogger()
	logger.Info("keyword_filter: rejected item — embed infrastructure not ready", "processor", k.name, "item_id", item.ID, "title", item.GetTitle())
	return types.NewFilteredError(k.name, item.ID, "embed infrastructure not ready").
		WithDetail("title", item.GetTitle())
}
