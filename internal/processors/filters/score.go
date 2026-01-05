package filters

import (
	"context"
	"fmt"

	"cartero/internal/core"
	"cartero/internal/processors/names"
)

type ScoreFilterProcessor struct {
	name     string
	minScore int
}

func NewScoreFilterProcessor(name string, minScore int) *ScoreFilterProcessor {
	return &ScoreFilterProcessor{
		name:     name,
		minScore: minScore,
	}
}

func (s *ScoreFilterProcessor) Name() string {
	return s.name
}

func (s *ScoreFilterProcessor) DependsOn() []string {
	return []string{
		names.Dedupe,
	}
}

func (s *ScoreFilterProcessor) Process(ctx context.Context, item *core.Item) error {
	if score, ok := item.Metadata["score"].(int); ok {
		if score < s.minScore {
			return fmt.Errorf("ScoreFilterProcessor %s: item %s score %d is below minimum %d", s.name, item.ID, score, s.minScore)
		}
	}
	return nil
}

func MinScoreFilter(name string, minScore int) *ScoreFilterProcessor {
	return NewScoreFilterProcessor(name, minScore)
}
