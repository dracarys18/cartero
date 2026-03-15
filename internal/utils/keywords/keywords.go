package keywords

import (
	"regexp"
	"strings"
)

var tokenRegex = regexp.MustCompile(`[a-zA-Z0-9]+`)

type KeywordWithContext struct {
	Keyword string `json:"keyword"`
	Context string `json:"context_string"`
}

type Keywords struct {
	byNGram map[int][]string
}

func New(kws []KeywordWithContext) *Keywords {
	byNGram := make(map[int][]string)
	for _, kw := range kws {
		tokens := tokenRegex.FindAllString(kw.Keyword, -1)
		n := len(tokens)
		if n == 0 {
			continue
		}
		byNGram[n] = append(byNGram[n], strings.ToLower(kw.Keyword))
	}
	return &Keywords{byNGram: byNGram}
}

func (k *Keywords) Match(title, content string) (matched string, density float64) {
	fullText := title + " " + content
	tokens := tokenRegex.FindAllString(fullText, -1)
	lowered := make([]string, len(tokens))
	for i, t := range tokens {
		lowered[i] = strings.ToLower(t)
	}

	for n, kwList := range k.byNGram {
		ngrams := extractNGrams(lowered, n)
		if len(ngrams) == 0 {
			continue
		}
		counts := countMatches(ngrams, kwList)
		top, topCount := topEntry(counts)
		if topCount == 0 {
			continue
		}
		d := float64(topCount) / float64(len(ngrams))
		if d > density {
			density = d
			matched = top
		}
	}

	return matched, density
}

func (k *Keywords) MatchTitle(title string) string {
	tokens := tokenRegex.FindAllString(title, -1)
	lowered := make([]string, len(tokens))
	for i, t := range tokens {
		lowered[i] = strings.ToLower(t)
	}

	for n, kwList := range k.byNGram {
		ngrams := extractNGrams(lowered, n)
		counts := countMatches(ngrams, kwList)
		top, topCount := topEntry(counts)
		if topCount > 0 {
			return top
		}
	}
	return ""
}

func extractNGrams(tokens []string, n int) []string {
	if len(tokens) < n {
		return nil
	}
	ngrams := make([]string, 0, len(tokens)-n+1)
	for i := 0; i <= len(tokens)-n; i++ {
		ngrams = append(ngrams, strings.Join(tokens[i:i+n], " "))
	}
	return ngrams
}

func countMatches(ngrams []string, keywords []string) map[string]int {
	counts := make(map[string]int)
	kwSet := make(map[string]struct{}, len(keywords))
	for _, kw := range keywords {
		kwSet[kw] = struct{}{}
	}
	for _, ng := range ngrams {
		if _, ok := kwSet[ng]; ok {
			counts[ng]++
		}
	}
	return counts
}

func topEntry(counts map[string]int) (string, int) {
	var top string
	max := 0
	for k, v := range counts {
		if v > max {
			max = v
			top = k
		}
	}
	return top, max
}
