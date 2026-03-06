package filters

import (
	"context"
	"slices"
	"strings"

	"cartero/internal/processors/names"
	"cartero/internal/types"
	"cartero/internal/utils/keywords"
)

type KeywordFilterProcessor struct {
	name    string
	matcher *keywords.Keywords
}

func NewKeywordFilterProcessor(name string) *KeywordFilterProcessor {
	return &KeywordFilterProcessor{name: name}
}

func (k *KeywordFilterProcessor) Name() string {
	return k.name
}

func (k *KeywordFilterProcessor) Initialize(ctx context.Context, st types.StateAccessor) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	k.matcher = keywords.New(cfg.Keywords)
	return nil
}

func (k *KeywordFilterProcessor) DependsOn() []string {
	return []string{names.ExtractText}
}

func (k *KeywordFilterProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[k.name].Settings.KeywordFilterSettings
	logger := st.GetLogger()

	if len(cfg.Keywords) == 0 && len(cfg.ExactKeyword) == 0 {
		return nil
	}

	title := item.GetTitle()

	var content string
	if article := item.GetArticle(); article != nil {
		content = article.Text
	}

	titleLower := strings.ToLower(title)
	contentLower := strings.ToLower(content)

	for exactKeyword := range slices.Values(cfg.ExactKeyword) {
		kw := strings.ToLower(exactKeyword)
		if strings.Contains(titleLower, kw) || strings.Contains(contentLower, kw) {
			logger.Debug("KeywordFilterProcessor matched exact keyword", "processor", k.name, "item_id", item.ID, "keyword", exactKeyword)
			item.SetMatchedKeywords(exactKeyword)
			return nil
		}
	}

	if cfg.TitleBypass {
		if matched := k.matcher.MatchTitle(title); matched != "" {
			logger.Info("KeywordFilterProcessor matched via title bypass", "processor", k.name, "item_id", item.ID, "keyword", matched)
			item.SetMatchedKeywords(matched)
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
