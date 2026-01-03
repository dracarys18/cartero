package filters

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	mode            string
}

func NewKeywordFilterProcessor(name string, keywords []string, mode string) *KeywordFilterProcessor {
	stemmedKeywords := make(map[string]bool, len(keywords))
	for _, kw := range keywords {
		tokens := analyzeText(strings.ToLower(kw))
		for _, token := range tokens {
			stemmedKeywords[token] = true
		}
	}

	return &KeywordFilterProcessor{
		name:            name,
		stemmedKeywords: stemmedKeywords,
		mode:            mode,
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

func (k *KeywordFilterProcessor) Process(ctx context.Context, item *core.Item) error {
	threshold := 0.15
	title, ok := item.Metadata["title"].(string)
	if !ok {
		return nil
	}

	text := fmt.Sprintf("%s %s", title, item.TextContent)
	tokens := analyzeText(text)

	uniqueMatches := make(map[string]bool)
	for _, token := range tokens {
		if _, exists := k.stemmedKeywords[token]; exists {
			uniqueMatches[token] = true
		}
	}

	totalInterestCount := len(k.stemmedKeywords)
	if totalInterestCount == 0 {
		return nil
	}

	coverageScore := float64(len(uniqueMatches)) / float64(totalInterestCount)

	matches := coverageScore >= threshold

	switch k.mode {
	case "include":
		if !matches {
			return fmt.Errorf("KeywordFilterProcessor %s: item %s coverage %.2f%% is below threshold %.2f%%",
				k.name, item.ID, coverageScore*100, threshold*100)
		}
	case "exclude":
		if matches {
			return fmt.Errorf("KeywordFilterProcessor %s: item %s coverage %.2f%% exceeds exclusion threshold",
				k.name, item.ID, coverageScore*100)
		}
	}

	log.Printf("KeywordFilterProcessor %s: item %s coverage %.2f%%",
		k.name, item.ID, coverageScore*100)
	return nil
}

func analyzeText(text string) []string {
	if text == "" {
		return nil
	}

	tokenStream := analyzer.Analyze([]byte(text))

	tokens := make([]string, 0, len(tokenStream))
	for _, token := range tokenStream {
		tokens = append(tokens, string(token.Term))
	}

	return tokens
}

func KeywordFilter(name string, keywords []string, mode string) *KeywordFilterProcessor {
	return NewKeywordFilterProcessor(name, keywords, mode)
}
