package fleetpkg

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBuildManifest(t *testing.T) {
	doc, err := ReadYAMLDocument[BuildManifest]("testdata/my_package/_dev/build/build.yml")
	require.NoError(t, err)

	nodes, err := buildManifestECSReferencePath.Find(&doc.Node)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	nodes[0].Value = "git@8.3"

	buf := new(bytes.Buffer)
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	require.NoError(t, enc.Encode(&doc.Node))

	expected := `
dependencies:
  ecs:
    # A comment.
    reference: git@8.3
`[1:]

	assert.Equal(t, expected, buf.String())
}
