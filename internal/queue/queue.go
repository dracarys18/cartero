package queue

import (
	"context"
	"fmt"
	"os"
	"time"

	"cartero/internal/types"
)

const (
	groupName     = "cartero:workers"
	defaultPrefix = "cartero"
	consumeCount  = 1
	consumeBlock  = 5 * time.Second
)

type Queue struct {
	conn     QueueConnection
	prefix   string
	maxLen   int64
	consumer string
}

func New(conn QueueConnection, maxLen int64, consumer string) *Queue {
	prefix := os.Getenv("CARTERO_QUEUE_PREFIX")
	if prefix == "" {
		prefix = defaultPrefix
	}

	return &Queue{
		conn:     conn,
		prefix:   prefix,
		maxLen:   maxLen,
		consumer: consumer,
	}
}

func (q *Queue) SourceStream() string {
	return fmt.Sprintf("%s:source", q.prefix)
}

func (q *Queue) ProcessedStream() string {
	return fmt.Sprintf("%s:processed", q.prefix)
}

func (q *Queue) CreateGroup(ctx context.Context, stream string) error {
	return q.conn.CreateGroup(ctx, stream, groupName)
}

func (q *Queue) Publish(ctx context.Context, stream string, env types.Envelope) error {
	fields, err := marshalEnvelope(env)
	if err != nil {
		return err
	}
	_, err = q.conn.Publish(ctx, stream, q.maxLen, fields)
	return err
}

func (q *Queue) Consume(ctx context.Context, stream string) ([]types.Envelope, []string, error) {
	messages, err := q.conn.Consume(ctx, stream, groupName, q.consumer, consumeCount, consumeBlock)
	if err != nil {
		return nil, nil, err
	}

	envelopes := make([]types.Envelope, 0, len(messages))
	ids := make([]string, 0, len(messages))

	for _, msg := range messages {
		env, err := unmarshalEnvelope(msg.Data)
		if err != nil {
			return nil, nil, fmt.Errorf("message %s: %w", msg.ID, err)
		}
		envelopes = append(envelopes, env)
		ids = append(ids, msg.ID)
	}

	return envelopes, ids, nil
}

func (q *Queue) Ack(ctx context.Context, stream string, ids ...string) error {
	return q.conn.Ack(ctx, stream, groupName, ids...)
}

func (q *Queue) Close() error {
	return q.conn.Close()
}
