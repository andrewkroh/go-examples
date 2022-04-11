package changelog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const releaseYAML = `# newer versions go on top
- version: "1.2.1"
  changes:
    - description: Add documentation for multi-fields
      type: enhancement
      link: https://github.com/elastic/integrations/pull/2916
`

func TestFromYAML(t *testing.T) {
	var cl Changelog

	err := yaml.Unmarshal([]byte(releaseYAML), &cl)
	require.NoError(t, err)
	require.Len(t, cl, 1)

	rel, err := NewReleaseFromNode(cl[0])
	require.NoError(t, err)
	assert.EqualValues(t, "1.2.1", rel.Version)
	require.Len(t, rel.Changes, 1)

	change := rel.Changes[0]
	assert.Equal(t, "Add documentation for multi-fields", change.Description)
	assert.Equal(t, "enhancement", change.Type)
	assert.Equal(t, "https://github.com/elastic/integrations/pull/2916", change.Link)
}

func TestToYAML(t *testing.T) {
	rel := Release{
		Version: "1.2.1",
		Changes: []Change{
			{
				Description: "Add documentation for multi-fields",
				Type:        "enhancement",
				Link:        "https://github.com/elastic/integrations/pull/2916",
			},
		},
	}

	node, err := rel.ToYAMLNode()
	require.NoError(t, err)
	node.HeadComment = "# newer versions go on top"

	cl := Changelog{*node}
	data, err := yaml.Marshal(cl)
	require.NoError(t, err)
	assert.Equal(t, releaseYAML, string(data))
}

func TestDoubleQuotedYAMLString(t *testing.T) {
	var ver VersionString = "1.2.3"
	yml, err := yaml.Marshal(ver)
	require.NoError(t, err)
	assert.Equal(t, `"1.2.3"`, strings.TrimSpace(string(yml)))
}
