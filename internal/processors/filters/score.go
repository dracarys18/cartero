package filters

import (
	"context"

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
	logger := st.GetLogger()
	minScore := cfg.MinScore

	if score, ok := item.Metadata["score"].(int); ok {
		if score < minScore {
			logger.Info("ScoreFilterProcessor rejected item", "processor", s.name, "item_id", item.ID, "score", score, "min_score", minScore)
			return types.NewFilteredError(s.name, item.ID, "score below minimum").
				WithDetail("score", score).
				WithDetail("min_score", minScore)
		}
	}
	return nil
}

func MinScoreFilter(name string) *ScoreFilterProcessor {
	return NewScoreFilterProcessor(name)
}
