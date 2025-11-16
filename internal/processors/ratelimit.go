package processors

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

func (r *RateLimitProcessor) Process(ctx context.Context, item *core.Item) (*core.ProcessedItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if now.Sub(r.windowStart) >= r.window {
		r.counter = 0
		r.windowStart = now
	}

	processed := &core.ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]interface{}),
		Skip:     false,
	}

	if r.counter >= r.limit {
		processed.Skip = true
		processed.Metadata["rate_limited"] = true
		processed.Metadata["limit"] = r.limit
		processed.Metadata["window"] = r.window.String()
		return processed, nil
	}

	r.counter++
	processed.Metadata["rate_limit_count"] = r.counter
	processed.Metadata["rate_limit_remaining"] = r.limit - r.counter

	return processed, nil
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

func (t *TokenBucketProcessor) Process(ctx context.Context, item *core.Item) (*core.ProcessedItem, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastRefill)

	tokensToAdd := int(elapsed / t.refillRate)
	if tokensToAdd > 0 {
		t.tokens = min(t.capacity, t.tokens+tokensToAdd)
		t.lastRefill = now
	}

	processed := &core.ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]interface{}),
		Skip:     false,
	}

	if t.tokens <= 0 {
		processed.Skip = true
		processed.Metadata["rate_limited"] = true
		processed.Metadata["bucket_empty"] = true
		return processed, nil
	}

	t.tokens--
	processed.Metadata["tokens_remaining"] = t.tokens
	processed.Metadata["bucket_capacity"] = t.capacity

	return processed, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
