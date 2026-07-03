package filters

import (
	"context"

	"cartero/internal/dag"
	"cartero/internal/types"
)

const (
	filterBlocklist       = "blocklist"
	filterPublishedDedupe = "published_dedupe"
	filterRank            = "rank"
	filterRerank          = "rerank"
	filterDiversify       = "diversify"
	filterLimit           = "limit"
)

type Filter interface {
	Name() string
	DependsOn() []string
	Filter(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error)
}

type Chain struct {
	filters map[string]Filter
	order   []string
}

func NewChain(fs ...Filter) *Chain {
	c := &Chain{filters: make(map[string]Filter, len(fs))}
	for _, f := range fs {
		c.filters[f.Name()] = f
	}
	sorter := dag.NewTopologicalSorter()
	for _, f := range c.filters {
		sorter.AddNode(f.Name(), filterNode{f})
	}
	order, err := sorter.Sort()
	if err != nil {
		panic("filter chain: " + err.Error())
	}
	c.order = order
	return c
}

func (c *Chain) Filter(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	logger := state.GetLogger()
	for _, name := range c.order {
		f := c.filters[name]
		out, err := f.Filter(ctx, state, items)
		if err != nil {
			return nil, err
		}
		logger.Info("filter applied", "filter", name, "in", len(items), "out", len(out))
		items = out
	}
	return items, nil
}

type filterNode struct{ f Filter }

func (n filterNode) GetName() string           { return n.f.Name() }
func (n filterNode) GetDependencies() []string { return n.f.DependsOn() }

type fromProcessor struct {
	name string
	p    types.Processor
}

func FromProcessor(name string, p types.Processor) Filter { return fromProcessor{name: name, p: p} }

func (a fromProcessor) Name() string { return a.name }
func (a fromProcessor) DependsOn() []string {
	return append([]string{filterBlocklist}, a.p.DependsOn()...)
}

func (a fromProcessor) Filter(ctx context.Context, state types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	logger := state.GetLogger()
	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		err := a.p.Process(ctx, state, item)
		if types.IsFiltered(err) {
			continue
		}
		if err != nil {
			logger.Error("filter: processor error, dropping item", "filter", a.p.Name(), "item_id", item.ID, "error", err)
			continue
		}
		out = append(out, item)
	}
	return out, nil
}
