package sqlite

import (
	"cartero/internal/storage"
	"cartero/internal/utils/hash"
	"context"
	"database/sql"
	"fmt"
	"time"
)

type itemStore struct {
	db *sql.DB
}

func newItemStore(db *sql.DB) storage.ItemStore {
	return &itemStore{db: db}
}

func (s *itemStore) Store(ctx context.Context, item storage.Item) error {
	h := hash.HashURL(item.GetURL())

	query := `
		INSERT INTO items (id, hash, source, timestamp)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, item.GetID(), h, item.GetSource(), item.GetTimestamp())
	if err != nil {
		return fmt.Errorf("failed to store item: %w", err)
	}

	return nil
}

func (s *itemStore) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

func (s *itemStore) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE hash = ?)`, hash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check hash existence: %w", err)
	}
	return exists, nil
}

func (s *itemStore) MarkPublished(ctx context.Context, itemID, target string) error {
	query := `
		INSERT INTO published (item_id, target)
		VALUES (?, ?)
		ON CONFLICT(item_id, target) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, itemID, target)
	if err != nil {
		return fmt.Errorf("failed to mark as published: %w", err)
	}

	return nil
}

func (s *itemStore) IsPublished(ctx context.Context, itemID, target string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM published WHERE item_id = ? AND target = ?)`, itemID, target).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check published status: %w", err)
	}
	return exists, nil
}

func (s *itemStore) DeleteOlderThan(ctx context.Context, age time.Duration) error {
	cutoff := time.Now().Add(-age)
	query := `DELETE FROM items WHERE timestamp < ?`

	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old items: %w", err)
	}

	if rows, err := result.RowsAffected(); err == nil {
		_ = rows
	}

	return nil
}
