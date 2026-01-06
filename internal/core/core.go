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

	pipeline.Run(ctx, state)
	return nil
}
