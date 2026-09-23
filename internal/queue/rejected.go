package queue

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RejectedSet struct {
	client *redis.Client
	key    string
}

func NewRejectedSet(client *redis.Client, prefix string) *RejectedSet {
	return &RejectedSet{client: client, key: prefix + ":rejected"}
}

func (r *RejectedSet) Add(ctx context.Context, id string) error {
	return r.client.SAdd(ctx, r.key, id).Err()
}

func (r *RejectedSet) Has(ctx context.Context, id string) (bool, error) {
	return r.client.SIsMember(ctx, r.key, id).Result()
}
