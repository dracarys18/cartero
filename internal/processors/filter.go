package processors

import (
	"context"
	"fmt"
	"strings"

	"cartero/internal/core"
)

type FilterProcessor struct {
	name       string
	filterFn   func(*core.Item) bool
	skipOnFail bool
}

type FilterFunc func(*core.Item) bool

func NewFilterProcessor(name string, filterFn FilterFunc) *FilterProcessor {
	return &FilterProcessor{
		name:       name,
		filterFn:   filterFn,
		skipOnFail: true,
	}
}

func (f *FilterProcessor) Name() string {
	return f.name
}

func (f *FilterProcessor) Process(ctx context.Context, item *core.Item) (*core.ProcessedItem, error) {
	processed := &core.ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]interface{}),
		Skip:     false,
	}

	if f.filterFn != nil {
		if !f.filterFn(item) {
			processed.Skip = true
			processed.Metadata["filtered"] = true
			processed.Metadata["filter"] = f.name
		}
	}

	return processed, nil
}

func MinScoreFilter(name string, minScore int) *FilterProcessor {
	return NewFilterProcessor(name, func(item *core.Item) bool {
		if score, ok := item.Metadata["score"].(int); ok {
			return score >= minScore
		}
		return true
	})
}

func KeywordFilter(name string, keywords []string, mode string) *FilterProcessor {
	return NewFilterProcessor(name, func(item *core.Item) bool {
		if title, ok := item.Metadata["title"].(string); ok {
			titleLower := strings.ToLower(title)

			switch mode {
			case "include":
				for _, keyword := range keywords {
					if strings.Contains(titleLower, strings.ToLower(keyword)) {
						return true
					}
				}
				return false
			case "exclude":
				for _, keyword := range keywords {
					if strings.Contains(titleLower, strings.ToLower(keyword)) {
						return false
					}
				}
				return true
			default:
				return true
			}
		}
		return true
	})
}

func MetadataFilter(name string, key string, value interface{}) *FilterProcessor {
	return NewFilterProcessor(name, func(item *core.Item) bool {
		if item.Metadata == nil {
			return false
		}

		if val, exists := item.Metadata[key]; exists {
			return val == value
		}

		return false
	})
}

func ChainFilters(name string, filters ...*FilterProcessor) *FilterProcessor {
	return NewFilterProcessor(name, func(item *core.Item) bool {
		for _, filter := range filters {
			if filter.filterFn != nil && !filter.filterFn(item) {
				return false
			}
		}
		return true
	})
}

type CompositeFilterProcessor struct {
	name       string
	processors []core.Processor
}

func NewCompositeFilterProcessor(name string, processors ...core.Processor) *CompositeFilterProcessor {
	return &CompositeFilterProcessor{
		name:       name,
		processors: processors,
	}
}

func (c *CompositeFilterProcessor) Name() string {
	return c.name
}

func (c *CompositeFilterProcessor) Process(ctx context.Context, item *core.Item) (*core.ProcessedItem, error) {
	processed := &core.ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]interface{}),
		Skip:     false,
	}

	for _, processor := range c.processors {
		result, err := processor.Process(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("processor %s failed: %w", processor.Name(), err)
		}

		if result.Skip {
			processed.Skip = true
			processed.Metadata = result.Metadata
			return processed, nil
		}

		for k, v := range result.Metadata {
			processed.Metadata[k] = v
		}

		processed.Data = result.Data
	}

	return processed, nil
}
