package components

import (
	"context"
	"fmt"

	"cartero/internal/graph"
)

type IComponent interface {
	Name() string
	Dependencies() []string
	Validate() error
	Initialize(ctx context.Context) error
	Close(ctx context.Context) error
}

type Registry struct {
	components map[string]IComponent
	order      []string
}

func NewRegistry() *Registry {
	return &Registry{
		components: make(map[string]IComponent),
		order:      make([]string, 0),
	}
}

func (r *Registry) Register(component IComponent) error {
	name := component.Name()
	if _, exists := r.components[name]; exists {
		return fmt.Errorf("component %s already registered", name)
	}
	r.components[name] = component
	return nil
}

func (r *Registry) Get(name string) IComponent {
	comp, exists := r.components[name]
	if !exists {
		panic(fmt.Sprintf("component %s not found", name))
	}
	return comp
}

func (r *Registry) InitializeAll(ctx context.Context) error {
	nodes := make(map[string]graph.Node)
	for name, comp := range r.components {
		nodes[name] = &componentNode{comp: comp}
	}

	order, err := graph.TopologicalSort(nodes)
	if err != nil {
		return err
	}

	for _, name := range order {
		comp := r.components[name]
		if err := comp.Validate(); err != nil {
			return fmt.Errorf("component %s validation failed: %w", name, err)
		}
	}

	for _, name := range order {
		comp := r.components[name]
		if err := comp.Initialize(ctx); err != nil {
			return fmt.Errorf("component %s initialization failed: %w", name, err)
		}
	}

	r.order = order
	return nil
}

type componentNode struct {
	comp IComponent
}

func (cn *componentNode) GetName() string {
	return cn.comp.Name()
}

func (cn *componentNode) GetDependencies() []string {
	return cn.comp.Dependencies()
}

func (r *Registry) CloseAll(ctx context.Context) error {
	for i := len(r.order) - 1; i >= 0; i-- {
		name := r.order[i]
		comp := r.components[name]
		if err := comp.Close(ctx); err != nil {
			fmt.Printf("Error closing component %s: %v\n", name, err)
		}
	}
	return nil
}
