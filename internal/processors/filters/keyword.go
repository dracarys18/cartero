package filters

import (
	"context"
	"maps"
	"slices"
	"strings"

	"cartero/internal/config"
	"cartero/internal/processors/names"
	"cartero/internal/types"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/registry"
)

var analyzer analysis.Analyzer

func init() {
	cache := registry.NewCache()
	var err error
	analyzer, err = en.AnalyzerConstructor(nil, cache)
	if err != nil {
		panic(err)
	}
}

type KeywordFilterProcessor struct {
	name string
}

func NewKeywordFilterProcessor(name string) *KeywordFilterProcessor {
	return &KeywordFilterProcessor{
		name: name,
	}
}

func (k *KeywordFilterProcessor) Name() string {
	return k.name
}

func (k *KeywordFilterProcessor) DependsOn() []string {
	return []string{
		names.ExtractText,
	}
}

func (k *KeywordFilterProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	stateConfig := st.GetConfig()
	cfg := stateConfig.Processors[k.name].Settings.KeywordFilterSettings
	logger := st.GetLogger()

	stemmedKeywords := make(map[string]bool, len(cfg.Keywords))
	for _, kw := range cfg.Keywords {
		tokens := analyzeText(strings.ToLower(kw))
		for token := range maps.Keys(tokens) {
			stemmedKeywords[token] = true
		}
	}

	exactKeywords := make([]string, len(cfg.ExactKeyword))
	for i, kw := range cfg.ExactKeyword {
		exactKeywords[i] = strings.ToLower(kw)
	}

	totalInterestCount := len(stemmedKeywords)
	if totalInterestCount == 0 {
		return nil
	}

	title, ok := item.Metadata["title"].(string)
	if !ok {
		title = ""
	}

	title = strings.ToLower(title)
	content := strings.ToLower(item.TextContent)

	for exactKeyword := range slices.Values(exactKeywords) {
		if strings.Contains(title, exactKeyword) || strings.Contains(content, exactKeyword) {
			logger.Debug("KeywordFilterProcessor item matched with the exact keyword", "processor", k.name, "item_id", item.ID, "keyword", exactKeyword)
			return nil
		}
	}

	stemmedTitle := analyzeText(title)
	stemmedContent := analyzeText(content)

	contentMatches := make(map[string]bool)
	totalTitleMatches := 0

	for token := range maps.Keys(stemmedKeywords) {
		if _, exists := stemmedTitle[token]; exists {
			totalTitleMatches++
		}

		if _, exists := stemmedContent[token]; exists {
			contentMatches[token] = true
		}
	}

	contentScore := float64(len(contentMatches)) / float64(totalInterestCount)
	matches := contentScore >= cfg.KeywordThreshold
	if totalTitleMatches >= 1 {
		matches = true
		logger.Info("KeywordFilterProcessor item matched due to title keyword presence", "processor", k.name, "item_id", item.ID, "title_matches", totalTitleMatches)
	}

	switch cfg.Mode {
	case "include":
		if !matches {
			logger.Info("KeywordFilterProcessor rejected item", "processor", k.name, "item_id", item.ID, "coverage", contentScore*100, "threshold", cfg.KeywordThreshold*100)
			return types.NewFilteredError(k.name, item.ID, "keyword coverage below threshold").
				WithDetail("coverage_percentage", contentScore*100).
				WithDetail("threshold_percentage", cfg.KeywordThreshold*100)
		}
	case "exclude":
		if matches {
			logger.Info("KeywordFilterProcessor rejected item", "processor", k.name, "item_id", item.ID, "coverage", contentScore*100)
			return types.NewFilteredError(k.name, item.ID, "keyword coverage exceeds exclusion threshold").
				WithDetail("coverage_percentage", contentScore*100)
		}
	}

	logger.Debug("KeywordFilterProcessor item coverage", "processor", k.name, "item_id", item.ID, "coverage", contentScore*100)
	return nil
}

func analyzeText(text string) map[string]bool {
	if text == "" {
		return nil
	}

	tokenStream := analyzer.Analyze([]byte(text))

	tokens := make(map[string]bool, len(tokenStream))
	for _, token := range tokenStream {
		tokens[string(token.Term)] = true
	}

	return tokens
}
