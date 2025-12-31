package core

import (
	"context"
	"fmt"
	"log"
	"maps"
	"sync"

	"cartero/internal/graph"
)

type ProcessorNode struct {
	Name      string
	Processor Processor
	DependsOn []string
}

func (pn *ProcessorNode) GetName() string {
	return pn.Name
}

func (pn *ProcessorNode) GetDependencies() []string {
	return pn.DependsOn
}

type ProcessorExecutor struct {
	nodes          map[string]*ProcessorNode
	executionOrder []string
	mu             sync.RWMutex
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

	nodes := make(map[string]graph.Node)
	for name, node := range pe.nodes {
		nodes[name] = node
	}

	if err := graph.ValidateGraph(nodes); err != nil {
		return err
	}

	order, err := graph.TopologicalSort(nodes)
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
			log.Printf("Processor %s filtered out item %s", node.Name, item.ID)
			return nil, nil
		}

		processed = result
		log.Printf("Processor %s completed successfully", node.Name)
	}

	return processed, nil
}
