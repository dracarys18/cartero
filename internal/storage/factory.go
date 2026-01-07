package storage

import (
	"context"
	"fmt"

	"cartero/internal/config"
)

var factoryFuncs = map[string]func(string) (StorageInterface, error){}

func RegisterFactory(storageType string, fn func(string) (StorageInterface, error)) {
	factoryFuncs[storageType] = fn
}

func New(ctx context.Context, cfg config.StorageConfig) (StorageInterface, error) {
	storageType := cfg.Type
	if storageType == "" {
		storageType = "sqlite"
	}

	fn, exists := factoryFuncs[storageType]
	if !exists {
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}

	return fn(cfg.Path)
}
