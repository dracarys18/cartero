package queue

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RejectedSet struct {
	client *redis.Client
	key    string
	ttl    time.Duration
}

func NewRejectedSet(client *redis.Client, prefix string, ttl time.Duration) *RejectedSet {
	return &RejectedSet{client: client, key: prefix + ":rejected", ttl: ttl}
}

func (r *RejectedSet) Add(ctx context.Context, id string) error {
	if err := r.client.SAdd(ctx, r.key, id).Err(); err != nil {
		return err
	}
	if r.ttl > 0 {
		return r.client.Expire(ctx, r.key, r.ttl).Err()
	}
	return nil
}

func (r *RejectedSet) Has(ctx context.Context, id string) (bool, error) {
	return r.client.SIsMember(ctx, r.key, id).Result()
}
