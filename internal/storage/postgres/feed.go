package postgres

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

func (s *feedStore) InsertEntry(ctx context.Context, id, title, link, description, content, author, source, imageURL, matchedKeywords string, publishedAt time.Time) error {
	query := `
		INSERT INTO feed_entries (id, title, link, description, content, author, source, image_url, matched_keywords, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT(id) DO NOTHING
	`

	publishedAtNull := sql.NullTime{Valid: !publishedAt.IsZero(), Time: publishedAt}

	_, err := s.db.ExecContext(ctx, query, id, title, link, description, content, author, source, imageURL, matchedKeywords, publishedAtNull)
	if err != nil {
		return fmt.Errorf("failed to insert feed entry: %w", err)
	}

	return nil
}

func (s *feedStore) ListRecentEntries(ctx context.Context, limit int) ([]storage.FeedEntry, error) {
	query := `
		SELECT id, title, link, description, content, author, source, image_url, matched_keywords, published_at, created_at
		FROM feed_entries
		ORDER BY published_at DESC, created_at DESC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return s.scanEntries(rows, limit)
}

func (s *feedStore) ListEntriesPaginated(ctx context.Context, page, perPage int, startDate, endDate time.Time) (*storage.PaginationResult, error) {
	offset := (page - 1) * perPage

	query := `
		SELECT id, title, link, description, content, author, source, image_url, matched_keywords, published_at, created_at,
		       COUNT(*) OVER() AS total_count
		FROM feed_entries
		WHERE created_at >= $1 AND created_at < $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.db.QueryContext(ctx, query, startDate, endDate, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []storage.FeedEntry
	var total int

	for rows.Next() {
		var entry storage.FeedEntry
		var publishedAt sql.NullTime
		var imageURL sql.NullString
		var matchedKeywords sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Title,
			&entry.Link,
			&entry.Description,
			&entry.Content,
			&entry.Author,
			&entry.Source,
			&imageURL,
			&matchedKeywords,
			&publishedAt,
			&entry.CreatedAt,
			&total,
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

		if matchedKeywords.Valid {
			entry.MatchedKeywords = matchedKeywords.String
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	return &storage.PaginationResult{
		Entries:     entries,
		Total:       total,
		Page:        page,
		PerPage:     perPage,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}, nil
}

func (s *feedStore) scanEntries(rows *sql.Rows, capacity int) ([]storage.FeedEntry, error) {
	entries := make([]storage.FeedEntry, 0, capacity)
	for rows.Next() {
		var entry storage.FeedEntry
		var publishedAt sql.NullTime
		var imageURL sql.NullString
		var matchedKeywords sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Title,
			&entry.Link,
			&entry.Description,
			&entry.Content,
			&entry.Author,
			&entry.Source,
			&imageURL,
			&matchedKeywords,
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

		if matchedKeywords.Valid {
			entry.MatchedKeywords = matchedKeywords.String
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

func (s *feedStore) DeleteOlderThan(ctx context.Context, age time.Duration) error {
	cutoff := time.Now().Add(-age)
	query := `DELETE FROM feed_entries WHERE created_at < $1`

	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old entries: %w", err)
	}

	if rows, err := result.RowsAffected(); err == nil {
		_ = rows
	}

	return nil
}
