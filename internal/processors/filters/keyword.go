package filters

import (
	"context"
	"slices"
	"strings"

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

func (k *KeywordFilterProcessor) Initialize(_ context.Context, st types.StateAccessor) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	k.matcher = keywords.New(cfg.Keywords)
	return nil
}

func (k *KeywordFilterProcessor) DependsOn() []string {
	return []string{names.ExtractText, names.EmbedCategory}
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

	if cfg.EmbedModel != "" && item.GetEmbedding() != nil {
		err := k.processEmbedding(ctx, st, item)
		if err != errEmbedUnavailable {
			return err
		}
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

	logger.Debug("KeywordFilterProcessor density calculated", "processor", k.name, "item_id", item.ID, "density", density*100, "keywords", matchedKeyword, "threshold", cfg.KeywordThreshold*100)

	if density >= cfg.KeywordThreshold {
		logger.Info("KeywordFilterProcessor matched via density", "processor", k.name, "item_id", item.ID, "density", density*100, "keywords", matchedKeyword)
		item.SetMatchedKeywords(matchedKeyword)
		return nil
	}

	logger.Info("KeywordFilterProcessor rejected item", "processor", k.name, "item_id", item.ID, "density", density*100, "threshold", cfg.KeywordThreshold*100)
	return types.NewFilteredError(k.name, item.ID, "keyword density below threshold").
		WithDetail("density_percentage", density*100).
		WithDetail("threshold_percentage", cfg.KeywordThreshold*100)
}
