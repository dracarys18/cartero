package processors

import (
	"context"
	"fmt"
	"strings"

	"cartero/internal/core"
)

type TransformProcessor struct {
	name        string
	transformFn func(*core.Item) (interface{}, error)
}

type TransformFunc func(*core.Item) (interface{}, error)

func NewTransformProcessor(name string, transformFn TransformFunc) *TransformProcessor {
	return &TransformProcessor{
		name:        name,
		transformFn: transformFn,
	}
}

func (t *TransformProcessor) Name() string {
	return t.name
}

func (t *TransformProcessor) Process(ctx context.Context, item *core.Item) (*core.ProcessedItem, error) {
	processed := &core.ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]interface{}),
	}

	if t.transformFn != nil {
		transformed, err := t.transformFn(item)
		if err != nil {
			return nil, fmt.Errorf("transform failed: %w", err)
		}
		processed.Data = transformed
		processed.Metadata["transformed"] = true
		processed.Metadata["transformer"] = t.name
	}

	return processed, nil
}

func FieldExtractor(name string, fields []string) *TransformProcessor {
	return NewTransformProcessor(name, func(item *core.Item) (interface{}, error) {
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
	return NewTransformProcessor(name, func(item *core.Item) (interface{}, error) {
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
	return NewTransformProcessor(name, func(item *core.Item) (interface{}, error) {
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
	return NewTransformProcessor(name, func(item *core.Item) (interface{}, error) {
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

func (c *ChainTransformProcessor) Process(ctx context.Context, item *core.Item) (*core.ProcessedItem, error) {
	processed := &core.ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]interface{}),
	}

	currentData := item.Content

	for _, transformer := range c.transformers {
		tempItem := &core.Item{
			ID:        item.ID,
			Content:   currentData,
			Metadata:  item.Metadata,
			Source:    item.Source,
			Timestamp: item.Timestamp,
		}

		result, err := transformer.Process(ctx, tempItem)
		if err != nil {
			return nil, fmt.Errorf("transformer %s failed: %w", transformer.Name(), err)
		}

		currentData = result.Data

		for k, v := range result.Metadata {
			processed.Metadata[k] = v
		}
	}

	processed.Data = currentData

	return processed, nil
}
