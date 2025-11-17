package filters

import (
	"cartero/internal/core"
)

func MinScoreFilter(name string, minScore int) *FilterProcessor {
	return NewFilterProcessor(name, func(item *core.Item) bool {
		if score, ok := item.Metadata["score"].(int); ok {
			return score >= minScore
		}
		return true
	})
}
