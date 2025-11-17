package filters

import (
	"cartero/internal/core"
)

func MetadataFilter(name string, key string, value interface{}) *FilterProcessor {
	return NewFilterProcessor(name, func(item *core.Item) bool {
		if item.Metadata == nil {
			return false
		}

		if val, exists := item.Metadata[key]; exists {
			return val == value
		}

		return false
	})
}
