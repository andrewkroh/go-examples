package graph_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/go-examples/fydler/internal/graph"
)

type node string

func (n node) ID() string { return string(n) }

func TestTopologicalSort(t *testing.T) {
	a, b, c, d := node("A"), node("B"), node("C"), node("D")
	g := graph.New([]graph.Node{d, c, b, a}, []graph.Edge{
		{From: a, To: b},
		{From: b, To: c},
		{From: b, To: d},
	})

	sorted, err := graph.TopologicalSort(g)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []graph.Node{a, b, c, d}, sorted)
}
