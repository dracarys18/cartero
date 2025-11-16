package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cartero/internal/core"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
	db   *sql.DB
	path string
}

func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	log.Printf("SQLite storage: opening database at %s", path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &SQLiteStorage{
		db:   db,
		path: path,
	}

	if err := storage.initialize(); err != nil {
		db.Close()
		return nil, err
	}

	log.Printf("SQLite storage: database initialized successfully")
	return storage, nil
}

func (s *SQLiteStorage) initialize() error {
	log.Printf("SQLite storage: initializing schema")
	schema := `
	CREATE TABLE IF NOT EXISTS items (
		id TEXT PRIMARY KEY,
		hash TEXT NOT NULL,
		source TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS published (
		item_id TEXT NOT NULL,
		target TEXT NOT NULL,
		published_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (item_id, target),
		FOREIGN KEY (item_id) REFERENCES items(id)
	);

	CREATE INDEX IF NOT EXISTS idx_items_hash ON items(hash);
	CREATE INDEX IF NOT EXISTS idx_items_source ON items(source);
	CREATE INDEX IF NOT EXISTS idx_items_timestamp ON items(timestamp);
	CREATE INDEX IF NOT EXISTS idx_published_item_id ON published(item_id);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		log.Printf("SQLite storage: error initializing schema: %v", err)
	}
	return err
}

func (s *SQLiteStorage) Store(ctx context.Context, item *core.Item) error {
	hash := s.computeHash(item)

	query := `
		INSERT INTO items (id, hash, source, timestamp)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, item.ID, hash, item.Source, item.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to store item: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) Get(ctx context.Context, id string) (*core.Item, error) {
	return nil, fmt.Errorf("get not supported in hash-only storage")
}

func (s *SQLiteStorage) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT COUNT(*) FROM items WHERE id = ?`

	var count int
	err := s.db.QueryRowContext(ctx, query, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return count > 0, nil
}

func (s *SQLiteStorage) MarkPublished(ctx context.Context, itemID, target string) error {
	query := `
		INSERT INTO published (item_id, target)
		VALUES (?, ?)
		ON CONFLICT(item_id, target) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, itemID, target)
	if err != nil {
		log.Printf("SQLite storage: error marking item %s as published to %s: %v", itemID, target, err)
		return fmt.Errorf("failed to mark as published: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) IsPublished(ctx context.Context, itemID, target string) (bool, error) {
	query := `SELECT COUNT(*) FROM published WHERE item_id = ? AND target = ?`

	var count int
	err := s.db.QueryRowContext(ctx, query, itemID, target).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check published status: %w", err)
	}

	return count > 0, nil
}

func (s *SQLiteStorage) Close() error {
	log.Printf("SQLite storage: closing database connection")
	return s.db.Close()
}

func (s *SQLiteStorage) DeleteOlderThan(ctx context.Context, age time.Duration) error {
	cutoff := time.Now().Add(-age)
	query := `DELETE FROM items WHERE timestamp < ?`

	log.Printf("SQLite storage: deleting items older than %v (cutoff: %s)", age, cutoff.Format(time.RFC3339))
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		log.Printf("SQLite storage: error deleting old items: %v", err)
		return fmt.Errorf("failed to delete old items: %w", err)
	}

	if rows, err := result.RowsAffected(); err == nil {
		log.Printf("SQLite storage: deleted %d old items", rows)
	}

	return nil
}

func (s *SQLiteStorage) computeHash(item *core.Item) string {
	data := map[string]interface{}{
		"id":      item.ID,
		"source":  item.Source,
		"content": item.Content,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

func (s *SQLiteStorage) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	query := `SELECT COUNT(*) FROM items WHERE hash = ?`

	var count int
	err := s.db.QueryRowContext(ctx, query, hash).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check hash existence: %w", err)
	}

	return count > 0, nil
}
