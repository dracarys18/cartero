package processors

import (
	"cartero/internal/processors/filters"
)

type ScoreFilterProcessor = filters.ScoreFilterProcessor
type KeywordFilterProcessor = filters.KeywordFilterProcessor
type MetadataFilterProcessor = filters.MetadataFilterProcessor
type PublishedAtFilterProcessor = filters.PublishedAtFilterProcessor
type DedupeProcessor = filters.DedupeProcessor
type EmbedDedupeProcessor = filters.EmbedDedupeProcessor
type RateLimitProcessor = filters.RateLimitProcessor
type TokenBucketProcessor = filters.TokenBucketProcessor

var (
	NewScoreFilterProcessor       = filters.NewScoreFilterProcessor
	NewKeywordFilterProcessor     = filters.NewKeywordFilterProcessor
	NewMetadataFilterProcessor    = filters.NewMetadataFilterProcessor
	NewPublishedAtFilterProcessor = filters.NewPublishedAtFilterProcessor
	MinScoreFilter                = filters.MinScoreFilter
	MetadataFilter                = filters.MetadataFilter
	PublishedAtFilter             = filters.PublishedAtFilter
	NewDedupeProcessor            = filters.NewDedupeProcessor
	NewEmbedDedupeProcessor       = filters.NewEmbedDedupeProcessor
	NewRateLimitProcessor         = filters.NewRateLimitProcessor
	NewTokenBucketProcessor       = filters.NewTokenBucketProcessor
)
