package queue

import (
	"os"
)

const (
	defaultPrefix = "cartero"
)

type Queue struct {
	conn   QueueConnection
	prefix string
}

func New(conn QueueConnection) *Queue {
	prefix := os.Getenv("CARTERO_QUEUE_PREFIX")
	if prefix == "" {
		prefix = defaultPrefix
	}

	return &Queue{
		conn:   conn,
		prefix: prefix,
	}
}

func (q *Queue) Prefix() string {
	return q.prefix
}

func (q *Queue) Close() error {
	return q.conn.Close()
}
