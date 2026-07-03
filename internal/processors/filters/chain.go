package filters

import (
	"context"

	"cartero/internal/dag"
	"cartero/internal/types"
)

const (
	filterPublishedDedupe = "published_dedupe"
	filterRank            = "rank"
	filterRerank          = "rerank"
	filterDiversify       = "diversify"
	filterLimit           = "limit"
)

type Processor interface {
	Name() string
	DependsOn() []string
	Process(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error)
}

type Chain struct {
	processors map[string]Processor
	order      []string
}

func NewChain(ps ...Processor) *Chain {
	c := &Chain{processors: make(map[string]Processor, len(ps))}
	for _, p := range ps {
		c.processors[p.Name()] = p
	}
	sorter := dag.NewTopologicalSorter()
	for _, p := range c.processors {
		sorter.AddNode(p.Name(), procNode{p})
	}
	order, err := sorter.Sort()
	if err != nil {
		panic("processor chain: " + err.Error())
	}
	c.order = order
	return c
}

func (c *Chain) Process(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	logger := state.GetLogger()
	for _, name := range c.order {
		p := c.processors[name]
		out, err := p.Process(ctx, state, items)
		if err != nil {
			return nil, err
		}
		logger.Info("processor applied", "processor", name, "in", len(items), "out", len(out))
		items = out
	}
	return items, nil
}

type procNode struct{ p Processor }

func (n procNode) GetName() string           { return n.p.Name() }
func (n procNode) GetDependencies() []string { return n.p.DependsOn() }
