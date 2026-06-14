package storage

import (
	"context"
	"database/sql"
	"time"
)

type StorageInterface interface {
	GetConnection() *sql.DB
	Entries() EntryStore
	Close(ctx context.Context) error
}

type Item interface {
	GetID() string
	GetURL() string
	GetTitle() string
	GetSource() string
	GetTimestamp() time.Time
}

type FeedEntry struct {
	ID              string
	Title           string
	Link            string
	Description     string
	Content         string
	Author          string
	Source          string
	ImageURL        string
	MatchedKeywords string
	Hash            string
	EntryTimestamp  time.Time
	PublishedAt     time.Time
	CreatedAt       time.Time
}

type PaginationResult struct {
	Entries     []FeedEntry
	Total       int
	Page        int
	PerPage     int
	TotalPages  int
	HasNext     bool
	HasPrevious bool
}

type EntryStore interface {
	Store(ctx context.Context, item Item) error
	Exists(ctx context.Context, id string) (bool, error)
	ExistsByHash(ctx context.Context, hash string) (bool, error)
	MarkPublished(ctx context.Context, itemID, target string) error
	IsPublished(ctx context.Context, itemID, target string) (bool, error)
	InsertEntry(ctx context.Context, id, title, link, description, content, author, source, imageURL, matchedKeywords string, publishedAt time.Time) error
	ListRecentEntries(ctx context.Context, limit int) ([]FeedEntry, error)
	ListEntriesPaginated(ctx context.Context, page, perPage int, startDate, endDate time.Time) (*PaginationResult, error)
	DeleteOlderThan(ctx context.Context, age time.Duration) error
}
