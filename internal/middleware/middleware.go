package middleware

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"cartero/internal/dag"
	"cartero/internal/types"
)

type processorAdapter struct {
	processor types.Processor
}

func (pa *processorAdapter) GetName() string {
	return pa.processor.Name()
}

func (pa *processorAdapter) GetDependencies() []string {
	return pa.processor.DependsOn()
}

type ProcessorChain struct {
	state      types.StateAccessor
	processors map[string]types.Processor
	order      []string
	mu         sync.RWMutex
}

func New(s types.StateAccessor) types.ProcessorChain {
	return &ProcessorChain{
		state:      s,
		processors: make(map[string]types.Processor),
	}
}

func (pc *ProcessorChain) With(name string, processor types.Processor) types.ProcessorChain {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.processors[name] = processor
	pc.order = nil
	return pc
}

func (pc *ProcessorChain) WithMultiple(procs map[string]types.Processor) types.ProcessorChain {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	maps.Copy(pc.processors, procs)
	pc.order = nil
	return pc
}

func (pc *ProcessorChain) Build() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	sorter := dag.NewTopologicalSorter()
	for name, proc := range pc.processors {
		sorter.AddNode(name, &processorAdapter{processor: proc})
	}

	order, err := sorter.Sort()
	if err != nil {
		panic(fmt.Sprintf("failed to build processor chain: %v", err))
	}

	pc.order = order
}

func (pc *ProcessorChain) Execute(ctx context.Context, state types.StateAccessor, item *types.Item) error {
	logger := state.GetLogger()

	pc.mu.RLock()
	order := pc.order

	if len(order) == 0 {
		pc.mu.RUnlock()
		logger.Debug("No processors configured, skipping processor chain")
		return nil
	}

	orderCopy := make([]string, len(order))
	copy(orderCopy, order)
	pc.mu.RUnlock()

	logger.Info("Starting processor chain execution", "processor_count", len(orderCopy), "order", orderCopy)

	pc.mu.RLock()
	defer pc.mu.RUnlock()

	for _, name := range orderCopy {
		processor := pc.processors[name]
		logger.Info("Executing processor", "processor", name)
		if err := processor.Process(ctx, pc.state, item); err != nil {
			if types.IsFiltered(err) {
				logger.Info("Item filtered", "processor", name, "item_id", item.ID, "reason", err.Error())
				return err
			}
			logger.Error("Processor failed with error", "processor", name, "item_id", item.ID, "error", err)
			return fmt.Errorf("processor %s failed: %w", name, err)
		}
		logger.Debug("Processor completed successfully", "processor", name)
	}

	logger.Info("Processor chain execution completed successfully")
	return nil
}
