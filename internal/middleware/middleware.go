package middleware

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"cartero/internal/types"
)

type ProcessorChain struct {
	state      types.StateAccessor
	processors map[string]types.Processor
	order      []string
	mu         sync.RWMutex
}

var _ types.ProcessorChain = (*ProcessorChain)(nil)

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

	order, err := pc.topologicalSort()
	if err != nil {
		panic(fmt.Sprintf("failed to build processor chain: %v", err))
	}

	pc.order = order
}

func (pc *ProcessorChain) Execute(ctx context.Context, state types.StateAccessor, item *types.Item) error {
	logger := state.GetLogger()
	order := pc.getExecutionOrder()

	logger.Info("Starting processor chain execution", "processor_count", len(order), "order", order)
	for _, name := range order {
		processor := pc.processors[name]
		logger.Info("Executing processor", "processor", name)
		if err := processor.Process(ctx, pc.state, item); err != nil {
			logger.Error("Processor failed", "processor", name, "error", err)
			return fmt.Errorf("processor %s failed: %w", name, err)
		}
		logger.Debug("Processor completed successfully", "processor", name)
	}

	logger.Info("Processor chain execution completed successfully")
	return nil
}

func (pc *ProcessorChain) getExecutionOrder() []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if pc.order == nil {
		panic("processor order not initialized")
	}

	return pc.order

}

func (pc *ProcessorChain) topologicalSort() ([]string, error) {
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	for name, proc := range pc.processors {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}

		for _, dep := range proc.DependsOn() {
			if _, ok := pc.processors[dep]; !ok {
				return nil, fmt.Errorf("processor %s depends on %s which doesn't exist", name, dep)
			}
			adjList[dep] = append(adjList[dep], name)
			inDegree[name]++
		}
	}

	queue := []string{}
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, neighbor := range adjList[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(pc.processors) {
		return nil, fmt.Errorf("circular dependency detected in processors")
	}

	return result, nil
}
