package fleetpkg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
)

func TestReadYAMLDocument(t *testing.T) {
	doc, err := ReadYAMLDocument[Manifest]("testdata/my_package/manifest.yml")
	require.NoError(t, err)

	assert.Equal(t, "1.4.0", doc.OriginalData.Version)
	assert.Contains(t, string(doc.RawYAML), "version: 1.4.0")

	p, err := yamlpath.NewPath("$.version")
	require.NoError(t, err)

	nodes, err := p.Find(&doc.Node)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	assert.Equal(t, "1.4.0", nodes[0].Value)
}

func TestReadPackage(t *testing.T) {
	pkg, err := ReadPackage("testdata/my_package")
	require.NoError(t, err)

	assert.NotNil(t, pkg.BuildManifest)

	assert.Len(t, pkg.DataStreams, 1)
	assert.NotNil(t, pkg.DataStreams["item_usages"].DefaultPipeline)
	assert.NotNil(t, pkg.DataStreams["item_usages"].SampleEvent)
}

func TestEditPackage(t *testing.T) {
	pkg, err := ReadPackage("testdata/my_package")
	require.NoError(t, err)

	if pkg.BuildManifest != nil {
		old, err := pkg.BuildManifest.SetBuildManifestECSReference("git@8.3")
		require.NoError(t, err)
		assert.Equal(t, "git@8.2", old)
	}

	for _, ds := range pkg.DataStreams {
		if ds.DefaultPipeline != nil {
			old, err := ds.DefaultPipeline.SetIngestNodePipelineECSVersion("8.3.0")
			require.NoError(t, err)
			assert.Equal(t, "8.2.0", old)
		}

		if ds.SampleEvent != nil {
			old, err := ds.SampleEvent.SetSampleEventECSVersion("8.3.0")
			require.NoError(t, err)
			assert.Equal(t, "8.2.0", old)
		}
	}

	pkg.BuildManifest.WriteYAML(os.Stdout)
	for _, ds := range pkg.DataStreams {
		ds.DefaultPipeline.WriteYAML(os.Stdout)
		ds.SampleEvent.WriteJSON(os.Stdout, 2)
	}
}
