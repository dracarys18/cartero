package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

type feedStore struct {
	db *sql.DB
}

func newFeedStore(db *sql.DB) FeedStore {
	return &feedStore{db: db}
}

func (s *feedStore) InsertEntry(ctx context.Context, id, title, link, description, content, author, source string, publishedAt time.Time) error {
	query := `
		INSERT INTO feed_entries (id, title, link, description, content, author, source, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`

	publishedAtNull := sql.NullTime{Valid: !publishedAt.IsZero(), Time: publishedAt}

	_, err := s.db.ExecContext(ctx, query, id, title, link, description, content, author, source, publishedAtNull)
	if err != nil {
		return fmt.Errorf("failed to insert feed entry: %w", err)
	}

	return nil
}

func (s *feedStore) ListRecentEntries(ctx context.Context, limit int) ([]FeedEntry, error) {
	query := `
		SELECT id, title, link, description, content, author, source, published_at, created_at
		FROM feed_entries
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	entries := make([]FeedEntry, 0)
	for rows.Next() {
		var entry FeedEntry
		var publishedAt sql.NullTime

		err := rows.Scan(
			&entry.ID,
			&entry.Title,
			&entry.Link,
			&entry.Description,
			&entry.Content,
			&entry.Author,
			&entry.Source,
			&publishedAt,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}

		if publishedAt.Valid {
			entry.PublishedAt = publishedAt.Time
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

func (s *feedStore) DeleteOlderThan(ctx context.Context, age time.Duration) error {
	cutoff := time.Now().Add(-age)
	query := `DELETE FROM feed_entries WHERE created_at < ?`

	log.Printf("Deleting feed entries older than %v (cutoff: %s)", age, cutoff.Format(time.RFC3339))
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old entries: %w", err)
	}

	if rows, err := result.RowsAffected(); err == nil {
		log.Printf("Deleted %d old feed entries", rows)
	}

	return nil
}
