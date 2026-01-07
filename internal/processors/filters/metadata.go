package filters

import (
	"context"

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
		logger := st.GetLogger()
		logger.Info("MetadataFilterProcessor rejected item", "processor", m.name, "item_id", item.ID, "reason", "no metadata")
		return types.NewFilteredError(m.name, item.ID, "no metadata found")
	}

	return nil
}

func MetadataFilter(name string, key string, value interface{}) *MetadataFilterProcessor {
	return NewMetadataFilterProcessor(name, key, value)
}
