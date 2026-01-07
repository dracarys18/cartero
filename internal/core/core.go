package core

import (
	"context"
	"fmt"

	"cartero/internal/types"
)

func ProcessCore(ctx context.Context, state types.StateAccessor) error {
	logger := state.GetLogger()
	pipeline := state.GetPipeline().(*Pipeline)

	registry := state.GetRegistry()

	logger.Info("Initializing components...")
	if err := registry.InitializeAll(ctx); err != nil {
		return fmt.Errorf("component initialization failed: %w", err)
	}

	logger.Info("Initializing pipeline...")

	if err := pipeline.Run(ctx, state); err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}
	return nil
}
