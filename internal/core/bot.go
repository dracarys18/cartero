package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cartero/internal/types"
)

type Bot struct {
	name       string
	pipeline   *Pipeline
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

	b.pipeline.StartConsumers(ctx, b.state)

	if b.runOnce {
		return b.runOnceMode(ctx)
	}

	return b.runContinuousMode(ctx)
}

func (b *Bot) runOnceMode(ctx context.Context) error {
	defer b.markStopped()

	if err := ProcessCore(ctx, b.state); err != nil && err != context.Canceled {
		return fmt.Errorf("process core failed: %w", err)
	}

	return nil
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

	if err := ProcessCore(runCtx, b.state); err != nil && err != context.Canceled {
		return fmt.Errorf("process core failed: %w", err)
	}

	return nil
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

