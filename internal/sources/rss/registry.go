package rss

import (
	"fmt"
	"sync"
)

var (
	loaders = make(map[string]SourceLoader)
	mu      sync.RWMutex
)

func RegisterLoader(loaderType string, loader SourceLoader) {
	mu.Lock()
	defer mu.Unlock()
	loaders[loaderType] = loader
}

func GetLoader(loaderType string) (SourceLoader, error) {
	mu.RLock()
	defer mu.RUnlock()

	loader, exists := loaders[loaderType]
	if !exists {
		return nil, fmt.Errorf("unknown loader type: %s", loaderType)
	}

	return loader, nil
}
