package processors

import (
	"cartero/internal/processors/filters"
)

type FilterProcessor = filters.FilterProcessor
type FilterFunc = filters.FilterFunc

var (
	NewFilterProcessor = filters.NewFilterProcessor
	MinScoreFilter     = filters.MinScoreFilter
	KeywordFilter      = filters.KeywordFilter
	MetadataFilter     = filters.MetadataFilter
)
