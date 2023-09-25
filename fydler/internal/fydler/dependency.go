package fydler

import (
	"golang.org/x/exp/maps"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/graph"
)

type node struct {
	Analyzer *analysis.Analyzer
}

func newNode(a *analysis.Analyzer) *node {
	return &node{Analyzer: a}
}

func (n *node) ID() string {
	return n.Analyzer.Name
}

// buildGraph builds a directed graph where the analyzers are the nodes (vertices)
// and the edges represent dependencies between the nodes.
func buildGraph(analyzers []*analysis.Analyzer) *graph.Graph {
	nodes := make([]*node, 0, len(analyzers))
	for _, a := range analyzers {
		nodes = append(nodes, &node{Analyzer: a})
	}

	edges := map[graph.Edge]struct{}{}
	nodeSet := map[string]graph.Node{}
	for _, n := range nodes {
		nodeSet[n.ID()] = n
		for _, e := range allEdges(n.Analyzer) {
			edges[e] = struct{}{}
		}
	}
	for e := range edges {
		nodeSet[e.From.ID()] = e.From
		nodeSet[e.To.ID()] = e.To
	}

	return graph.New(maps.Values(nodeSet), maps.Keys(edges))
}

// allEdges recursively follows the required analyzers of n to build
// graph edges representing dependencies.
func allEdges(n *analysis.Analyzer) []graph.Edge {
	var edges []graph.Edge
	for _, r := range n.Requires {
		edges = append(edges, graph.Edge{From: newNode(r), To: newNode(n)})
		edges = append(edges, allEdges(r)...)
	}
	return edges
}

// dependencyOrder returns a list of analyzers ordered such that an analyzer's
// required analyzers always come before it in the list.
func dependencyOrder(analyzers []*analysis.Analyzer) ([]*analysis.Analyzer, error) {
	g := buildGraph(analyzers)

	nodes, err := graph.TopologicalSort(g)
	if err != nil {
		return nil, err
	}

	orderedAnalyzers := make([]*analysis.Analyzer, 0, len(nodes))
	for _, n := range nodes {
		orderedAnalyzers = append(orderedAnalyzers, n.(*node).Analyzer)
	}
	return orderedAnalyzers, nil
}
