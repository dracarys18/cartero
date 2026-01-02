package filters

import (
	"context"
	"fmt"

	"cartero/internal/core"
)

type MetadataFilterProcessor struct {
	name  string
	key   string
	value interface{}
}

func NewMetadataFilterProcessor(name string, key string, value interface{}) *MetadataFilterProcessor {
	return &MetadataFilterProcessor{
		name:  name,
		key:   key,
		value: value,
	}
}

func (m *MetadataFilterProcessor) Name() string {
	return m.name
}

func (m *MetadataFilterProcessor) DependsOn() []string {
	return []string{}
}

func (m *MetadataFilterProcessor) Process(ctx context.Context, item *core.Item) error {
	if item.Metadata == nil {
		return fmt.Errorf("MetadataFilterProcessor %s: item %s has no metadata", m.name, item.ID)
	}

	if val, exists := item.Metadata[m.key]; exists {
		if val != m.value {
			return fmt.Errorf("MetadataFilterProcessor %s: item %s metadata[%s] = %v, expected %v", m.name, item.ID, m.key, val, m.value)
		}
		return nil
	}

	return fmt.Errorf("MetadataFilterProcessor %s: item %s does not have metadata key %s", m.name, item.ID, m.key)
}

func MetadataFilter(name string, key string, value interface{}) *MetadataFilterProcessor {
	return NewMetadataFilterProcessor(name, key, value)
}
