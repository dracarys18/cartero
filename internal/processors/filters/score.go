package filters

import (
	"context"

	"cartero/internal/config"
	"cartero/internal/processors/names"
	"cartero/internal/types"
)

type ScoreFilterProcessor struct {
	name     string
	settings config.ScoreFilterSettings
}

func NewScoreFilterProcessor(name string, settings config.ScoreFilterSettings) *ScoreFilterProcessor {
	return &ScoreFilterProcessor{
		name:     name,
		settings: settings,
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

func (s *ScoreFilterProcessor) Process(_ context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	logger := st.GetLogger()
	minScore := s.settings.MinScore

	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		if score, ok := item.Metadata["score"].(int); ok && score < minScore {
			logger.Debug("score_filter: dropped item", "processor", s.name, "item_id", item.ID, "score", score, "min_score", minScore)
			continue
		}
		out = append(out, item)
	}
	return out, nil
}
