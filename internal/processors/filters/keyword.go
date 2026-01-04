package filters

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"cartero/internal/config"
	"cartero/internal/core"
	"cartero/internal/processors/names"

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
	name            string
	stemmedKeywords map[string]bool
	exactKeywords   []string
	mode            string
	threshold       float64
}

func NewKeywordFilterProcessor(name string, s config.KeywordFilterSettings) *KeywordFilterProcessor {
	stemmedKeywords := make(map[string]bool, len(s.Keywords))
	for _, kw := range s.Keywords {
		tokens := analyzeText(strings.ToLower(kw))
		for token := range maps.Keys(tokens) {
			stemmedKeywords[token] = true
		}
	}

	// Store exact keywords as lowercase strings (not stemmed)
	exactKeywords := make([]string, len(s.ExactKeyword))
	for i, kw := range s.ExactKeyword {
		exactKeywords[i] = strings.ToLower(kw)
	}

	return &KeywordFilterProcessor{
		name:            name,
		stemmedKeywords: stemmedKeywords,
		mode:            s.Mode,
		exactKeywords:   exactKeywords,
		threshold:       s.KeywordThreshold,
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

// Filters if the title if more than 30% of the title is filled with the keywords. Or 15% of the content is filled with keywords.
// Adds a special weight if keywords are found in the title.
func (k *KeywordFilterProcessor) Process(ctx context.Context, item *core.Item) error {
	totalInterestCount := len(k.stemmedKeywords)
	if totalInterestCount == 0 {
		return nil
	}

	title, ok := item.Metadata["title"].(string)
	if !ok {
		title = ""
	}

	title = strings.ToLower(title)
	content := strings.ToLower(item.TextContent)

	for exactKeyword := range slices.Values(k.exactKeywords) {
		if strings.Contains(title, exactKeyword) || strings.Contains(content, exactKeyword) {
			slog.Debug("KeywordFilterProcessor item matched with the exact keyword", "processor", k.name, "item_id", item.ID, "keyword", exactKeyword)
			return nil
		}
	}

	stemmmedTitle := analyzeText(title)
	stemmedContent := analyzeText(content)

	contentMatches := make(map[string]bool)
	totalTitleMatches := 0

	for token := range maps.Keys(k.stemmedKeywords) {
		if _, exists := stemmmedTitle[token]; exists {
			totalTitleMatches++
		}

		if _, exists := stemmedContent[token]; exists {
			contentMatches[token] = true
		}
	}

	contentScore := float64(len(contentMatches)) / float64(totalInterestCount)
	matches := contentScore >= k.threshold
	if totalTitleMatches >= 1 {
		matches = true
		slog.Info("KeywordFilterProcessor item matched due to title keyword presence", "processor", k.name, "item_id", item.ID, "title_matches", totalTitleMatches)
	}

	switch k.mode {
	case "include":
		if !matches {
			return fmt.Errorf("KeywordFilterProcessor %s: item %s coverage %.2f%% is below threshold %.2f%%",
				k.name, item.ID, contentScore*100, k.threshold*100)
		}
	case "exclude":
		if matches {
			return fmt.Errorf("KeywordFilterProcessor %s: item %s coverage %.2f%% exceeds exclusion threshold",
				k.name, item.ID, contentScore*100)
		}
	}

	slog.Debug("KeywordFilterProcessor item coverage", "processor", k.name, "item_id", item.ID, "coverage", contentScore*100)
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
