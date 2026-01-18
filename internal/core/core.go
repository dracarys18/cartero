package core

import (
	"context"
	"fmt"

	"cartero/internal/types"
)

func ProcessCore(ctx context.Context, state types.StateAccessor) error {
	logger := state.GetLogger()
	pipeline := state.GetPipeline().(*Pipeline)

	logger.Info("Running pipeline...")

	if err := pipeline.Run(ctx, state); err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}
	return nil
}
