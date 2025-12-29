package core

import (
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
	routes            []SourceRoute
	filters           []Filter
	processors        []Processor
	processorConfigs  map[string]ProcessorConfig
	processorExecutor *ProcessorExecutor
	storage           Storage
	mu                sync.RWMutex
	running           bool
}

func NewPipeline(storage Storage) *Pipeline {
	return &Pipeline{
		routes:            make([]SourceRoute, 0),
		filters:           make([]Filter, 0),
		processors:        make([]Processor, 0),
		processorConfigs:  make(map[string]ProcessorConfig),
		processorExecutor: NewProcessorExecutor(),
		storage:           storage,
		running:           false,
	}
}

func (p *Pipeline) AddRoute(route SourceRoute) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.routes = append(p.routes, route)
	return p
}

func (p *Pipeline) AddFilters(filters []Filter) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.filters = filters
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
	p.processorExecutor.AddProcessor(config.Name, processor, config.DependsOn)
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
			log.Printf("Initializing target: %s for source: %s", target.Name(), route.Source.Name())
			if err := target.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize target %s: %w", target.Name(), err)
			}
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
			if p.storage != nil {
				exists, err := p.storage.Exists(ctx, item.ID)
				if err != nil {
					return err
				}
				if !exists {
					if err := p.storage.Store(ctx, item); err != nil {
						log.Printf("Source %s: error storing item %s: %v", route.Source.Name(), item.ID, err)
						return err
					}
					log.Printf("Source %s: stored new item %s", route.Source.Name(), item.ID)
				}
			}

			if err := p.processItem(ctx, item, route); err != nil {
				return err
			}
		}
	}
}

func (p *Pipeline) processItem(ctx context.Context, item *Item, route SourceRoute) error {
	log.Printf("Item %s: Running %d filters", item.ID, len(p.filters))
	for _, filter := range p.filters {
		shouldProcess, err := filter.ShouldProcess(ctx, item)
		if err != nil {
			return fmt.Errorf("filter %s error: %w", filter.Name(), err)
		}
		if !shouldProcess {
			log.Printf("Item %s: FILTERED OUT by filter '%s'", item.ID, filter.Name())
			return nil
		}
		log.Printf("Item %s: PASSED filter '%s'", item.ID, filter.Name())
	}

	log.Printf("Item %s: All filters passed! Running %d processors", item.ID, len(p.processors))
	processed := &ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]interface{}),
	}

	filterFunc := func(target Target) bool {
		published, err := p.storage.IsPublished(ctx, processed.Original.ID, target.Name())
		if err != nil {
			log.Printf("Error checking if item %s published to %s: %v", processed.Original.ID, target.Name(), err)
			return false
		}
		if published {
			log.Printf("Item %s already published to target %s, skipping", processed.Original.ID, target.Name())
			return false
		}
		return true
	}
	route.Targets = utils.FilterArray(route.Targets, filterFunc)

	if len(route.Targets) == 0 {
		log.Printf("item %s: no targets to publish to after filtering published targets", item.ID)
		return nil
	}

	log.Printf("Item %s: All filters passed! Running %d processors", item.ID, len(p.processors))

	result, err := p.processorExecutor.ExecuteProcessors(ctx, item)
	if err != nil {
		return fmt.Errorf("processor execution error: %w", err)
	}
	processed = result

	log.Printf("Item %s: All processors completed! Publishing to %d targets", item.ID, len(route.Targets))
	return p.publishToTargets(ctx, processed, route)
}

func (p *Pipeline) publishToTargets(ctx context.Context, item *ProcessedItem, route SourceRoute) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(route.Targets))

	for i, target := range route.Targets {
		log.Printf("Queuing item %s for target %s", item.Original.ID, target.Name())
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
				log.Printf("Failed to publish item %s to target %s after retries: %v", item.Original.ID, t.Name(), err)
				errChan <- err
				return
			}

			log.Printf("Successfully published item %s to target %s", item.Original.ID, t.Name())

			if p.storage != nil {
				if err := p.storage.MarkPublished(ctx, item.Original.ID, t.Name()); err != nil {
					log.Printf("Error marking item %s as published to %s: %v", item.Original.ID, t.Name(), err)
					errChan <- err
				}
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

func (p *Pipeline) publishWithRetry(ctx context.Context, target Target, item *ProcessedItem) error {
	maxRetries := 3
	var lastErr error

	if attempt := 0; attempt == 0 {
		log.Printf("Publishing item %s to target %s", item.Original.ID, target.Name())
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := target.Publish(ctx, item)

		if err == nil && result.Success {
			if attempt > 0 {
				log.Printf("Target %s: item %s published successfully on attempt %d", target.Name(), item.Original.ID, attempt+1)
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

	if p.storage != nil {
		log.Printf("Closing storage")
		if err := p.storage.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
			errs = append(errs, fmt.Errorf("storage close error: %w", err))
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
