package core

import (
	"cartero/internal/types"
	"cartero/internal/utils"
	"context"
)

type SourceRoute struct {
	Source  types.Source
	Targets Targets
}

func (sr *SourceRoute) FilterTargets(ctx context.Context, state types.StateAccessor, item *types.Item) Targets {
	store := state.GetStorage()
	items := store.Items()

	predicate := func(target types.Target) bool {
		published, _ := items.IsPublished(ctx, item.ID, target.Name())
		return !published
	}

	return Targets(utils.FilterArray([]types.Target(sr.Targets), predicate))
}

func (sr *SourceRoute) Process(ctx context.Context, state types.StateAccessor) error {
	logger := state.GetLogger()
	logger.Debug("Source starting to fetch items", "source", sr.Source.Name())
	itemChan, errChan := sr.Source.Fetch(ctx, state)
	itemCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			if err != nil {
				return err
			}
		case item, ok := <-itemChan:
			if !ok {
				logger.Info("Source finished processing items", "source", sr.Source.Name(), "count", itemCount)
				return nil
			}

			itemCount++

			if err := sr.processItem(ctx, state, item); err != nil {
				return err
			}
		}
	}
}

func (sr *SourceRoute) processItem(ctx context.Context, state types.StateAccessor, item *types.Item) error {
	store := state.GetStorage()
	chain := state.GetChain()
	logger := state.GetLogger()

	filteredTargets := sr.FilterTargets(ctx, state, item)

	if len(filteredTargets) == 0 {
		logger.Debug("No targets to publish to after filtering published targets", "item_id", item.ID)
		return nil
	}

	logger.Info("Running processors", "item_id", item.ID)

	err := chain.Execute(ctx, state, item)
	if err != nil {
		if types.IsFiltered(err) {
			logger.Info("Item skipped due to filter", "item_id", item.ID, "filter_reason", err.Error())
			return nil
		}
		logger.Error("Processing failed", "item_id", item.ID, "error", err)
		return err
	}

	if err := store.Items().Store(ctx, item); err != nil {
		logger.Error("Error storing item", "source", sr.Source.Name(), "item_id", item.ID, "error", err)
		return err
	}
	logger.Debug("Stored new item", "source", sr.Source.Name(), "item_id", item.ID)
	logger.Debug("Publishing to targets", "item_id", item.ID, "target_count", len(filteredTargets))
	return filteredTargets.Process(ctx, state, item, logger)
}
