package sqlite

import (
	"cartero/internal/storage"
	"context"
	"database/sql"
	"fmt"
	"time"
)

type feedStore struct {
	db *sql.DB
}

func newFeedStore(db *sql.DB) storage.FeedStore {
	return &feedStore{db: db}
}

func (s *feedStore) InsertEntry(ctx context.Context, id, title, link, description, content, author, source, imageURL string, publishedAt time.Time) error {
	query := `
		INSERT INTO feed_entries (id, title, link, description, content, author, source, image_url, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`

	publishedAtNull := sql.NullTime{Valid: !publishedAt.IsZero(), Time: publishedAt}

	_, err := s.db.ExecContext(ctx, query, id, title, link, description, content, author, source, imageURL, publishedAtNull)
	if err != nil {
		return fmt.Errorf("failed to insert feed entry: %w", err)
	}

	return nil
}

func (s *feedStore) ListRecentEntries(ctx context.Context, limit int) ([]storage.FeedEntry, error) {
	query := `
		SELECT id, title, link, description, content, author, source, image_url, published_at, created_at
		FROM feed_entries
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	entries := make([]storage.FeedEntry, 0, limit)
	for rows.Next() {
		var entry storage.FeedEntry
		var publishedAt sql.NullTime
		var imageURL sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Title,
			&entry.Link,
			&entry.Description,
			&entry.Content,
			&entry.Author,
			&entry.Source,
			&imageURL,
			&publishedAt,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}

		if publishedAt.Valid {
			entry.PublishedAt = publishedAt.Time
		}

		if imageURL.Valid {
			entry.ImageURL = imageURL.String
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

	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old entries: %w", err)
	}

	if rows, err := result.RowsAffected(); err == nil {
		_ = rows
	}

	return nil
}
