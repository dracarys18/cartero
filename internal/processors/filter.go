package processors

import (
	"cartero/internal/processors/filters"
)

type ScoreFilterProcessor = filters.ScoreFilterProcessor
type PublishedAtFilterProcessor = filters.PublishedAtFilterProcessor
type DedupeProcessor = filters.DedupeProcessor

var (
	NewScoreFilterProcessor       = filters.NewScoreFilterProcessor
	NewPublishedAtFilterProcessor = filters.NewPublishedAtFilterProcessor
	NewDedupeProcessor            = filters.NewDedupeProcessor
)
