package storage

import (
	"context"
	"net/url"
	"time"
)

type StorageInterface interface {
	Entries() EntryStore
	Close(ctx context.Context) error
}

type Item interface {
	GetID() string
	GetURL() *url.URL
	GetTitle() string
	GetSource() string
	GetTimestamp() time.Time
	GetEmbedding() [][]float32
	GetLink() *url.URL
	GetDescription() string
	GetFeedContent() string
	GetAuthor() string
	GetImageURL() string
	GetMatchedKeywords() string
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
	ExistsByHash(ctx context.Context, hashes []string) ([]string, error)
	MarkPublished(ctx context.Context, itemID, target string) error
	IsPublished(ctx context.Context, itemID, target string) (bool, error)
	InsertEntry(ctx context.Context, id, title string, link *url.URL, description, content, author, source, imageURL, matchedKeywords string, publishedAt time.Time) error
	ListRecentEntries(ctx context.Context, limit int) ([]FeedEntry, error)
	ListPublishedEntries(ctx context.Context, target string, limit int) ([]FeedEntry, error)
	ListEntriesPaginated(ctx context.Context, page, perPage int, startDate, endDate time.Time) (*PaginationResult, error)
	SearchSemantic(ctx context.Context, embedding []float32, limit int, maxDistance float64) ([]FeedEntry, error)
	SetEmbedding(ctx context.Context, id string, embedding []float32) error
	FindNearestEmbedding(ctx context.Context, embedding []float32, threshold float64, since time.Time) (bool, error)
}
