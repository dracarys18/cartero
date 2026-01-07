package dag

import (
	"fmt"
)

type DependencyProvider interface {
	GetName() string
	GetDependencies() []string
}

type node struct {
	name      string
	dependsOn []string
}

type TopologicalSorter struct {
	nodes map[string]*node
}

func NewTopologicalSorter() *TopologicalSorter {
	return &TopologicalSorter{
		nodes: make(map[string]*node),
	}
}

func (ts *TopologicalSorter) AddNode(name string, depProvider DependencyProvider) {
	ts.nodes[name] = &node{
		name:      name,
		dependsOn: depProvider.GetDependencies(),
	}
}

func (ts *TopologicalSorter) Sort() ([]string, error) {
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	for name, n := range ts.nodes {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}

		for _, dep := range n.dependsOn {
			if _, ok := ts.nodes[dep]; !ok {
				return nil, fmt.Errorf("node %s depends on %s which doesn't exist", name, dep)
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

	if len(result) != len(ts.nodes) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}
