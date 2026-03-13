package filters

import (
	"context"
	"slices"
	"strings"

	"cartero/internal/components"
	"cartero/internal/config"
	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils"
	"cartero/internal/utils/keywords"
)

type KeywordFilterProcessor struct {
	name         string
	matcher      *keywords.Keywords
	keywordCache utils.SyncCache[string, []float32]
}

func NewKeywordFilterProcessor(name string) *KeywordFilterProcessor {
	return &KeywordFilterProcessor{name: name}
}

func (k *KeywordFilterProcessor) Name() string {
	return k.name
}

func (k *KeywordFilterProcessor) Initialize(ctx context.Context, st types.StateAccessor) error {
	kwCfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	k.matcher = keywords.New(kwCfg.Keywords)

	if len(kwCfg.Keywords) == 0 {
		return nil
	}

	registry := st.GetRegistry()
	pc := registry.Get(components.PlatformComponentName).(*components.PlatformComponent)
	embedder := pc.Embedder()
	if embedder == nil {
		return nil
	}

	err := k.keywordCache.Load(func() (map[string][]float32, error) {
		return buildKeywordEmbeddings(ctx, embedder, kwCfg.Keywords)
	})
	if err == nil {
		st.GetLogger().Info("keyword_filter: keyword embeddings initialized", "processor", k.name, "count", k.keywordCache.Len())
	}
	return err
}

func (k *KeywordFilterProcessor) DependsOn() []string {
	return []string{names.ExtractText, names.EmbedText}
}

func (k *KeywordFilterProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings

	if len(cfg.Keywords) == 0 && len(cfg.ExactKeyword) == 0 {
		return nil
	}

	if matched := k.matchTitle(cfg, item); matched != "" {
		st.GetLogger().Info("KeywordFilterProcessor matched via title", "processor", k.name, "item_id", item.ID, "keyword", matched)
		item.SetMatchedKeywords(matched)
		return nil
	}

	if k.keywordCache.Len() > 0 && item.GetEmbedding() != nil {
		return k.processEmbedding(st, item)
	}

	return k.processDensity(st, item)
}

func (k *KeywordFilterProcessor) matchTitle(cfg config.KeywordFilterSettings, item *types.Item) string {
	title := item.GetTitle()
	titleLower := strings.ToLower(title)

	for exactKeyword := range slices.Values(cfg.ExactKeyword) {
		if strings.Contains(titleLower, strings.ToLower(exactKeyword)) {
			return exactKeyword
		}
	}

	if !cfg.TitleBypass {
		return ""
	}
	return k.matcher.MatchTitle(title)
}

func (k *KeywordFilterProcessor) processDensity(st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	logger := st.GetLogger()

	title := item.GetTitle()

	var content string
	if article := item.GetArticle(); article != nil {
		content = article.Text
	}

	contentLower := strings.ToLower(content)

	for exactKeyword := range slices.Values(cfg.ExactKeyword) {
		if strings.Contains(contentLower, strings.ToLower(exactKeyword)) {
			logger.Debug("KeywordFilterProcessor matched exact keyword in content", "processor", k.name, "item_id", item.ID, "keyword", exactKeyword)
			item.SetMatchedKeywords(exactKeyword)
			return nil
		}
	}

	matchedKeyword, density := k.matcher.Match(title, content)

	logger.Debug("KeywordFilterProcessor density calculated", "processor", k.name, "item_id", item.ID, "density", density*100, "keywords", matchedKeyword, "threshold", cfg.DensityThreshold*100)

	if density >= cfg.DensityThreshold {
		logger.Info("KeywordFilterProcessor matched via density", "processor", k.name, "item_id", item.ID, "density", density*100, "keywords", matchedKeyword)
		item.SetMatchedKeywords(matchedKeyword)
		return nil
	}

	logger.Info("KeywordFilterProcessor rejected item", "processor", k.name, "item_id", item.ID, "density", density*100, "threshold", cfg.DensityThreshold*100)
	return types.NewFilteredError(k.name, item.ID, "keyword density below threshold").
		WithDetail("density_percentage", density*100).
		WithDetail("threshold_percentage", cfg.DensityThreshold*100)
}
