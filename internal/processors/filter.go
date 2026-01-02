package processors

import (
	"cartero/internal/processors/filters"
)

type ScoreFilterProcessor = filters.ScoreFilterProcessor
type KeywordFilterProcessor = filters.KeywordFilterProcessor
type MetadataFilterProcessor = filters.MetadataFilterProcessor
type DedupeProcessor = filters.DedupeProcessor
type ContentDedupeProcessor = filters.ContentDedupeProcessor
type RateLimitProcessor = filters.RateLimitProcessor
type TokenBucketProcessor = filters.TokenBucketProcessor

var (
	NewScoreFilterProcessor    = filters.NewScoreFilterProcessor
	NewKeywordFilterProcessor  = filters.NewKeywordFilterProcessor
	NewMetadataFilterProcessor = filters.NewMetadataFilterProcessor
	MinScoreFilter             = filters.MinScoreFilter
	KeywordFilter              = filters.KeywordFilter
	MetadataFilter             = filters.MetadataFilter
	NewDedupeProcessor         = filters.NewDedupeProcessor
	NewContentDedupeProcessor  = filters.NewContentDedupeProcessor
	NewRateLimitProcessor      = filters.NewRateLimitProcessor
	NewTokenBucketProcessor    = filters.NewTokenBucketProcessor
)
