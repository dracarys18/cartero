package components

import (
	"context"
	"fmt"

	"cartero/internal/storage"
)

type StorageComponent struct {
	dbPath string
	store  *storage.Store
}

func NewStorageComponent(dbPath string) *StorageComponent {
	return &StorageComponent{
		dbPath: dbPath,
	}
}

func (c *StorageComponent) Name() string {
	return StorageComponentName
}

func (c *StorageComponent) Dependencies() []string {
	return []string{}
}

func (c *StorageComponent) Validate() error {
	if c.dbPath == "" {
		return fmt.Errorf("storage: database path is required")
	}
	return nil
}

func (c *StorageComponent) Initialize(ctx context.Context) error {
	store, err := storage.New(c.dbPath)
	if err != nil {
		return fmt.Errorf("storage: failed to initialize store: %w", err)
	}

	c.store = store
	return nil
}

func (c *StorageComponent) Close(ctx context.Context) error {
	return nil
}

func (c *StorageComponent) Store() *storage.Store {
	return c.store
}
