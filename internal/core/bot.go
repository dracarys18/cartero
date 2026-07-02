package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cartero/internal/processors/filters"
	"cartero/internal/types"
)

type Bot struct {
	name       string
	pipeline   *Pipeline
	filters    *filters.Chain
	targets    Targets
	interval   time.Duration
	runOnce    bool
	state      types.StateAccessor
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
	errorCh    chan error
	shutdownFn func() error
}

type BotConfig struct {
	Name       string
	Pipeline   *Pipeline
	Filters    *filters.Chain
	Targets    Targets
	Interval   time.Duration
	RunOnce    bool
	State      types.StateAccessor
	ShutdownFn func() error
}

func NewBot(config BotConfig) *Bot {
	if config.Interval == 0 {
		config.Interval = 5 * time.Minute
	}

	return &Bot{
		name:       config.Name,
		pipeline:   config.Pipeline,
		filters:    config.Filters,
		targets:    config.Targets,
		interval:   config.Interval,
		runOnce:    config.RunOnce,
		state:      config.State,
		running:    false,
		stopCh:     make(chan struct{}),
		errorCh:    make(chan error, 10),
		shutdownFn: config.ShutdownFn,
	}
}

func (b *Bot) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return fmt.Errorf("bot already running")
	}
	b.running = true
	b.mu.Unlock()

	if b.runOnce {
		return b.runOnceMode(ctx)
	}

	return b.runContinuousMode(ctx)
}

func (b *Bot) runCycle(ctx context.Context) error {
	logger := b.state.GetLogger()

	items, err := b.pipeline.Gather(ctx, b.state)
	if err != nil {
		return fmt.Errorf("gather: %w", err)
	}
	logger.Info("gathered items", "count", len(items))

	items, err = b.filters.Filter(ctx, b.state, items)
	if err != nil {
		return fmt.Errorf("filter: %w", err)
	}

	return b.targets.Publish(ctx, b.state, items, logger)
}

func (b *Bot) runOnceMode(ctx context.Context) error {
	defer b.markStopped()
	return b.runCycle(ctx)
}

func (b *Bot) runContinuousMode(ctx context.Context) error {
	defer b.markStopped()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	if err := b.executeRun(ctx); err != nil {
		b.errorCh <- err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b.stopCh:
			return nil
		case <-ticker.C:
			if err := b.executeRun(ctx); err != nil {
				select {
				case b.errorCh <- err:
				default:
				}
			}
		}
	}
}

func (b *Bot) executeRun(ctx context.Context) error {
	runCtx, cancel := context.WithTimeout(ctx, b.interval-10*time.Second)
	defer cancel()

	return b.runCycle(runCtx)
}

func (b *Bot) Stop(ctx context.Context) error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	close(b.stopCh)

	if b.shutdownFn != nil {
		if err := b.shutdownFn(); err != nil {
			return fmt.Errorf("custom shutdown failed: %w", err)
		}
	}

	return nil
}

func (b *Bot) Name() string {
	return b.name
}

func (b *Bot) markStopped() {
	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
}

