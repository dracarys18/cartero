package queue

import (
	"context"
	"time"
)

type Message struct {
	ID     string
	Stream string
	Data   map[string]any
}

type QueueConnection interface {
	CreateGroup(ctx context.Context, stream, group string) error
	Publish(ctx context.Context, stream string, maxLen int64, fields map[string]any) (string, error)
	Consume(ctx context.Context, stream, group, consumer string, count int64, block time.Duration) ([]Message, error)
	Ack(ctx context.Context, stream, group string, ids ...string) error
	Close() error
}
