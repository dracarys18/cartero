package filters

import (
	"context"
	"maps"
	"slices"
	"strings"

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

	stemmedKeywords := make(map[string]string, len(cfg.Keywords))

	for _, kw := range cfg.Keywords {
		kwLower := strings.ToLower(kw)
		tokens := analyzeText(kwLower)
		for token := range maps.Keys(tokens) {
			stemmedKeywords[token] = kw
		}
	}

	exactKeywords := make([]string, len(cfg.ExactKeyword))
	for i, kw := range cfg.ExactKeyword {
		exactKeywords[i] = strings.ToLower(kw)
	}

	if len(stemmedKeywords) == 0 && len(exactKeywords) == 0 {
		return nil
	}

	title := item.GetTitle()

	var content string
	if article := item.GetArticle(); article != nil {
		content = article.Text
	}

	titleLower := strings.ToLower(title)
	contentLower := strings.ToLower(content)

	for exactKeyword := range slices.Values(exactKeywords) {
		if strings.Contains(titleLower, exactKeyword) || strings.Contains(contentLower, exactKeyword) {
			logger.Debug("KeywordFilterProcessor matched exact keyword", "processor", k.name, "item_id", item.ID, "keyword", exactKeyword)
			item.SetMatchedKeywords(exactKeyword)
			return nil
		}
	}

	if cfg.TitleBypass {
		stemmedTitle := analyzeText(titleLower)
		for stemmed, original := range stemmedKeywords {
			if _, exists := stemmedTitle[stemmed]; exists {
				logger.Info("KeywordFilterProcessor matched via title bypass", "processor", k.name, "item_id", item.ID, "keyword", original)
				item.SetMatchedKeywords(original)
				return nil
			}
		}
	}

	fullText := titleLower + " " + contentLower
	density, matchedKeywords := calculateKeywordDensity(fullText, stemmedKeywords)

	logger.Debug("KeywordFilterProcessor density calculated", "processor", k.name, "item_id", item.ID, "density", density*100, "keywords", matchedKeywords, "threshold", cfg.KeywordThreshold*100)

	if density >= cfg.KeywordThreshold {
		logger.Info("KeywordFilterProcessor matched via density", "processor", k.name, "item_id", item.ID, "density", density*100, "keywords", matchedKeywords)
		item.SetMatchedKeywords(matchedKeywords)
		return nil
	}

	logger.Info("KeywordFilterProcessor rejected item", "processor", k.name, "item_id", item.ID, "density", density*100, "threshold", cfg.KeywordThreshold*100)
	return types.NewFilteredError(k.name, item.ID, "keyword density below threshold").
		WithDetail("density_percentage", density*100).
		WithDetail("threshold_percentage", cfg.KeywordThreshold*100)
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

func calculateKeywordDensity(text string, stemmedKeywords map[string]string) (float64, string) {
	if text == "" {
		return 0, ""
	}

	tokenStream := analyzer.Analyze([]byte(text))
	if len(tokenStream) == 0 {
		return 0, ""
	}

	keywordCounts := countKeywordOccurrences(tokenStream, stemmedKeywords)
	if len(keywordCounts) == 0 {
		return 0, ""
	}

	topKeyword := findTopKeyword(keywordCounts)
	density := float64(keywordCounts[topKeyword]) / float64(len(tokenStream))

	return density, topKeyword
}

func countKeywordOccurrences(tokenStream analysis.TokenStream, stemmedKeywords map[string]string) map[string]int {
	counts := make(map[string]int)
	for _, token := range tokenStream {
		if keyword, exists := stemmedKeywords[string(token.Term)]; exists {
			counts[keyword]++
		}
	}
	return counts
}

func findTopKeyword(keywordCounts map[string]int) string {
	var topKeyword string
	maxCount := 0

	for keyword, count := range keywordCounts {
		if count > maxCount {
			maxCount = count
			topKeyword = keyword
		}
	}

	return topKeyword
}
