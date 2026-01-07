package filters

import (
	"context"
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
	logger := st.GetLogger()

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if now.Sub(r.windowStart) >= window {
		r.counter = 0
		r.windowStart = now
	}

	if r.counter >= limit {
		logger.Info("RateLimitProcessor rejected item", "processor", r.name, "item_id", item.ID, "current_count", r.counter, "limit", limit)
		return types.NewFilteredError(r.name, item.ID, "rate limit exceeded").
			WithDetail("current_count", r.counter).
			WithDetail("limit", limit)
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
	logger := st.GetLogger()

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
		logger.Info("TokenBucketProcessor rejected item", "processor", t.name, "item_id", item.ID, "available_tokens", t.tokens, "capacity", capacity)
		return types.NewFilteredError(t.name, item.ID, "no tokens available").
			WithDetail("available_tokens", t.tokens).
			WithDetail("capacity", capacity)
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
