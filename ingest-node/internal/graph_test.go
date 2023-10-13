package deps

import (
	"fmt"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"os"
	"testing"
)

func TestGraph(t *testing.T) {
	f, err := os.Open("../testdata/simple.yml")
	if err != nil {
		t.Fatal(err)
	}

	p, err := readPipeline(f)
	if err != nil {
		t.Fatal(err)
	}

	g := simple.NewDirectedGraph()

	var nodes []*Node
	for i := range p.Processors {
		n := newNode(&p.Processors[i])
		nodes = append(nodes, n)
		g.AddNode(n)
	}
	for i := range p.OnFailure {
		n := newNode(&p.OnFailure[i])
		nodes = append(nodes, n)
		g.AddNode(n)
	}

	for _, x := range nodes {
		for _, in := range x.inputs {
			for _, y := range nodes {
				if x.id == y.id {
					continue
				}

				for _, out := range y.outputs {
					if in == out {
						g.SetEdge(newEdge(y, x, in))
					}
				}
			}
		}
	}

	sorted, err := topo.Sort(g)
	if err != nil {
		t.Fatal(err)
	}

	for _, n := range sorted {
		n := n.(*Node)
		fmt.Printf("%s.%s\n", n.proc.Address(), n.proc.Type())
	}

	graphviz, err := dot.Marshal(g, "pipeline", "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", graphviz)
}
