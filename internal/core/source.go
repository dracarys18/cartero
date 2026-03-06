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

func (sr *SourceRoute) filterTargets(ctx context.Context, state types.StateAccessor, item *types.Item) Targets {
	store := state.GetStorage()
	items := store.Items()

	predicate := func(target types.Target) bool {
		published, _ := items.IsPublished(ctx, item.ID, target.Name())
		return !published
	}

	return Targets(utils.FilterArray([]types.Target(sr.Targets), predicate))
}

func (sr *SourceRoute) Process(ctx context.Context, state types.StateAccessor) error {
	return sr.Source.Publish(ctx, state)
}

func (sr *SourceRoute) processItem(ctx context.Context, state types.StateAccessor, item *types.Item) error {
	store := state.GetStorage()
	chain := state.GetChain()
	logger := state.GetLogger()
	q := state.GetQueue()

	filteredTargets := sr.filterTargets(ctx, state, item)
	if len(filteredTargets) == 0 {
		return nil
	}

	logger.Info("Running processors", "item_id", item.ID)

	if err := chain.Execute(ctx, state, item); err != nil {
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

	targetNames := make([]string, len(filteredTargets))
	for i, t := range filteredTargets {
		targetNames[i] = t.Name()
	}

	env := types.Envelope{Item: item, Targets: targetNames}
	if err := q.Publish(ctx, q.ProcessedStream(), env); err != nil {
		logger.Error("Failed to publish to processed stream", "item_id", item.ID, "error", err)
		return err
	}

	return nil
}

func (sr *SourceRoute) resolveTargets(names []string) Targets {
	nameSet := make(map[string]struct{}, len(names))
	for _, n := range names {
		nameSet[n] = struct{}{}
	}

	var result Targets
	for _, t := range sr.Targets {
		if _, ok := nameSet[t.Name()]; ok {
			result = append(result, t)
		}
	}
	return result
}
