package components

import (
	"context"
	"fmt"

	"cartero/internal/storage"
)

type StorageComponent struct {
	store storage.StorageInterface
}

func NewStorageComponent(st storage.StorageInterface) *StorageComponent {
	return &StorageComponent{
		store: st,
	}
}

func (c *StorageComponent) Name() string {
	return StorageComponentName
}

func (c *StorageComponent) Dependencies() []string {
	return []string{}
}

func (c *StorageComponent) Validate() error {
	if c.store == nil {
		return fmt.Errorf("storage: storage interface is required")
	}
	return nil
}

func (c *StorageComponent) Initialize(ctx context.Context) error {
	return nil
}

func (c *StorageComponent) Close(ctx context.Context) error {
	if c.store != nil {
		return c.store.Close(ctx)
	}
	return nil
}

func (c *StorageComponent) Store() storage.StorageInterface {
	return c.store
}
