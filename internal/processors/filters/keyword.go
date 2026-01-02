package filters

import (
	"context"
	"fmt"
	"slices"
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
	stemmedKeywords []string
	mode            string
}

func NewKeywordFilterProcessor(name string, keywords []string, mode string) *KeywordFilterProcessor {
	stemmedKeywords := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		tokens := analyzeText(strings.ToLower(kw))
		stemmedKeywords = append(stemmedKeywords, tokens...)
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
	title, ok := item.Metadata["title"].(string)
	if !ok {
		return nil
	}

	tokens := analyzeText(title)

	matches := false
	for _, token := range tokens {
		if slices.Contains(k.stemmedKeywords, token) {
			matches = true
			break
		}
	}

	switch k.mode {
	case "include":
		if !matches {
			return fmt.Errorf("KeywordFilterProcessor %s: item %s does not contain any keywords (include mode)", k.name, item.ID)
		}
	case "exclude":
		if matches {
			return fmt.Errorf("KeywordFilterProcessor %s: item %s contains excluded keywords", k.name, item.ID)
		}
	}

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
