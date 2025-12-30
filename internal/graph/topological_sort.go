package graph

import (
	"fmt"
)

type Node interface {
	GetName() string
	GetDependencies() []string
}

func TopologicalSort(nodes map[string]Node) ([]string, error) {
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	result := make([]string, 0, len(nodes))

	var visit func(string) error
	visit = func(name string) error {
		if visited[name] {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("cycle detected in dependencies involving %s", name)
		}

		node, exists := nodes[name]
		if !exists {
			return fmt.Errorf("node %s not found", name)
		}

		visiting[name] = true

		for _, dep := range node.GetDependencies() {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[name] = false
		visited[name] = true
		result = append(result, name)
		return nil
	}

	for name := range nodes {
		if !visited[name] {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func ValidateGraph(nodes map[string]Node) error {
	for name, node := range nodes {
		for _, dep := range node.GetDependencies() {
			if _, exists := nodes[dep]; !exists {
				return fmt.Errorf("node %s depends on %s which does not exist", name, dep)
			}
		}
	}
	return nil
}
