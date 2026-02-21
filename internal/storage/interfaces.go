package storage

import (
	"context"
	"database/sql"
	"time"
)

type StorageInterface interface {
	GetConnection() *sql.DB
	Items() ItemStore
	Feed() FeedStore
	Close(ctx context.Context) error
}

type Item interface {
	GetID() string
	GetSource() string
	GetContent() interface{}
	GetTimestamp() time.Time
}

type ItemStore interface {
	Store(ctx context.Context, item Item) error
	Exists(ctx context.Context, id string) (bool, error)
	ExistsByHash(ctx context.Context, hash string) (bool, error)
	MarkPublished(ctx context.Context, itemID, target string) error
	IsPublished(ctx context.Context, itemID, target string) (bool, error)
	DeleteOlderThan(ctx context.Context, age time.Duration) error
}

type FeedEntry struct {
	ID          string
	Title       string
	Link        string
	Description string
	Content     string
	Author      string
	Source      string
	ImageURL    string
	PublishedAt time.Time
	CreatedAt   time.Time
}

type FeedStore interface {
	InsertEntry(ctx context.Context, id, title, link, description, content, author, source, imageURL string, publishedAt time.Time) error
	ListRecentEntries(ctx context.Context, limit int) ([]FeedEntry, error)
	DeleteOlderThan(ctx context.Context, age time.Duration) error
}
