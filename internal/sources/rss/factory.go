package rss

import (
	"cartero/internal/types"
	"fmt"
)

type SourceConfig struct {
	Type  string
	Kind  string
	Value string
}

func (c SourceConfig) LoaderKey() string {
	if c.Kind == "" {
		return c.Type
	}
	return c.Type + "_" + c.Kind
}

func NewSource(name string, config SourceConfig, maxItems int) (types.Source, error) {
	loader, err := GetLoader(config.LoaderKey())
	if err != nil {
		return nil, fmt.Errorf("failed to get loader: %w", err)
	}

	feeds, err := loader.Load(config.Value, maxItems)
	if err != nil {
		return nil, fmt.Errorf("failed to load feeds: %w", err)
	}

	if len(feeds) == 0 {
		return nil, fmt.Errorf("no feeds loaded")
	}

	return NewMultiRSSSource(name, feeds), nil
}
