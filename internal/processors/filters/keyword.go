package filters

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"cartero/internal/processors/names"
	"cartero/internal/types"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

var wordRegex = regexp.MustCompile(`\b[a-zA-Z0-9]+\b`)

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
		if strings.Contains(titleLower, strings.ToLower(exactKeyword)) || strings.Contains(contentLower, strings.ToLower(exactKeyword)) {
			logger.Debug("KeywordFilterProcessor matched exact keyword", "processor", k.name, "item_id", item.ID, "keyword", exactKeyword)
			item.SetMatchedKeywords(exactKeyword)
			return nil
		}
	}

	if cfg.TitleBypass {
		titleWords := extractWords(title)
		titleCounts := countKeywordOccurrencesExact(titleWords, cfg.Keywords)
		if len(titleCounts) > 0 {
			matched := findTopKeyword(titleCounts)
			logger.Info("KeywordFilterProcessor matched via title bypass", "processor", k.name, "item_id", item.ID, "keyword", matched)
			item.SetMatchedKeywords(matched)
			return nil
		}
	}

	fullText := title + " " + content
	density, matchedKeywords := calculateKeywordDensity(fullText, cfg.Keywords)

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

func extractWords(text string) []string {
	return wordRegex.FindAllString(text, -1)
}

func matchKeywordExact(keyword, word string) bool {
	return strings.EqualFold(keyword, word)
}

func matchKeywordFuzzy(keyword, word string) bool {
	kw := strings.ToLower(keyword)
	w := strings.ToLower(word)

	if kw == w {
		return true
	}

	minLen := float64(min(len(kw), len(w)))
	maxLen := float64(max(len(kw), len(w)))

	if minLen == 0 || maxLen == 0 {
		return false
	}

	lengthRatio := minLen / maxLen
	isPrefix := strings.HasPrefix(w, kw) || strings.HasPrefix(kw, w)

	if isPrefix {
		if lengthRatio < 0.65 {
			return false
		}
	} else {
		if lengthRatio < 0.90 {
			return false
		}
	}

	distance := fuzzy.LevenshteinDistance(kw, w)

	maxDistance := 1
	if len(kw) > 5 {
		maxDistance = 2
	}

	return distance <= maxDistance
}

func countKeywordOccurrencesExact(words []string, keywords []string) map[string]int {
	counts := make(map[string]int)

	for _, word := range words {
		for _, kw := range keywords {
			if matchKeywordExact(kw, word) {
				counts[kw]++
			}
		}
	}

	return counts
}

func countKeywordOccurrencesFuzzy(words []string, keywords []string) map[string]int {
	counts := make(map[string]int)

	for _, word := range words {
		for _, kw := range keywords {
			if matchKeywordFuzzy(kw, word) {
				counts[kw]++
			}
		}
	}

	return counts
}

func calculateKeywordDensity(text string, keywords []string) (float64, string) {
	words := extractWords(text)
	if len(words) == 0 {
		return 0, ""
	}

	keywordCounts := countKeywordOccurrencesFuzzy(words, keywords)
	if len(keywordCounts) == 0 {
		return 0, ""
	}

	topKeyword := findTopKeyword(keywordCounts)
	density := float64(keywordCounts[topKeyword]) / float64(len(words))

	return density, topKeyword
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
