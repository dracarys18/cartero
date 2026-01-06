package filters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cartero/internal/config"
	"cartero/internal/types"
)

type RateLimitProcessor struct {
	name        string
	counter     int
	windowStart time.Time
	mu          sync.Mutex
}

func NewRateLimitProcessor(name string) *RateLimitProcessor {
	return &RateLimitProcessor{
		name:        name,
		counter:     0,
		windowStart: time.Now(),
	}
}

func (r *RateLimitProcessor) Name() string {
	return r.name
}

func (r *RateLimitProcessor) DependsOn() []string {
	return []string{}
}

func (r *RateLimitProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[r.name].Settings.RateLimitSettings
	window := config.ParseDuration(cfg.Window, 1*time.Minute)
	limit := cfg.Limit

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if now.Sub(r.windowStart) >= window {
		r.counter = 0
		r.windowStart = now
	}

	if r.counter >= limit {
		return fmt.Errorf("RateLimitProcessor %s: rate limit exceeded (%d/%d)", r.name, r.counter, limit)
	}

	r.counter++
	return nil
}

type TokenBucketProcessor struct {
	name       string
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucketProcessor(name string) *TokenBucketProcessor {
	return &TokenBucketProcessor{
		name:       name,
		lastRefill: time.Now(),
	}
}

func (t *TokenBucketProcessor) Name() string {
	return t.name
}

func (t *TokenBucketProcessor) DependsOn() []string {
	return []string{}
}

func (t *TokenBucketProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	cfg := st.GetConfig().Processors[t.name].Settings.TokenBucketSettings
	capacity := cfg.Capacity
	refillRate := config.ParseDuration(cfg.RefillRate, 1*time.Second)

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastRefill)

	tokensToAdd := int(elapsed / refillRate)
	if tokensToAdd > 0 {
		t.tokens = min(capacity, t.tokens+tokensToAdd)
		t.lastRefill = now
	}

	if t.tokens <= 0 {
		return fmt.Errorf("TokenBucketProcessor %s: no tokens available (tokens: %d)", t.name, t.tokens)
	}

	t.tokens--
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
