package filters

import (
	"context"
	"fmt"

	"cartero/internal/types"
)

type MetadataFilterProcessor struct {
	name string
}

func NewMetadataFilterProcessor(name string, key string, value interface{}) *MetadataFilterProcessor {
	return &MetadataFilterProcessor{
		name: name,
	}
}

func (m *MetadataFilterProcessor) Name() string {
	return m.name
}

func (m *MetadataFilterProcessor) DependsOn() []string {
	return []string{}
}

func (m *MetadataFilterProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	if item.Metadata == nil {
		return fmt.Errorf("MetadataFilterProcessor %s: item %s has no metadata", m.name, item.ID)
	}

	return nil
}

func MetadataFilter(name string, key string, value interface{}) *MetadataFilterProcessor {
	return NewMetadataFilterProcessor(name, key, value)
}
