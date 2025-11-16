package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Bot struct {
	name       string
	pipeline   *Pipeline
	interval   time.Duration
	runOnce    bool
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

	if err := b.pipeline.Initialize(ctx); err != nil {
		b.mu.Lock()
		b.running = false
		b.mu.Unlock()
		return fmt.Errorf("failed to initialize pipeline: %w", err)
	}

	if b.runOnce {
		return b.runOnceMode(ctx)
	}

	return b.runContinuousMode(ctx)
}

func (b *Bot) runOnceMode(ctx context.Context) error {
	defer b.markStopped()

	if err := b.pipeline.Run(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("pipeline execution failed: %w", err)
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

	if err := b.pipeline.Run(runCtx); err != nil && err != context.Canceled {
		return fmt.Errorf("pipeline run failed: %w", err)
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

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := b.pipeline.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("pipeline shutdown failed: %w", err)
	}

	if b.shutdownFn != nil {
		if err := b.shutdownFn(); err != nil {
			return fmt.Errorf("custom shutdown failed: %w", err)
		}
	}

	return nil
}

func (b *Bot) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

func (b *Bot) Name() string {
	return b.name
}

func (b *Bot) Errors() <-chan error {
	return b.errorCh
}

func (b *Bot) markStopped() {
	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
}

type BotManager struct {
	bots map[string]*Bot
	mu   sync.RWMutex
}

func NewBotManager() *BotManager {
	return &BotManager{
		bots: make(map[string]*Bot),
	}
}

func (m *BotManager) Register(bot *Bot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.bots[bot.Name()]; exists {
		return fmt.Errorf("bot %s already registered", bot.Name())
	}

	m.bots[bot.Name()] = bot
	return nil
}

func (m *BotManager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.bots, name)
}

func (m *BotManager) Get(name string) (*Bot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bot, exists := m.bots[name]
	return bot, exists
}

func (m *BotManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	bots := make([]*Bot, 0, len(m.bots))
	for _, bot := range m.bots {
		bots = append(bots, bot)
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(bots))

	for _, bot := range bots {
		wg.Add(1)
		go func(b *Bot) {
			defer wg.Done()
			if err := b.Start(ctx); err != nil && err != context.Canceled {
				errChan <- fmt.Errorf("bot %s failed: %w", b.Name(), err)
			}
		}(bot)
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

func (m *BotManager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	bots := make([]*Bot, 0, len(m.bots))
	for _, bot := range m.bots {
		bots = append(bots, bot)
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(bots))

	for _, bot := range bots {
		wg.Add(1)
		go func(b *Bot) {
			defer wg.Done()
			if err := b.Stop(ctx); err != nil {
				errChan <- fmt.Errorf("bot %s stop failed: %w", b.Name(), err)
			}
		}(bot)
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

func (m *BotManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.bots))
	for name := range m.bots {
		names = append(names, name)
	}
	return names
}
