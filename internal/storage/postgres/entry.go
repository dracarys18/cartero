package postgres

import (
	"cartero/internal/storage"
	"cartero/internal/utils/hash"
	"context"
	"database/sql"
	"fmt"
	"net/url"
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
	publishedAt := sql.NullTime{Valid: !item.GetTimestamp().IsZero(), Time: item.GetTimestamp()}

	query := `
		INSERT INTO feed_entries (id, hash, source, entry_timestamp, title, link, description, content, author, image_url, matched_keywords, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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

	_, err := s.db.ExecContext(ctx, query,
		item.GetID(), h, item.GetSource(), item.GetTimestamp(), item.GetTitle(),
		item.GetLink().String(), item.GetDescription(), item.GetFeedContent(), item.GetAuthor(),
		item.GetImageURL(), item.GetMatchedKeywords(), publishedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to store entry: %w", err)
	}

	embeddings := item.GetEmbedding()
	if len(embeddings) > 0 {
		if err := s.SetEmbedding(ctx, item.GetID(), embeddings[0]); err != nil {
			return fmt.Errorf("failed to store embedding: %w", err)
		}
		if err := s.setChunks(ctx, item.GetID(), embeddings); err != nil {
			return fmt.Errorf("failed to store chunks: %w", err)
		}
	}

	return nil
}

func (s *entryStore) setChunks(ctx context.Context, id string, embeddings [][]float32) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM item_chunks WHERE item_id = $1`, id); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO item_chunks (item_id, chunk_index, embedding) VALUES ($1, $2, $3)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for idx, vec := range embeddings {
		if len(vec) == 0 {
			continue
		}
		if _, err := stmt.ExecContext(ctx, id, idx, pgvector.NewHalfVector(vec)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *entryStore) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM feed_entries WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

func (s *entryStore) ExistsByHash(ctx context.Context, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT hash FROM feed_entries WHERE hash = ANY($1)`, hashes)
	if err != nil {
		return nil, fmt.Errorf("failed to check hash existence: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var existing []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		existing = append(existing, h)
	}
	return existing, rows.Err()
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

func (s *entryStore) InsertEntry(ctx context.Context, id, title string, link *url.URL, description, content, author, source, imageURL, matchedKeywords string, publishedAt time.Time) error {
	query := `
		INSERT INTO feed_entries (id, title, link, description, content, author, source, image_url, matched_keywords, hash, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT(id) DO UPDATE SET
			title = EXCLUDED.title,
			link = EXCLUDED.link,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			author = EXCLUDED.author,
			image_url = EXCLUDED.image_url,
			matched_keywords = EXCLUDED.matched_keywords,
			hash = EXCLUDED.hash,
			published_at = EXCLUDED.published_at
	`

	publishedAtNull := sql.NullTime{Valid: !publishedAt.IsZero(), Time: publishedAt}

	_, err := s.db.ExecContext(ctx, query, id, title, link.String(), description, content, author, source, imageURL, matchedKeywords, hash.HashURL(link), publishedAtNull)
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

func (s *entryStore) ListPublishedEntries(ctx context.Context, target string, limit int) ([]storage.FeedEntry, error) {
	query := `
		SELECT fe.id, fe.title, fe.link, fe.description, fe.content, fe.author, fe.source, fe.image_url, fe.matched_keywords, fe.hash, fe.entry_timestamp, fe.published_at, fe.created_at
		FROM feed_entries fe
		JOIN published p ON p.item_id = fe.id AND p.target = $1
		ORDER BY p.published_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, target, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query published entries: %w", err)
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
		var link, description, content, author sql.NullString
		var publishedAt sql.NullTime
		var imageURL sql.NullString
		var matchedKeywords sql.NullString
		var hash sql.NullString
		var entryTimestamp sql.NullTime

		err := rows.Scan(
			&entry.ID,
			&entry.Title,
			&link,
			&description,
			&content,
			&author,
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

		if link.Valid {
			entry.Link = link.String
		}

		if description.Valid {
			entry.Description = description.String
		}

		if content.Valid {
			entry.Content = content.String
		}

		if author.Valid {
			entry.Author = author.String
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
		var link, description, content, author sql.NullString
		var publishedAt sql.NullTime
		var imageURL sql.NullString
		var matchedKeywords sql.NullString
		var hash sql.NullString
		var entryTimestamp sql.NullTime

		err := rows.Scan(
			&entry.ID,
			&entry.Title,
			&link,
			&description,
			&content,
			&author,
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

		if link.Valid {
			entry.Link = link.String
		}

		if description.Valid {
			entry.Description = description.String
		}

		if content.Valid {
			entry.Content = content.String
		}

		if author.Valid {
			entry.Author = author.String
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return entries, nil
}

func (s *entryStore) Search(ctx context.Context, query string, embedding []float32, limit int, maxDistance float64) ([]storage.FeedEntry, error) {
	entries, err := s.searchLexical(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	if len(entries) > 0 || len(embedding) == 0 {
		return entries, nil
	}
	return s.searchSemantic(ctx, embedding, limit, maxDistance)
}

func (s *entryStore) searchLexical(ctx context.Context, query string, limit int) ([]storage.FeedEntry, error) {
	q := `
		SELECT fe.id, fe.title, fe.link, fe.description, fe.content, fe.author, fe.source, fe.image_url, fe.matched_keywords, fe.hash, fe.entry_timestamp, fe.published_at, fe.created_at
		FROM feed_entries fe
		WHERE to_tsvector('english', coalesce(fe.title, '') || ' ' || coalesce(fe.description, '')) @@ websearch_to_tsquery('english', $1)
		ORDER BY ts_rank(to_tsvector('english', coalesce(fe.title, '') || ' ' || coalesce(fe.description, '')), websearch_to_tsquery('english', $1)) DESC, fe.published_at DESC NULLS LAST
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, q, query, limit)
	if err != nil {
		return nil, fmt.Errorf("lexical search failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return s.scanEntries(rows, limit)
}

func (s *entryStore) searchSemantic(ctx context.Context, embedding []float32, limit int, maxDistance float64) ([]storage.FeedEntry, error) {
	vec := pgvector.NewHalfVector(embedding)

	q := `
		SELECT fe.id, fe.title, fe.link, fe.description, fe.content, fe.author, fe.source, fe.image_url, fe.matched_keywords, fe.hash, fe.entry_timestamp, fe.published_at, fe.created_at
		FROM item_embeddings e
		JOIN feed_entries fe ON fe.id = e.id
		WHERE (e.embedding <=> $1) < $3
		ORDER BY e.embedding <=> $1
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, q, vec, limit, maxDistance)
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return s.scanEntries(rows, limit)
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
