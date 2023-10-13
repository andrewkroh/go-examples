package deps

import (
	"encoding/binary"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"hash/crc64"
)

type Node struct {
	id      int64
	proc    *Processor
	inputs  []string
	outputs []string
}

func newNode(p *Processor) *Node {
	h := crc64.New(crc64.MakeTable(crc64.ISO))
	for _, idx := range p.address {
		if err := binary.Write(h, binary.BigEndian, idx); err != nil {
			panic(err)
		}
	}
	in, out, err := p.dataFlow()
	if err != nil {
		panic(err)
	}
	return &Node{
		id:      int64(h.Sum64()),
		proc:    p,
		inputs:  in,
		outputs: out,
	}
}

//func (g *Node) DOTID() string {
//	return "Processor " + strconv.FormatInt(g.id, 10)
//}

// ID allows GraphNode to satisfy the graph.Node interface.
func (g *Node) ID() int64 {
	return g.id
}

func (n *Node) Attributes() []encoding.Attribute {
	return []encoding.Attribute{
		{Key: "label", Value: n.proc.Address() + "." + n.proc.Type()},
	}
}

// Edge implements graph.Edge.
var _ graph.Edge = (*Edge)(nil)

// Edge implements encoding.Attributer to label edges in graphviz.
var _ encoding.Attributer = (*Edge)(nil)

type Edge struct {
	from, to *Node
	label    string
}

func newEdge(x, y *Node, label string) *Edge {
	return &Edge{from: x, to: y, label: label}
}

func (e *Edge) From() graph.Node {
	return e.from
}

func (e *Edge) To() graph.Node {
	return e.to
}

func (e *Edge) ReversedEdge() graph.Edge {
	return &Edge{
		from: e.to,
		to:   e.from,
	}
}

func (e *Edge) Attributes() []encoding.Attribute {
	if e.label == "" {
		return nil
	}
	return []encoding.Attribute{
		{Key: "label", Value: e.label},
	}
}
