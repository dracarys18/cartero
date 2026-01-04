package core

import (
	"cartero/internal/storage"
	"cartero/internal/utils"
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
)

type SourceRoute struct {
	Source  Source
	Targets []Target
}

type Pipeline struct {
	routes             []SourceRoute
	processors         []Processor
	processorConfigs   map[string]ProcessorConfig
	processorExecutor  *ProcessorExecutor
	itemStore          storage.ItemStore
	initializedTargets map[string]bool
	mu                 sync.RWMutex
	running            bool
}

func NewPipeline(itemStore storage.ItemStore) *Pipeline {
	return &Pipeline{
		routes:             make([]SourceRoute, 0),
		processors:         make([]Processor, 0),
		processorConfigs:   make(map[string]ProcessorConfig),
		processorExecutor:  NewProcessorExecutor(),
		itemStore:          itemStore,
		initializedTargets: make(map[string]bool),
		running:            false,
	}
}

func (p *Pipeline) AddRoute(route SourceRoute) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.routes = append(p.routes, route)
	return p
}

func (p *Pipeline) AddProcessor(processor Processor) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processors = append(p.processors, processor)
	return p
}

func (p *Pipeline) AddProcessorWithConfig(processor Processor, config ProcessorConfig) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processors = append(p.processors, processor)
	p.processorConfigs[config.Name] = config
	p.processorExecutor.AddProcessor(config.Type, processor)
	return p
}

func (p *Pipeline) Initialize(ctx context.Context) error {
	slog.Info("Initializing pipeline", "routes", len(p.routes))

	if err := p.processorExecutor.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize processor executor: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, route := range p.routes {
		slog.Info("Initializing source", "source", route.Source.Name())
		if err := route.Source.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize source %s: %w", route.Source.Name(), err)
		}

		for _, target := range route.Targets {
			if p.initializedTargets[target.Name()] {
				continue
			}

			slog.Info("Initializing target", "target", target.Name(), "source", route.Source.Name())
			if err := target.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize target %s: %w", target.Name(), err)
			}

			p.initializedTargets[target.Name()] = true
		}
	}

	slog.Info("Pipeline initialization complete")
	return nil
}

func (p *Pipeline) Run(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pipeline already running")
	}
	p.running = true
	p.mu.Unlock()

	slog.Info("Starting pipeline execution", "sources", len(p.routes))

	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		slog.Info("Pipeline execution completed")
	}()

	var wg sync.WaitGroup

	for _, route := range p.routes {
		wg.Add(1)
		slog.Debug("Launching goroutine for source", "source", route.Source.Name())
		go func(r SourceRoute) {
			defer wg.Done()
			if err := p.processSource(ctx, r); err != nil {
				slog.Error("Error processing source", "source", r.Source.Name(), "error", err)
			} else {
				slog.Info("Source completed successfully", "source", r.Source.Name())
			}
		}(route)
	}

	wg.Wait()
	return nil
}

func (p *Pipeline) processSource(ctx context.Context, route SourceRoute) error {
	slog.Debug("Source starting to fetch items", "source", route.Source.Name())
	itemChan, errChan := route.Source.Fetch(ctx)

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
				slog.Info("Source finished processing items", "source", route.Source.Name(), "count", itemCount)
				return nil
			}

			itemCount++
			exists, err := p.itemStore.Exists(ctx, item.ID)
			if err != nil {
				return err
			}
			if !exists {
				if err := p.itemStore.Store(ctx, item); err != nil {
					slog.Error("Error storing item", "source", route.Source.Name(), "item_id", item.ID, "error", err)
					return err
				}
				slog.Debug("Stored new item", "source", route.Source.Name(), "item_id", item.ID)
			}

			if err := p.processItem(ctx, item, route); err != nil {
				return err
			}
		}
	}
}

func (p *Pipeline) processItem(ctx context.Context, item *Item, route SourceRoute) error {
	filterFunc := func(target Target) bool {
		published, err := p.itemStore.IsPublished(ctx, item.ID, target.Name())
		if err != nil {
			slog.Error("Error checking if item published", "item_id", item.ID, "target", target.Name(), "error", err)
			return false
		}
		if published {
			slog.Debug("Item already published to target, skipping", "item_id", item.ID, "target", target.Name())
			return false
		}
		return true
	}
	route.Targets = utils.FilterArray(route.Targets, filterFunc)

	if len(route.Targets) == 0 {
		slog.Debug("No targets to publish to after filtering published targets", "item_id", item.ID)
		return nil
	}

	slog.Debug("Running processors", "item_id", item.ID, "count", len(p.processors))

	err := p.processorExecutor.ExecuteProcessors(ctx, item)
	if err != nil {
		slog.Debug("Item filtered out during processing", "item_id", item.ID, "error", err)
		return nil
	}

	slog.Debug("All processors completed, publishing to targets", "item_id", item.ID, "targets", len(route.Targets))
	return p.publishToTargets(ctx, item, route)
}

func (p *Pipeline) publishToTargets(ctx context.Context, item *Item, route SourceRoute) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(route.Targets))

	for i, target := range route.Targets {
		slog.Debug("Queuing item for target", "item_id", item.ID, "target", target.Name())
		wg.Add(1)
		go func(t Target, idx int) {
			defer wg.Done()

			if idx > 0 {
				if sleeper, ok := t.(interface{ Sleep(context.Context) error }); ok {
					if err := sleeper.Sleep(ctx); err != nil {
						return
					}
				}
			}

			if err := p.publishWithRetry(ctx, t, item); err != nil {
				slog.Error("Failed to publish item to target after retries", "item_id", item.ID, "target", t.Name(), "error", err)
				errChan <- err
				return
			}

			slog.Info("Successfully published item to target", "item_id", item.ID, "target", t.Name())

			if err := p.itemStore.MarkPublished(ctx, item.ID, t.Name()); err != nil {
				slog.Error("Error marking item as published", "item_id", item.ID, "target", t.Name(), "error", err)
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

func (p *Pipeline) publishWithRetry(ctx context.Context, target Target, item *Item) error {
	maxRetries := 3
	var lastErr error

	if attempt := 0; attempt == 0 {
		slog.Debug("Publishing item to target", "item_id", item.ID, "target", target.Name())
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := target.Publish(ctx, item)

		if err == nil && result.Success {
			if attempt > 0 {
				slog.Info("Item published successfully on retry", "target", target.Name(), "item_id", item.ID, "attempt", attempt+1)
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

			slog.Warn("Publish attempt failed, retrying", "target", target.Name(), "attempt", attempt+1, "max_attempts", maxRetries+1, "wait_duration", waitDuration, "error", lastErr)

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

func (p *Pipeline) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down pipeline")
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error

	for _, route := range p.routes {
		slog.Debug("Shutting down source", "source", route.Source.Name())
		if err := route.Source.Shutdown(ctx); err != nil {
			slog.Error("Error shutting down source", "source", route.Source.Name(), "error", err)
			errs = append(errs, fmt.Errorf("source %s shutdown error: %w", route.Source.Name(), err))
		}

		for _, target := range route.Targets {
			slog.Debug("Shutting down target", "target", target.Name())
			if err := target.Shutdown(ctx); err != nil {
				slog.Error("Error shutting down target", "target", target.Name(), "error", err)
				errs = append(errs, fmt.Errorf("target %s shutdown error: %w", target.Name(), err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	slog.Info("Pipeline shutdown complete")
	return nil
}

func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
