package core

import (
	"context"
	"fmt"

	"cartero/internal/types"
)

func ProcessCore(ctx context.Context, state types.StateAccessor) error {
	pipelineIface := state.GetPipeline()
	pipeline := pipelineIface.(*Pipeline)

	chainIface := state.GetChain()
	chain := chainIface.(types.ProcessorChain)

	registry := state.GetRegistry()

	if err := registry.InitializeAll(ctx); err != nil {
		return fmt.Errorf("component initialization failed: %w", err)
	}

	for _, route := range pipeline.GetRoutes() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		items, errs := route.Source.Fetch(ctx)

		for {
			select {
			case item, ok := <-items:
				if !ok {
					items = nil
					continue
				}

				if err := chain.Execute(ctx, item); err != nil {
					continue
				}

				for _, target := range route.Targets {
					target.Publish(ctx, item)
				}

			case err, ok := <-errs:
				if !ok {
					errs = nil
					break
				}
				_ = err

			case <-ctx.Done():
				return ctx.Err()
			}

			if items == nil && errs == nil {
				break
			}
		}
	}

	return nil
}
