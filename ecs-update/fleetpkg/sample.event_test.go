package fleetpkg

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
)

func TestSampleEvent(t *testing.T) {
	sampleEvent, err := ReadYAMLDocument[SampleEvent]("testdata/my_package/data_stream/item_usages/sample_event.json")
	require.NoError(t, err)

	sampleEventECSVersionPath, err := yamlpath.NewPath("$.ecs.version")
	require.NoError(t, err)

	nodes, err := sampleEventECSVersionPath.Find(&sampleEvent.Node)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	nodes[0].Value = "8.3.0"

	ifc, err := yamlNodeToInterface(&sampleEvent.Node)
	require.NoError(t, err)

	out, err := json.MarshalIndent(ifc, "", "  ")
	require.NoError(t, err)

	assert.Contains(t, string(out), strings.TrimSpace(`
  "ecs": {
    "version": "8.3.0"
  },`))
}
