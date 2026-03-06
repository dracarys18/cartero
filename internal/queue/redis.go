package queue

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConnection struct {
	client *redis.Client
}

func NewRedisConnection(addr, password string, db int) (*RedisConnection, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &RedisConnection{client: client}, nil
}

func (r *RedisConnection) CreateGroup(ctx context.Context, stream, group string) error {
	err := r.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (r *RedisConnection) Publish(ctx context.Context, stream string, maxLen int64, fields map[string]any) (string, error) {
	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: maxLen,
		Approx: true,
		Values: fields,
	}).Result()
}

func (r *RedisConnection) Consume(ctx context.Context, stream, group, consumer string, count int64, block time.Duration) ([]Message, error) {
	results, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		return nil, err
	}

	var messages []Message
	for _, s := range results {
		for _, msg := range s.Messages {
			data := make(map[string]any, len(msg.Values))
			for k, v := range msg.Values {
				data[k] = v
			}
			messages = append(messages, Message{
				ID:     msg.ID,
				Stream: s.Stream,
				Data:   data,
			})
		}
	}

	return messages, nil
}

func (r *RedisConnection) Ack(ctx context.Context, stream, group string, ids ...string) error {
	return r.client.XAck(ctx, stream, group, ids...).Err()
}

func (r *RedisConnection) Close() error {
	return r.client.Close()
}
