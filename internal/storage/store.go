package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

type Store struct {
	items ItemStore
	feeds FeedStore
}

func New(dbPath string) (*Store, error) {
	slog.Info("Initializing storage", "path", dbPath)

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(conn); err != nil {
		conn.Close()
		return nil, err
	}

	slog.Info("Storage initialized successfully")

	return &Store{
		items: newItemStore(conn),
		feeds: newFeedStore(conn),
	}, nil
}

func runMigrations(conn *sql.DB) error {
	slog.Debug("Running database migrations")

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	migrationsDir := filepath.Join("db", "migrations")
	if _, err := os.Stat(migrationsDir); err != nil {
		if os.IsNotExist(err) {
			slog.Debug("Migrations directory not found, skipping migrations", "path", migrationsDir)
			return nil
		}
		return fmt.Errorf("failed to access migrations directory: %w", err)
	}

	if err := goose.Up(conn, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Debug("Migrations completed successfully")
	return nil
}

func (s *Store) Items() ItemStore {
	return s.items
}

func (s *Store) Feed() FeedStore {
	return s.feeds
}
