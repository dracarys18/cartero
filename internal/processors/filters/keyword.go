package filters

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
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
	name                 string
	stemmedKeywords      map[string]bool
	stemmedExactKeywords map[string]bool
	mode                 string
	threshold            float64
}

func NewKeywordFilterProcessor(name string, s config.KeywordFilterSettings) *KeywordFilterProcessor {
	stemmedKeywords := make(map[string]bool, len(s.Keywords))
	stemmedExactKeywords := make(map[string]bool, len(s.ExactKeyword))
	for _, kw := range s.Keywords {
		tokens := analyzeText(strings.ToLower(kw))
		for token := range maps.Keys(tokens) {
			stemmedKeywords[token] = true
		}
	}

	for _, kw := range s.ExactKeyword {
		tokens := analyzeText(strings.ToLower(kw))
		for token := range maps.Keys(tokens) {
			stemmedExactKeywords[token] = true
		}
	}

	return &KeywordFilterProcessor{
		name:                 name,
		stemmedKeywords:      stemmedKeywords,
		mode:                 s.Mode,
		stemmedExactKeywords: stemmedExactKeywords,
		threshold:            s.KeywordThreshold,
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
	stemmmedTitle := analyzeText(title)
	if !ok {
		return nil
	}
	content := item.TextContent
	stemmedContent := analyzeText(content)

	for token := range maps.Keys(k.stemmedExactKeywords) {
		_, tileExist := stemmmedTitle[token]
		_, contentExists := stemmedContent[token]

		if tileExist || contentExists {
			slog.Debug("KeywordFilterProcessor item matched with the exact keyword", "processor", k.name, "item_id", item.ID)
			return nil
		}
	}

	contentMatches := make(map[string]bool)
	titleMatches := make(map[string]bool)
	totalTitleMatches := 0

	for token := range maps.Keys(k.stemmedKeywords) {
		if _, exists := stemmmedTitle[token]; exists {
			totalTitleMatches++
			titleMatches[token] = true
		}

		if _, exists := stemmedContent[token]; exists {
			contentMatches[token] = true
		}
	}

	contentScore := float64(len(contentMatches)) / float64(totalInterestCount)
	score := contentScore
	if len(titleMatches) > 0 {
		score = score * 0.10 * float64(len(titleMatches))
	}

	matches := score >= k.threshold
	titleDensity := float64(len(stemmmedTitle)) / float64(totalTitleMatches)

	if titleDensity > 0.30 && totalTitleMatches >= 2 {
		matches = true
	}

	switch k.mode {
	case "include":
		if !matches {
			return fmt.Errorf("KeywordFilterProcessor %s: item %s coverage %.2f%% is below threshold %.2f%%",
				k.name, item.ID, score*100, k.threshold*100)
		}
	case "exclude":
		if matches {
			return fmt.Errorf("KeywordFilterProcessor %s: item %s coverage %.2f%% exceeds exclusion threshold",
				k.name, item.ID, score*100)
		}
	}

	slog.Debug("KeywordFilterProcessor item coverage", "processor", k.name, "item_id", item.ID, "coverage", score*100)
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
