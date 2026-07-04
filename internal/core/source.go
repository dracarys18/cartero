package core

import (
	"cartero/internal/types"
	"context"
)

type SourceRoute struct {
	Source  types.Source
	Targets Targets
}

func (sr *SourceRoute) Process(ctx context.Context, state types.StateAccessor) ([]*types.Item, error) {
	return sr.Source.Fetch(ctx, state)
}
