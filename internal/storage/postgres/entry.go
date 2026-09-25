package postgres

import (
	"cartero/internal/storage"
	"cartero/internal/utils/hash"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
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

	if model, embedding := item.GetEmbedding(); len(embedding) > 0 {
		if err := s.SetEmbedding(ctx, item.GetID(), model, embedding); err != nil {
			return err
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

func (s *entryStore) ListEntriesPaginated(ctx context.Context, page, perPage int, filter storage.EntryFilter) (*storage.PaginationResult, error) {
	offset := (page - 1) * perPage

	query := `
		SELECT id, title, link, description, content, author, source, image_url, matched_keywords, hash, entry_timestamp, published_at, created_at,
		       COUNT(*) OVER() AS total_count
		FROM feed_entries
		WHERE title <> '' AND created_at >= $1 AND created_at < $2
		  AND (coalesce(cardinality($3::text[]), 0) = 0 OR matched_keywords = ANY($3::text[]))
		  AND (coalesce(cardinality($4::text[]), 0) = 0 OR source = ANY($4::text[]))
		ORDER BY created_at DESC
		LIMIT $5 OFFSET $6
	`

	rows, err := s.db.QueryContext(ctx, query, filter.Since, filter.Until, filter.Topics, filter.Sources, perPage, offset)
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

func (s *entryStore) Search(ctx context.Context, query string, limit int) ([]storage.FeedEntry, error) {
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

func (s *entryStore) SetEmbedding(ctx context.Context, id, model string, embedding []float32) error {
	query := `
		INSERT INTO entry_embeddings (id, model, embedding)
		VALUES ($1, $2, $3::vector)
		ON CONFLICT(id) DO UPDATE SET model = EXCLUDED.model, embedding = EXCLUDED.embedding
	`
	if _, err := s.db.ExecContext(ctx, query, id, model, vectorLiteral(embedding)); err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}
	return nil
}

func (s *entryStore) FindSimilarEntry(ctx context.Context, model string, embedding []float32, threshold float64, since time.Time) (bool, error) {
	query := `
		SELECT 1 - (e.embedding <=> $1::vector)
		FROM entry_embeddings e
		JOIN feed_entries fe ON fe.id = e.id
		WHERE e.model = $2 AND fe.created_at >= $3
		ORDER BY e.embedding <=> $1::vector
		LIMIT 1
	`
	var similarity float64
	err := s.db.QueryRowContext(ctx, query, vectorLiteral(embedding), model, since).Scan(&similarity)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to search embeddings: %w", err)
	}
	return similarity >= threshold, nil
}

func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

func (s *entryStore) ListFacets(ctx context.Context, filter storage.EntryFilter) ([]storage.Facet, []storage.Facet, error) {
	topics, err := s.facet(ctx, `
		SELECT matched_keywords, COUNT(*)
		FROM feed_entries
		WHERE title <> '' AND matched_keywords <> '' AND created_at >= $1 AND created_at < $2
		  AND (coalesce(cardinality($3::text[]), 0) = 0 OR source = ANY($3::text[]))
		GROUP BY 1
		ORDER BY 2 DESC, 1
	`, filter, filter.Sources)
	if err != nil {
		return nil, nil, err
	}

	sources, err := s.facet(ctx, `
		SELECT source, COUNT(*)
		FROM feed_entries
		WHERE title <> '' AND created_at >= $1 AND created_at < $2
		  AND (coalesce(cardinality($3::text[]), 0) = 0 OR matched_keywords = ANY($3::text[]))
		GROUP BY 1
		ORDER BY 2 DESC, 1
	`, filter, filter.Topics)
	if err != nil {
		return nil, nil, err
	}
	return topics, sources, nil
}

func (s *entryStore) facet(ctx context.Context, query string, filter storage.EntryFilter, other []string) ([]storage.Facet, error) {
	rows, err := s.db.QueryContext(ctx, query, filter.Since, filter.Until, other)
	if err != nil {
		return nil, fmt.Errorf("failed to query facets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var facets []storage.Facet
	for rows.Next() {
		var f storage.Facet
		if err := rows.Scan(&f.Value, &f.Count); err != nil {
			return nil, err
		}
		facets = append(facets, f)
	}
	return facets, rows.Err()
}
