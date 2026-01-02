package core

import (
	"cartero/internal/storage"
	"cartero/internal/utils"
	"context"
	"fmt"
	"log"
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
	log.Printf("Initializing pipeline with %d routes", len(p.routes))

	if err := p.processorExecutor.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize processor executor: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, route := range p.routes {
		log.Printf("Initializing source: %s", route.Source.Name())
		if err := route.Source.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize source %s: %w", route.Source.Name(), err)
		}

		for _, target := range route.Targets {
			if p.initializedTargets[target.Name()] {
				continue
			}

			log.Printf("Initializing target: %s for source: %s", target.Name(), route.Source.Name())
			if err := target.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize target %s: %w", target.Name(), err)
			}

			p.initializedTargets[target.Name()] = true
		}
	}

	log.Printf("Pipeline initialization complete")
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

	log.Printf("Starting pipeline execution with %d sources", len(p.routes))

	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		log.Printf("Pipeline execution completed")
	}()

	var wg sync.WaitGroup

	for _, route := range p.routes {
		wg.Add(1)
		log.Printf("Launching goroutine for source: %s", route.Source.Name())
		go func(r SourceRoute) {
			defer wg.Done()
			if err := p.processSource(ctx, r); err != nil {
				log.Printf("Error processing source %s: %v", r.Source.Name(), err)
			} else {
				log.Printf("Source %s completed successfully", r.Source.Name())
			}
		}(route)
	}

	wg.Wait()
	return nil
}

func (p *Pipeline) processSource(ctx context.Context, route SourceRoute) error {
	log.Printf("Source %s: starting to fetch items", route.Source.Name())
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
				log.Printf("Source %s: finished processing %d items", route.Source.Name(), itemCount)
				return nil
			}

			itemCount++
			exists, err := p.itemStore.Exists(ctx, item.ID)
			if err != nil {
				return err
			}
			if !exists {
				if err := p.itemStore.Store(ctx, item); err != nil {
					log.Printf("Source %s: error storing item %s: %v", route.Source.Name(), item.ID, err)
					return err
				}
				log.Printf("Source %s: stored new item %s", route.Source.Name(), item.ID)
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
			log.Printf("Error checking if item %s published to %s: %v", item.ID, target.Name(), err)
			return false
		}
		if published {
			log.Printf("Item %s already published to target %s, skipping", item.ID, target.Name())
			return false
		}
		return true
	}
	route.Targets = utils.FilterArray(route.Targets, filterFunc)

	if len(route.Targets) == 0 {
		log.Printf("item %s: no targets to publish to after filtering published targets", item.ID)
		return nil
	}

	log.Printf("Item %s: Running %d processors", item.ID, len(p.processors))

	err := p.processorExecutor.ExecuteProcessors(ctx, item)
	if err != nil {
		log.Printf("Item %s: filtered out during processing: %v", item.ID, err)
		return nil
	}

	log.Printf("Item %s: All processors completed! Publishing to %d targets", item.ID, len(route.Targets))
	return p.publishToTargets(ctx, item, route)
}

func (p *Pipeline) publishToTargets(ctx context.Context, item *Item, route SourceRoute) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(route.Targets))

	for i, target := range route.Targets {
		log.Printf("Queuing item %s for target %s", item.ID, target.Name())
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
				log.Printf("Failed to publish item %s to target %s after retries: %v", item.ID, t.Name(), err)
				errChan <- err
				return
			}

			log.Printf("Successfully published item %s to target %s", item.ID, t.Name())

			if err := p.itemStore.MarkPublished(ctx, item.ID, t.Name()); err != nil {
				log.Printf("Error marking item %s as published to %s: %v", item.ID, t.Name(), err)
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
		log.Printf("Publishing item %s to target %s", item.ID, target.Name())
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := target.Publish(ctx, item)

		if err == nil && result.Success {
			if attempt > 0 {
				log.Printf("Target %s: item %s published successfully on attempt %d", target.Name(), item.ID, attempt+1)
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

			log.Printf("Target %s: attempt %d/%d failed, retrying in %v: %v",
				target.Name(), attempt+1, maxRetries+1, waitDuration, lastErr)

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
	log.Printf("Shutting down pipeline")
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error

	for _, route := range p.routes {
		log.Printf("Shutting down source: %s", route.Source.Name())
		if err := route.Source.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down source %s: %v", route.Source.Name(), err)
			errs = append(errs, fmt.Errorf("source %s shutdown error: %w", route.Source.Name(), err))
		}

		for _, target := range route.Targets {
			log.Printf("Shutting down target: %s", target.Name())
			if err := target.Shutdown(ctx); err != nil {
				log.Printf("Error shutting down target %s: %v", target.Name(), err)
				errs = append(errs, fmt.Errorf("target %s shutdown error: %w", target.Name(), err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	log.Printf("Pipeline shutdown complete")
	return nil
}

func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
