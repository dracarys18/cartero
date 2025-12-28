package processors

import (
	"cartero/internal/processors/filters"
)

type FilterProcessor = filters.FilterProcessor
type FilterFunc = filters.FilterFunc
type DedupeProcessor = filters.DedupeProcessor
type ContentDedupeProcessor = filters.ContentDedupeProcessor
type RateLimitProcessor = filters.RateLimitProcessor
type TokenBucketProcessor = filters.TokenBucketProcessor

var (
	NewFilterProcessor        = filters.NewFilterProcessor
	MinScoreFilter            = filters.MinScoreFilter
	KeywordFilter             = filters.KeywordFilter
	MetadataFilter            = filters.MetadataFilter
	NewDedupeProcessor        = filters.NewDedupeProcessor
	NewContentDedupeProcessor = filters.NewContentDedupeProcessor
	NewRateLimitProcessor     = filters.NewRateLimitProcessor
	NewTokenBucketProcessor   = filters.NewTokenBucketProcessor
)
