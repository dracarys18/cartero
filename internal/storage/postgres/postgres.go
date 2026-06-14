package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"cartero/internal/storage"
)

func init() {
	storage.RegisterFactory("postgres", New)
}

type PostgresStorage struct {
	conn  *sql.DB
	items storage.ItemStore
	feeds storage.FeedStore
}

func New(dsn string) (storage.StorageInterface, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)

	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &PostgresStorage{
		conn:  conn,
		items: newItemStore(conn),
		feeds: newFeedStore(conn),
	}, nil
}

func runMigrations(conn *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	migrationsDir := filepath.Join("db", "migrations", "postgres")
	if _, err := os.Stat(migrationsDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to access migrations directory: %w", err)
	}

	if err := goose.Up(conn, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetConnection() *sql.DB {
	return s.conn
}

func (s *PostgresStorage) Items() storage.ItemStore {
	return s.items
}

func (s *PostgresStorage) Feed() storage.FeedStore {
	return s.feeds
}

func (s *PostgresStorage) Close(ctx context.Context) error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
