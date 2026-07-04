package queue

type QueueConnection interface {
	Close() error
}
