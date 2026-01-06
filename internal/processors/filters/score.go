package filters

import (
	"context"
	"fmt"

	"cartero/internal/processors/names"
	"cartero/internal/types"
)

type ScoreFilterProcessor struct {
	name string
}

func NewScoreFilterProcessor(name string) *ScoreFilterProcessor {
	return &ScoreFilterProcessor{
		name: name,
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

func (s *ScoreFilterProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[s.name].Settings.ScoreFilterSettings
	minScore := cfg.MinScore

	if score, ok := item.Metadata["score"].(int); ok {
		if score < minScore {
			return fmt.Errorf("ScoreFilterProcessor %s: item %s score %d is below minimum %d", s.name, item.ID, score, minScore)
		}
	}
	return nil
}

func MinScoreFilter(name string) *ScoreFilterProcessor {
	return NewScoreFilterProcessor(name)
}
