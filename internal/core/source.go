package core

import (
	"cartero/internal/types"
	"context"
)

type SourceRoute struct {
	Source  types.Source
	Targets Targets
}

func (sr *SourceRoute) Process(ctx context.Context, state types.StateAccessor) error {
	return sr.Source.Publish(ctx, state)
}
