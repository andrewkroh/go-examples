package deps

import (
	"fmt"
	ingestnode "github.com/andrewkroh/go-ingest-node"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRead(t *testing.T) {
	f, err := os.Open("../testdata/simple.yml")
	if err != nil {
		t.Fatal(err)
	}

	p, err := readPipeline(f)
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range p.Processors {
		yml, _ := yaml.Marshal(p.YAML())
		fmt.Printf("%s\n---\n%s\n%+v\n", p.Type(), yml, p.Value())
	}
}

func TestTemplateField(t *testing.T) {
	f := ingestnode.Field("{{{json.ip}}}")

	keys, err := mustacheVariables(string(f))
	if err != nil {
		t.Fatal(err)
	}

	t.Log(keys)
}
