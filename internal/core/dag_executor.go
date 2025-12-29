package core

import (
	"context"
	"fmt"
	"log"
	"maps"
	"sync"
)

type ProcessorNode struct {
	Name      string
	Processor Processor
	DependsOn []string
}

type ProcessorExecutor struct {
	nodes              map[string]*ProcessorNode
	executionOrder     []string
	executionOrderOnce sync.Once
	mu                 sync.RWMutex
}

func NewProcessorExecutor() *ProcessorExecutor {
	return &ProcessorExecutor{
		nodes: make(map[string]*ProcessorNode),
	}
}

func (pe *ProcessorExecutor) AddProcessor(name string, processor Processor, dependsOn []string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.nodes[name] = &ProcessorNode{
		Name:      name,
		Processor: processor,
		DependsOn: dependsOn,
	}
}

func (pe *ProcessorExecutor) Initialize() error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if len(pe.nodes) == 0 {
		return nil
	}

	if err := pe.validateGraph(pe.nodes); err != nil {
		return err
	}

	order, err := pe.TopologicalSort(pe.nodes)
	if err != nil {
		return err
	}

	pe.executionOrder = order
	log.Printf("Pipeline: Processor execution order: %v", pe.executionOrder)
	return nil
}

func (pe *ProcessorExecutor) ExecuteProcessors(ctx context.Context, item *Item) (*ProcessedItem, error) {

	processed := &ProcessedItem{
		Original: item,
		Data:     item.Content,
		Metadata: make(map[string]any),
	}

	pe.mu.RLock()
	executionOrder := pe.executionOrder
	nodes := make(map[string]*ProcessorNode)
	maps.Copy(nodes, pe.nodes)
	pe.mu.RUnlock()

	for _, processorName := range executionOrder {
		node := nodes[processorName]
		log.Printf("Executing processor: %s (depends_on=%v)", node.Name, node.DependsOn)

		result, err := node.Processor.Process(ctx, item)
		if err != nil {
			log.Printf("Processor %s failed: %v", node.Name, err)
			return nil, fmt.Errorf("processor %s error: %w", node.Name, err)
		}

		if result == nil {
			// Item was filtered out by this processor
			log.Printf("Processor %s filtered out item %s", node.Name, item.ID)
			return nil, nil
		}

		processed = result
		log.Printf("Processor %s completed successfully", node.Name)
	}

	return processed, nil
}

func (pe *ProcessorExecutor) validateGraph(nodes map[string]*ProcessorNode) error {
	for name, node := range nodes {
		for _, dep := range node.DependsOn {
			if _, exists := nodes[dep]; !exists {
				return fmt.Errorf("processor %s depends on %s which does not exist", name, dep)
			}
		}
	}
	return nil
}

func (pe *ProcessorExecutor) TopologicalSort(nodes map[string]*ProcessorNode) ([]string, error) {
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	result := make([]string, 0, len(nodes))

	for name := range nodes {
		if !visited[name] {
			if err := pe.visit(name, nodes, visited, visiting, &result); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func (pe *ProcessorExecutor) visit(name string, nodes map[string]*ProcessorNode, visited, visiting map[string]bool, result *[]string) error {
	if visiting[name] {
		return fmt.Errorf("cycle detected in processor dependencies involving %s", name)
	}

	if visited[name] {
		return nil
	}

	visiting[name] = true

	node := nodes[name]
	for _, dep := range node.DependsOn {
		if err := pe.visit(dep, nodes, visited, visiting, result); err != nil {
			return err
		}
	}

	visiting[name] = false
	visited[name] = true
	*result = append(*result, name)

	return nil
}
