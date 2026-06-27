package postgres

import (
	"cartero/internal/storage"
	"cartero/internal/utils/hash"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pgvector/pgvector-go"
)

type entryStore struct {
	db *sql.DB
}

func newEntryStore(db *sql.DB) storage.EntryStore {
	return &entryStore{db: db}
}

func (s *entryStore) Store(ctx context.Context, item storage.Item) error {
	h := hash.HashURL(item.GetURL())

	query := `
		INSERT INTO feed_entries (id, hash, source, entry_timestamp, title)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, item.GetID(), h, item.GetSource(), item.GetTimestamp(), item.GetTitle())
	if err != nil {
		return fmt.Errorf("failed to store entry: %w", err)
	}

	embeddings := item.GetEmbedding()
	if len(embeddings) > 0 {
		if err := s.SetEmbedding(ctx, item.GetID(), embeddings[0]); err != nil {
			return fmt.Errorf("failed to store embedding: %w", err)
		}
	}

	return nil
}

func (s *entryStore) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM feed_entries WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

func (s *entryStore) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM feed_entries WHERE hash = $1)`, hash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check hash existence: %w", err)
	}
	return exists, nil
}

func (s *entryStore) MarkPublished(ctx context.Context, itemID, target string) error {
	query := `
		INSERT INTO published (item_id, target)
		VALUES ($1, $2)
		ON CONFLICT(item_id, target) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, itemID, target)
	if err != nil {
		return fmt.Errorf("failed to mark as published: %w", err)
	}

	return nil
}

func (s *entryStore) IsPublished(ctx context.Context, itemID, target string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM published WHERE item_id = $1 AND target = $2)`, itemID, target).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check published status: %w", err)
	}
	return exists, nil
}

func (s *entryStore) InsertEntry(ctx context.Context, id, title, link, description, content, author, source, imageURL, matchedKeywords string, publishedAt time.Time) error {
	query := `
		INSERT INTO feed_entries (id, title, link, description, content, author, source, image_url, matched_keywords, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT(id) DO UPDATE SET
			title = EXCLUDED.title,
			link = EXCLUDED.link,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			author = EXCLUDED.author,
			image_url = EXCLUDED.image_url,
			matched_keywords = EXCLUDED.matched_keywords,
			published_at = EXCLUDED.published_at
	`

	publishedAtNull := sql.NullTime{Valid: !publishedAt.IsZero(), Time: publishedAt}

	_, err := s.db.ExecContext(ctx, query, id, title, link, description, content, author, source, imageURL, matchedKeywords, publishedAtNull)
	if err != nil {
		return fmt.Errorf("failed to insert feed entry: %w", err)
	}

	return nil
}

func (s *entryStore) ListRecentEntries(ctx context.Context, limit int) ([]storage.FeedEntry, error) {
	query := `
		SELECT id, title, link, description, content, author, source, image_url, matched_keywords, hash, entry_timestamp, published_at, created_at
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

func (s *entryStore) ListEntriesPaginated(ctx context.Context, page, perPage int, startDate, endDate time.Time) (*storage.PaginationResult, error) {
	offset := (page - 1) * perPage

	query := `
		SELECT id, title, link, description, content, author, source, image_url, matched_keywords, hash, entry_timestamp, published_at, created_at,
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
		var hash sql.NullString
		var entryTimestamp sql.NullTime

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
			&hash,
			&entryTimestamp,
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

		if hash.Valid {
			entry.Hash = hash.String
		}

		if entryTimestamp.Valid {
			entry.EntryTimestamp = entryTimestamp.Time
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

func (s *entryStore) scanEntries(rows *sql.Rows, capacity int) ([]storage.FeedEntry, error) {
	entries := make([]storage.FeedEntry, 0, capacity)
	for rows.Next() {
		var entry storage.FeedEntry
		var publishedAt sql.NullTime
		var imageURL sql.NullString
		var matchedKeywords sql.NullString
		var hash sql.NullString
		var entryTimestamp sql.NullTime

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
			&hash,
			&entryTimestamp,
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

		if hash.Valid {
			entry.Hash = hash.String
		}

		if entryTimestamp.Valid {
			entry.EntryTimestamp = entryTimestamp.Time
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

func (s *entryStore) DeleteOlderThan(ctx context.Context, age time.Duration) error {
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

func (s *entryStore) SetEmbedding(ctx context.Context, id string, embedding []float32) error {
	query := `
		INSERT INTO item_embeddings (id, embedding)
		VALUES ($1, $2)
		ON CONFLICT(id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, id, pgvector.NewHalfVector(embedding))
	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	return nil
}

func (s *entryStore) FindNearestEmbedding(ctx context.Context, embedding []float32, threshold float64, since time.Time) (bool, error) {
	vec := pgvector.NewHalfVector(embedding)

	query := `
		SELECT 1 - (embedding <=> $2) AS similarity
		FROM item_embeddings
		WHERE created_at >= $1
		ORDER BY embedding <=> $2
		LIMIT 1
	`

	var similarity float64
	err := s.db.QueryRowContext(ctx, query, since, vec).Scan(&similarity)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to search embeddings: %w", err)
	}

	return similarity >= threshold, nil
}
