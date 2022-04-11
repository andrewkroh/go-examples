package changelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const releaseYAML = `# newer versions go on top
- version: 1.2.1
  changes:
    - description: Add documentation for multi-fields
      type: enhancement
      link: https://github.com/elastic/integrations/pull/2916
`

func TestRelease_UnmarshalYAML(t *testing.T) {
	var releases []Release

	err := yaml.Unmarshal([]byte(releaseYAML), &releases)
	require.NoError(t, err)
	require.Len(t, releases, 1)

	rel := releases[0]
	assert.Equal(t, "# newer versions go on top", rel.HeadComment)
	assert.Equal(t, "1.2.1", rel.Version)
	require.Len(t, rel.Changes, 1)

	change := rel.Changes[0]
	assert.Equal(t, "Add documentation for multi-fields", change.Description)
	assert.Equal(t, "enhancement", change.Type)
	assert.Equal(t, "https://github.com/elastic/integrations/pull/2916", change.Link)
}

func TestRelease_MarshalYAML(t *testing.T) {
	rel := []Release{
		{
			HeadComment: "# newer versions go on top",
			Version:     "1.2.1",
			Changes: []Change{
				{
					Description: "Add documentation for multi-fields",
					Type:        "enhancement",
					Link:        "https://github.com/elastic/integrations/pull/2916",
				},
			},
		},
	}

	data, err := yaml.Marshal(rel)
	require.NoError(t, err)
	assert.Equal(t, releaseYAML, string(data))
}

func TestMarshalReleaseWithHeadComment(t *testing.T) {
	r := &Release{
		HeadComment: "# Comment 1",
		Version:     "1.2.3",
	}

	data, err := yaml.Marshal(r)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# Comment 1")
}
