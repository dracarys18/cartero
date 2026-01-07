package core

import (
	"cartero/internal/types"
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
)

type Targets []types.Target

func (t Targets) Process(ctx context.Context, state types.StateAccessor, item *types.Item, logger *slog.Logger) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(t))
	store := state.GetStorage()

	for i, target := range t {
		logger.Debug("Queuing item for target", "item_id", item.ID, "target", target.Name())
		wg.Add(1)
		go func(tgt types.Target, idx int) {
			defer wg.Done()

			if idx > 0 {
				if sleeper, ok := tgt.(interface{ Sleep(context.Context) error }); ok {
					if err := sleeper.Sleep(ctx); err != nil {
						return
					}
				}
			}

			if err := publishWithRetry(ctx, tgt, item, logger); err != nil {
				logger.Error("Failed to publish item to target after retries", "item_id", item.ID, "target", tgt.Name(), "error", err)
				errChan <- err
				return
			}

			logger.Info("Successfully published item to target", "item_id", item.ID, "target", tgt.Name())

			if err := store.Items().MarkPublished(ctx, item.ID, tgt.Name()); err != nil {
				logger.Error("Error marking item as published", "item_id", item.ID, "target", tgt.Name(), "error", err)
				errChan <- err
			}
		}(target, i)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func publishWithRetry(ctx context.Context, target types.Target, item *types.Item, logger *slog.Logger) error {
	maxRetries := 3
	var lastErr error

	if attempt := 0; attempt == 0 {
		logger.Debug("Publishing item to target", "item_id", item.ID, "target", target.Name())
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := target.Publish(ctx, item)

		if err == nil && result.Success {
			if attempt > 0 {
				logger.Info("Item published successfully on retry", "target", target.Name(), "item_id", item.ID, "attempt", attempt+1)
			}
			return nil
		}

		if err != nil {
			lastErr = fmt.Errorf("target %s error: %w", target.Name(), err)
		} else if !result.Success {
			lastErr = fmt.Errorf("target %s publish failed: %v", target.Name(), result.Error)
		}

		if attempt < maxRetries {
			waitDuration := time.Duration(math.Pow(2, float64(attempt))) * time.Second

			if result != nil && result.Metadata != nil {
				if retryAfter, ok := result.Metadata["retry_after"].(float64); ok && retryAfter > 0 {
					waitDuration = time.Duration(retryAfter*1000) * time.Millisecond
				}
			}

			logger.Warn("Publish attempt failed, retrying", "target", target.Name(), "attempt", attempt+1, "max_attempts", maxRetries+1, "wait_duration", waitDuration, "error", lastErr)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitDuration):
				continue
			}
		}
	}

	return fmt.Errorf("target %s: max retries (%d) exceeded: %w", target.Name(), maxRetries+1, lastErr)
}
