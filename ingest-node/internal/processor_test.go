package deps

import (
	"testing"

	ingestnode "github.com/andrewkroh/go-ingest-node"
)

func TestProcessor(t *testing.T) {
	p := &Processor{
		container: &ingestnode.ProcessorContainer{
			Set: &ingestnode.SetProcessor{
				Field: "foo",
			},
		},
	}

	typeName, proc := p.value()
	if proc == nil {
		t.Errorf("value() returned nil, expected non-nil")
	}
	if typeName != "set" {
		t.Errorf("name != set, got %q", typeName)
	}
}
