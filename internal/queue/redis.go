package queue

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisConnection struct {
	client *redis.Client
}

func NewRedisConnection(addr, password string, db int) (*RedisConnection, error) {
	client := redis.NewClient(&redis.Options{
		Addr:          addr,
		Password:      password,
		DB:            db,
		UnstableResp3: true,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &RedisConnection{client: client}, nil
}

func (r *RedisConnection) Client() *redis.Client {
	return r.client
}

func (r *RedisConnection) Close() error {
	return r.client.Close()
}
