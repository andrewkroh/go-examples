// Package graph contains a simple graph data structure. It contains
// a topological sorting function for ordering nodes in a DAG.
package graph

import (
	"cmp"
	"errors"
	"slices"
	"strings"

	"golang.org/x/exp/maps"
)

var errGraphIsCyclic = errors.New("graph has at least one cycle")

type Node interface {
	ID() string
}

type Edge struct {
	From, To Node
}

func (e Edge) String() string {
	return e.From.ID() + " -> " + e.To.ID()
}

type Graph struct {
	nodes []Node
	edges map[Edge]struct{}
}

func New(nodes []Node, edges []Edge) *Graph {
	edgeSet := map[Edge]struct{}{}
	for _, e := range edges {
		edgeSet[e] = struct{}{}
	}
	return &Graph{
		nodes: nodes,
		edges: edgeSet,
	}
}

// String generates a graphviz representation of the graph.
// You can write the output to a .dot file and generate a
// visualization with:
//
//	dot -Tsvg deps.dot
func (g *Graph) String() string {
	var sb strings.Builder
	sb.WriteString("digraph G {\n")
	for _, n := range g.nodes {
		sb.WriteString("  ")
		sb.WriteString(n.ID())
		sb.WriteByte(';')
		sb.WriteByte('\n')
	}
	for e := range g.edges {
		sb.WriteString("  ")
		sb.WriteString(e.String())
		sb.WriteByte(';')
		sb.WriteByte('\n')
	}
	sb.WriteByte('}')
	return sb.String()
}

// Nodes returns all the nodes in the graph.
func (g *Graph) Nodes() []Node {
	return g.nodes
}

// OutgoingNodesFrom returns all nodes that have an edge leading from n.
func (g *Graph) OutgoingNodesFrom(n Node) []Node {
	var nodes []Node
	for e := range g.edges {
		if e.From.ID() == n.ID() {
			nodes = append(nodes, e.To)
		}
	}
	return nodes
}

// IncomingNodesTo returns all nodes that have an edge leading to n.
func (g *Graph) IncomingNodesTo(n Node) []Node {
	var nodes []Node
	for e := range g.edges {
		if e.To.ID() == n.ID() {
			nodes = append(nodes, e.From)
		}
	}
	return nodes
}

// Edge returns the edge from m to n if it exists, otherwise nil is returned.
func (g *Graph) Edge(m, n Node) *Edge {
	for e := range g.edges {
		if e.From.ID() == m.ID() && e.To.ID() == n.ID() {
			return &e
		}
	}
	return nil
}

func (g *Graph) clone() *Graph {
	return &Graph{
		nodes: slices.Clone(g.nodes),
		edges: maps.Clone(g.edges),
	}
}

// TopologicalSort sorts the nodes in the graph topologically using
// Kahn's algorithm. If the graph contains any cycles, then an error
// will be returned.
func TopologicalSort(g *Graph) ([]Node, error) {
	g = g.clone()

	// Nodes that have no incoming edge.
	var s []Node
	for _, n := range g.nodes {
		if len(g.IncomingNodesTo(n)) == 0 {
			s = append(s, n)
			continue
		}
	}
	slices.SortFunc(s, compareNodes)

	// While S is not empty do.
	var l []Node
	for len(s) > 0 {
		// Remove a node n from S.
		n := s[0]
		s = s[1:]
		// Add n to L.
		l = append(l, n)

		// For each node m with an edge e from n to m.
		var nextS []Node
		for _, m := range g.OutgoingNodesFrom(n) {
			if e := g.Edge(n, m); e != nil {
				// Remove edge e from the graph.
				delete(g.edges, *e)
			}
			// If m has no other incoming edges then insert m into S.
			if from := g.IncomingNodesTo(m); len(from) == 0 {
				nextS = append(nextS, m)
			}
		}

		// Modification of Kahn's algorithm to make it deterministic.
		slices.SortFunc(nextS, compareNodes)
		s = append(s, nextS...)
	}

	if len(g.edges) > 0 {
		return nil, errGraphIsCyclic
	}
	return l, nil
}

func compareNodes(a, b Node) int {
	return cmp.Compare(a.ID(), b.ID())
}
