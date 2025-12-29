package core

import (
	"context"
	"time"
)

type Item struct {
	ID        string
	Content   interface{}
	Metadata  map[string]interface{}
	Source    string
	Timestamp time.Time
}

type ProcessedItem struct {
	Original *Item
	Data     interface{}
	Metadata map[string]interface{}
}

type PublishResult struct {
	Success   bool
	Target    string
	ItemID    string
	Timestamp time.Time
	Error     error
	Metadata  map[string]interface{}
}

type Source interface {
	Name() string
	Initialize(ctx context.Context) error
	Fetch(ctx context.Context) (<-chan *Item, <-chan error)
	Shutdown(ctx context.Context) error
}

type Filter interface {
	Name() string
	ShouldProcess(ctx context.Context, item *Item) (bool, error)
}

type Processor interface {
	Name() string
	Process(ctx context.Context, item *Item) (*ProcessedItem, error)
}

type ProcessorConfig struct {
	Name      string
	Type      string
	Enabled   bool
	DependsOn []string
}

type Target interface {
	Name() string
	Initialize(ctx context.Context) error
	Publish(ctx context.Context, item *ProcessedItem) (*PublishResult, error)
	Shutdown(ctx context.Context) error
}

type Storage interface {
	Store(ctx context.Context, item *Item) error
	Get(ctx context.Context, id string) (*Item, error)
	Exists(ctx context.Context, id string) (bool, error)
	MarkPublished(ctx context.Context, itemID, target string) error
	IsPublished(ctx context.Context, itemID, target string) (bool, error)
	Close() error
}
