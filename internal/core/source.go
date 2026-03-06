package core

import (
	"cartero/internal/types"
	"cartero/internal/utils"
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
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
	logger := state.GetLogger()
	q := state.GetQueue()
	sourceStream := q.SourceStream()

	if err := sr.Source.Publish(ctx, state); err != nil {
		return err
	}

	itemCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		envelopes, ids, err := q.Consume(ctx, sourceStream)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				logger.Info("Source finished processing items", "source", sr.Source.Name(), "count", itemCount)
				return nil
			}
			return err
		}

		if len(envelopes) == 0 {
			logger.Info("Source finished processing items", "source", sr.Source.Name(), "count", itemCount)
			return nil
		}

		for i, env := range envelopes {
			itemCount++
			if err := sr.processItem(ctx, state, env.Item); err != nil {
				return err
			}
			if err := q.Ack(ctx, sourceStream, ids[i]); err != nil {
				logger.Error("Failed to ack source message", "source", sr.Source.Name(), "id", ids[i], "error", err)
			}
		}
	}
}

func (sr *SourceRoute) processItem(ctx context.Context, state types.StateAccessor, item *types.Item) error {
	store := state.GetStorage()
	chain := state.GetChain()
	logger := state.GetLogger()
	q := state.GetQueue()

	filteredTargets := sr.filterTargets(ctx, state, item)
	if len(filteredTargets) == 0 {
		logger.Debug("No targets to publish to after filtering published targets", "item_id", item.ID)
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

	processedStream := q.ProcessedStream()
	if err := q.Publish(ctx, processedStream, item, targetNames); err != nil {
		logger.Error("Failed to publish to processed stream", "item_id", item.ID, "error", err)
		return err
	}

	envelopes, ids, err := q.Consume(ctx, processedStream)
	if err != nil {
		logger.Error("Failed to consume from processed stream", "item_id", item.ID, "error", err)
		return err
	}

	for i, env := range envelopes {
		deliveryTargets := sr.resolveTargets(env.Targets)
		if err := deliveryTargets.Process(ctx, state, env.Item, logger); err != nil {
			logger.Error("Failed to deliver item to targets", "item_id", env.Item.ID, "error", err)
			return err
		}
		if err := q.Ack(ctx, processedStream, ids[i]); err != nil {
			logger.Error("Failed to ack processed message", "item_id", env.Item.ID, "id", ids[i], "error", err)
		}
	}

	return nil
}

func (sr *SourceRoute) resolveTargets(names []string) Targets {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}

	var result Targets
	for _, t := range sr.Targets {
		if nameSet[t.Name()] {
			result = append(result, t)
		}
	}
	return result
}
