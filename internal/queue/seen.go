package queue

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type SeenStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

func NewSeenStore(client *redis.Client, prefix string, ttl time.Duration) *SeenStore {
	return &SeenStore{client: client, prefix: prefix, ttl: ttl}
}

func (s *SeenStore) Seen(ctx context.Context, hash string) (bool, error) {
	n, err := s.client.Exists(ctx, s.key(hash)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *SeenStore) Mark(ctx context.Context, hash string) error {
	return s.client.Set(ctx, s.key(hash), 1, s.ttl).Err()
}

func (s *SeenStore) key(hash string) string {
	return s.prefix + ":seen:" + hash
}
