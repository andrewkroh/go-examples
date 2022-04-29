package filter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const yml = `
# File
- name: source
  title: Source
  group: 2
  type: group
  fields:
    - name: geo.city_name
      level: core
      type: keyword
      description: City name.
      ignore_above: 1024
`

func TestRemoveKeys(t *testing.T) {
	// Decode
	var node yaml.Node
	dec := yaml.NewDecoder(bytes.NewBufferString(yml))
	err := dec.Decode(&node)
	require.NoError(t, err)

	// Filter
	numChanges := Keys(&node, "title", "group", "level")
	assert.Equal(t, 3, numChanges)

	// Encode
	out := new(bytes.Buffer)
	enc := yaml.NewEncoder(out)
	enc.SetIndent(2)
	err = enc.Encode(&node)
	require.NoError(t, err)

	const expected = `
# File
- name: source
  type: group
  fields:
    - name: geo.city_name
      type: keyword
      description: City name.
      ignore_above: 1024
`

	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(out.String()))
}
