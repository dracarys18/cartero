package processors

import (
	"context"
	"fmt"
	"strings"

	"cartero/internal/processors/names"
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
	return []string{
		names.KeywordFilter,
	}
}

func (t *TransformProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	logger := st.GetLogger()
	if t.transformFn != nil {
		transformed, err := t.transformFn(item)
		if err != nil {
			logger.Info("TransformProcessor rejected item", "processor", t.name, "item_id", item.ID, "reason", "transform failed", "error", err)
			return fmt.Errorf("transform failed: %w", err)
		}
		if err := item.ModifyContent(func() interface{} { return transformed }); err != nil {
			return err
		}
		if err := item.AddMetadata("transformed", true); err != nil {
			return err
		}
		if err := item.AddMetadata("transformer", t.name); err != nil {
			return err
		}
	}

	return nil
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

func EnrichTransformer(name string, enrichments map[string]interface{}) *TransformProcessor {
	return NewTransformProcessor(name, func(item *types.Item) (interface{}, error) {
		result := make(map[string]interface{})

		if data, ok := item.Content.(map[string]interface{}); ok {
			for k, v := range data {
				result[k] = v
			}
		} else {
			result["content"] = item.Content
		}

		for k, v := range enrichments {
			result[k] = v
		}

		for k, v := range item.Metadata {
			result[k] = v
		}

		return result, nil
	})
}

func MapTransformer(name string, mapper func(interface{}) (interface{}, error)) *TransformProcessor {
	return NewTransformProcessor(name, func(item *types.Item) (interface{}, error) {
		return mapper(item.Content)
	})
}

type ChainTransformProcessor struct {
	name         string
	transformers []*TransformProcessor
}

func NewChainTransformProcessor(name string, transformers ...*TransformProcessor) *ChainTransformProcessor {
	return &ChainTransformProcessor{
		name:         name,
		transformers: transformers,
	}
}

func (c *ChainTransformProcessor) Name() string {
	return c.name
}

func (c *ChainTransformProcessor) DependsOn() []string {
	return []string{}
}

func (c *ChainTransformProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	currentData := item.Content

	for _, transformer := range c.transformers {
		tempItem := &types.Item{
			ID:        item.ID,
			Content:   currentData,
			Metadata:  item.Metadata,
			Source:    item.Source,
			Timestamp: item.Timestamp,
		}

		if err := transformer.Process(ctx, st, tempItem); err != nil {
			return fmt.Errorf("transformer %s failed: %w", transformer.Name(), err)
		}

		currentData = tempItem.GetContent()

		for k, v := range tempItem.Metadata {
			if err := item.AddMetadata(k, v); err != nil {
				return err
			}
		}
	}

	return item.ModifyContent(func() any { return currentData })
}
