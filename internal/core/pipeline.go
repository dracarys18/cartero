package core

import (
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
	routes     []SourceRoute
	processors []Processor
	storage    Storage
	mu         sync.RWMutex
	running    bool
}

func NewPipeline(storage Storage) *Pipeline {
	return &Pipeline{
		routes:     make([]SourceRoute, 0),
		processors: make([]Processor, 0),
		storage:    storage,
		running:    false,
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

func (p *Pipeline) Initialize(ctx context.Context) error {

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, route := range p.routes {
		if err := route.Source.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize source %s: %w", route.Source.Name(), err)
		}

		for _, target := range route.Targets {
			if err := target.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize target %s: %w", target.Name(), err)
			}
		}
	}

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

	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()

	var wg sync.WaitGroup

	for _, route := range p.routes {
		wg.Add(1)
		go func(r SourceRoute) {
			defer wg.Done()
			if err := p.processSource(ctx, r); err != nil {
				log.Printf("Error processing source %s: %v", r.Source.Name(), err)
			}
		}(route)
	}

	wg.Wait()
	return nil
}

func (p *Pipeline) processSource(ctx context.Context, route SourceRoute) error {
	itemChan, errChan := route.Source.Fetch(ctx)

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
				return nil
			}

			if p.storage != nil {
				exists, err := p.storage.Exists(ctx, item.ID)
				if err != nil {
					return err
				}
				if !exists {
					if err := p.storage.Store(ctx, item); err != nil {
						return err
					}
				}
			}

			if err := p.processItem(ctx, item, route); err != nil {
				return err
			}
		}
	}
}

func (p *Pipeline) processItem(ctx context.Context, item *Item, route SourceRoute) error {
	processed := &ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]interface{}),
		Skip:     false,
	}

	for _, processor := range p.processors {
		result, err := processor.Process(ctx, item)
		if err != nil {
			return fmt.Errorf("processor %s error: %w", processor.Name(), err)
		}
		if result.Skip {
			return nil
		}
		processed = result
	}

	if processed.Skip {
		return nil
	}

	return p.publishToTargets(ctx, processed, route)
}

func (p *Pipeline) publishToTargets(ctx context.Context, item *ProcessedItem, route SourceRoute) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(route.Targets))

	for i, target := range route.Targets {
		if p.storage != nil {
			published, err := p.storage.IsPublished(ctx, item.Original.ID, target.Name())
			if err != nil {
				return err
			}
			if published {
				continue
			}
		}

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
				errChan <- err
				return
			}

			if p.storage != nil {
				if err := p.storage.MarkPublished(ctx, item.Original.ID, t.Name()); err != nil {
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

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := target.Publish(ctx, item)

		if err == nil && result.Success {
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
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error

	for _, route := range p.routes {
		if err := route.Source.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("source %s shutdown error: %w", route.Source.Name(), err))
		}

		for _, target := range route.Targets {
			if err := target.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("target %s shutdown error: %w", target.Name(), err))
			}
		}
	}

	if p.storage != nil {
		if err := p.storage.Close(); err != nil {
			errs = append(errs, fmt.Errorf("storage close error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
