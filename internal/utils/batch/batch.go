package batch

import (
	"context"
	"sync"
)

func Run[T any](ctx context.Context, items []T, concurrency int, fn func(context.Context, T)) {
	if concurrency <= 0 {
		concurrency = 1
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for _, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(it T) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(ctx, it)
		}(item)
	}
	wg.Wait()
}
