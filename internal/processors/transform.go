package processors

import (
	"context"
	"fmt"
	"strings"

	procnames "cartero/internal/processors/names"
	"cartero/internal/types"
)

type TransformProcessor struct {
	name        string
	transformFn func(*types.Item) (interface{}, error)
}

type TransformFunc func(*types.Item) (interface{}, error)

func NewTransformProcessor(name string, transformFn TransformFunc) *TransformProcessor {
	return &TransformProcessor{
		name:        name,
		transformFn: transformFn,
	}
}

func (t *TransformProcessor) Name() string {
	return t.name
}

func (t *TransformProcessor) DependsOn() []string {
	return []string{procnames.Blocklist}
}

func (t *TransformProcessor) Process(_ context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	logger := st.GetLogger()
	if t.transformFn == nil {
		return items, nil
	}

	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		transformed, err := t.transformFn(item)
		if err != nil {
			logger.Debug("transform: dropped item", "processor", t.name, "item_id", item.ID, "reason", "transform failed", "error", err)
			continue
		}
		item.ModifyContent(transformed)
		item.AddMetadata("transformed", true)
		item.AddMetadata("transformer", t.name)
		out = append(out, item)
	}
	return out, nil
}

func FieldExtractor(name string, fields []string) *TransformProcessor {
	return NewTransformProcessor(name, func(item *types.Item) (interface{}, error) {
		result := make(map[string]interface{})

		if item.Metadata != nil {
			for _, field := range fields {
				if value, exists := item.Metadata[field]; exists {
					result[field] = value
				}
			}
		}

		result["source"] = item.Source
		result["timestamp"] = item.Timestamp

		return result, nil
	})
}

func TemplateTransformer(name string, template string) *TransformProcessor {
	return NewTransformProcessor(name, func(item *types.Item) (interface{}, error) {
		output := template

		replacements := map[string]string{
			"{id}":        item.ID,
			"{source}":    item.Source,
			"{timestamp}": item.Timestamp.String(),
		}

		for key, value := range item.Metadata {
			if str, ok := value.(string); ok {
				replacements[fmt.Sprintf("{%s}", key)] = str
			} else {
				replacements[fmt.Sprintf("{%s}", key)] = fmt.Sprintf("%v", value)
			}
		}

		for placeholder, value := range replacements {
			output = strings.ReplaceAll(output, placeholder, value)
		}

		return output, nil
	})
}

