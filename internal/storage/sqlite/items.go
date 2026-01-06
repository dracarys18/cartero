package sqlite

import (
	"cartero/internal/storage"
	"cartero/internal/utils/hash"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type itemStore struct {
	db *sql.DB
}

func newItemStore(db *sql.DB) storage.ItemStore {
	return &itemStore{db: db}
}

func (s *itemStore) Store(ctx context.Context, item storage.Item) error {
	data := map[string]interface{}{
		"id":      item.GetID(),
		"source":  item.GetSource(),
		"content": item.GetContent(),
	}

	jsonData, _ := json.Marshal(data)
	hash := hash.NewHash(jsonData).ComputeHash()

	query := `
		INSERT INTO items (id, hash, source, timestamp)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, item.GetID(), hash, item.GetSource(), item.GetTimestamp())
	if err != nil {
		return fmt.Errorf("failed to store item: %w", err)
	}

	return nil
}

func (s *itemStore) Exists(ctx context.Context, id string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE id = ?`, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}

func (s *itemStore) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE hash = ?`, hash).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check hash existence: %w", err)
	}
	return count > 0, nil
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
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM published WHERE item_id = ? AND target = ?`, itemID, target).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check published status: %w", err)
	}
	return count > 0, nil
}

func (s *itemStore) DeleteOlderThan(ctx context.Context, age time.Duration) error {
	cutoff := time.Now().Add(-age)
	query := `DELETE FROM items WHERE timestamp < ?`

	slog.Debug("Deleting items older than cutoff", "age", age, "cutoff", cutoff.Format(time.RFC3339))
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old items: %w", err)
	}

	if rows, err := result.RowsAffected(); err == nil {
		slog.Debug("Deleted old items", "count", rows)
	}

	return nil
}
