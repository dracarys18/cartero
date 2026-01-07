package core

import (
	"cartero/internal/types"
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Pipeline struct {
	routes             []SourceRoute
	processors         []types.Processor
	processorConfigs   map[string]types.ProcessorConfig
	initializedTargets map[string]bool
	mu                 sync.RWMutex
	running            bool
}

func (p *Pipeline) GetRoutes() []SourceRoute {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.routes
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		routes:             make([]SourceRoute, 0),
		processors:         make([]types.Processor, 0),
		processorConfigs:   make(map[string]types.ProcessorConfig),
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

func (p *Pipeline) AddProcessor(processor types.Processor) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processors = append(p.processors, processor)
	return p
}

func (p *Pipeline) AddProcessorWithConfig(processor types.Processor, config types.ProcessorConfig) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processors = append(p.processors, processor)
	p.processorConfigs[config.Name] = config
	return p
}

func (p *Pipeline) Initialize(ctx context.Context, logger *slog.Logger) error {
	logger.Info("Initializing pipeline", "routes", len(p.routes))

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, route := range p.routes {
		logger.Info("Initializing source", "source", route.Source.Name())
		if err := route.Source.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize source %s: %w", route.Source.Name(), err)
		}

		for _, target := range route.Targets {
			if p.initializedTargets[target.Name()] {
				continue
			}

			logger.Info("Initializing target", "target", target.Name(), "source", route.Source.Name())
			if err := target.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize target %s: %w", target.Name(), err)
			}

			p.initializedTargets[target.Name()] = true
		}
	}

	logger.Info("Pipeline initialization complete")
	return nil
}

func (p *Pipeline) Run(ctx context.Context, state types.StateAccessor) error {
	logger := state.GetLogger()
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pipeline already running")
	}
	p.running = true
	p.mu.Unlock()

	logger.Info("Starting pipeline execution", "sources", len(p.routes))

	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		logger.Info("Pipeline execution completed")
	}()

	var wg sync.WaitGroup

	for i := range p.routes {
		wg.Add(1)
		logger.Debug("Launching goroutine for source", "source", p.routes[i].Source.Name())
		go func(r *SourceRoute) {
			defer wg.Done()
			if err := r.Process(ctx, state); err != nil {
				logger.Error("Error processing source", "source", r.Source.Name(), "error", err)
			} else {
				logger.Info("Source completed successfully", "source", r.Source.Name())
			}
		}(&p.routes[i])
	}

	wg.Wait()
	return nil
}

func (p *Pipeline) Shutdown(ctx context.Context, logger *slog.Logger) error {
	logger.Info("Shutting down pipeline")
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error

	for _, route := range p.routes {
		logger.Debug("Shutting down source", "source", route.Source.Name())
		if err := route.Source.Shutdown(ctx); err != nil {
			logger.Error("Error shutting down source", "source", route.Source.Name(), "error", err)
			errs = append(errs, fmt.Errorf("source %s shutdown error: %w", route.Source.Name(), err))
		}

		for _, target := range route.Targets {
			logger.Debug("Shutting down target", "target", target.Name())
			if err := target.Shutdown(ctx); err != nil {
				logger.Error("Error shutting down target", "target", target.Name(), "error", err)
				errs = append(errs, fmt.Errorf("target %s shutdown error: %w", target.Name(), err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	logger.Info("Pipeline shutdown complete")
	return nil
}

func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
