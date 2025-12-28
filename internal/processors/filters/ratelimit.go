package filters

import (
	"context"
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

func (r *RateLimitProcessor) ShouldProcess(ctx context.Context, item *core.Item) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if now.Sub(r.windowStart) >= r.window {
		r.counter = 0
		r.windowStart = now
	}

	if r.counter >= r.limit {
		// Rate limit exceeded, should not process
		return false, nil
	}

	r.counter++
	return true, nil
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

func (t *TokenBucketProcessor) ShouldProcess(ctx context.Context, item *core.Item) (bool, error) {
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
		// No tokens available, should not process
		return false, nil
	}

	t.tokens--
	return true, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
