package filters

import (
	"slices"
	"strings"

	"cartero/internal/core"

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

func KeywordFilter(name string, keywords []string, mode string) *FilterProcessor {
	stemmedKeywords := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		tokens := analyzeText(strings.ToLower(kw))
		stemmedKeywords = append(stemmedKeywords, tokens...)
	}

	return NewFilterProcessor(name, func(item *core.Item) bool {
		title, ok := item.Metadata["title"].(string)
		if !ok {
			return true
		}

		tokens := analyzeText(title)

		matches := false
		for _, token := range tokens {
			if slices.Contains(stemmedKeywords, token) {
				matches = true
				break
			}
		}

		switch mode {
		case "include":
			return matches
		case "exclude":
			return !matches
		default:
			return true
		}
	})
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
