package filters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cartero/internal/core"
)

type RateLimitProcessor struct {
	name        string
	limit       int
	window      time.Duration
	counter     int
	windowStart time.Time
	mu          sync.Mutex
}

func NewRateLimitProcessor(name string, limit int, window time.Duration) *RateLimitProcessor {
	return &RateLimitProcessor{
		name:        name,
		limit:       limit,
		window:      window,
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

func (r *RateLimitProcessor) Process(ctx context.Context, item *core.Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if now.Sub(r.windowStart) >= r.window {
		r.counter = 0
		r.windowStart = now
	}

	if r.counter >= r.limit {
		return fmt.Errorf("RateLimitProcessor %s: rate limit exceeded (%d/%d)", r.name, r.counter, r.limit)
	}

	r.counter++
	return nil
}

type TokenBucketProcessor struct {
	name       string
	capacity   int
	tokens     int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucketProcessor(name string, capacity int, refillRate time.Duration) *TokenBucketProcessor {
	return &TokenBucketProcessor{
		name:       name,
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (t *TokenBucketProcessor) Name() string {
	return t.name
}

func (t *TokenBucketProcessor) DependsOn() []string {
	return []string{}
}

func (t *TokenBucketProcessor) Process(ctx context.Context, item *core.Item) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastRefill)

	tokensToAdd := int(elapsed / t.refillRate)
	if tokensToAdd > 0 {
		t.tokens = min(t.capacity, t.tokens+tokensToAdd)
		t.lastRefill = now
	}

	if t.tokens <= 0 {
		// No tokens available, filter it out
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
