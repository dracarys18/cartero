package core

import (
	"context"
	"log/slog"
	"maps"
	"sync"

	"cartero/internal/graph"
	"cartero/internal/types"
)

type ProcessorNode struct {
	Name      string
	Processor types.Processor
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

func (pe *ProcessorExecutor) AddProcessor(typ string, processor types.Processor) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.nodes[typ] = &ProcessorNode{
		Name:      typ,
		Processor: processor,
		DependsOn: processor.DependsOn(),
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

	order, err := graph.TopologicalSort(nodes)
	if err != nil {
		return err
	}

	pe.executionOrder = order
	slog.Info("Pipeline: Processor execution order", "order", pe.executionOrder)
	return nil
}

func (pe *ProcessorExecutor) ExecuteProcessors(ctx context.Context, item *types.Item) error {
	pe.mu.RLock()
	executionOrder := pe.executionOrder
	nodes := make(map[string]*ProcessorNode)
	maps.Copy(nodes, pe.nodes)
	pe.mu.RUnlock()

	for _, processorName := range executionOrder {
		node := nodes[processorName]
		slog.Debug("Executing processor", "processor", node.Name, "depends_on", node.DependsOn)

		err := node.Processor.Process(ctx, nil, item)
		if err != nil {
			slog.Warn("Processor stopped processing for item", "processor", node.Name, "item_id", item.ID, "error", err)
			return err
		}

		slog.Debug("Processor completed successfully", "processor", node.Name)
	}

	return nil
}
