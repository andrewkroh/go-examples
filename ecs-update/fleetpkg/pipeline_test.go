package fleetpkg

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIngestNodePipeline(t *testing.T) {
	doc, err := ReadYAMLDocument[IngestNodePipeline]("testdata/my_package/data_stream/item_usages/elasticsearch/ingest_pipeline/default.yml")
	require.NoError(t, err)

	nodes, err := ingestNodePipelineSetECSVersionValuePath.Find(&doc.Node)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	nodes[0].Value = "8.3.0"

	// Test round-tripping YAML (this does not retain all white-space).
	buf := new(bytes.Buffer)
	if bytes.HasPrefix(doc.RawYAML, []byte("---")) {
		buf.WriteString("---\n")
	}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	require.NoError(t, enc.Encode(&doc.Node))

	assert.Equal(t, strings.Replace(string(doc.RawYAML), `value: "8.2.0"`, `value: "8.3.0"`, 1), buf.String())
}
