package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml/parser"
	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEdit(t *testing.T) {
	dir := t.TempDir()
	if err := cp.Copy("testdata/my_package", dir); err != nil {
		t.Fatal(err)
	}

	pkg, err := fleetpkg.Read(dir)
	if err != nil {
		t.Fatal(err)
	}

	var cfg EditConfig
	cfg.BuildManifest.ECSReference = "git@9.1.2"
	cfg.Manifest.FormatVersion = "2.6.0"
	cfg.Manifest.FixDottedKeys = true
	cfg.IngestPipeline.ECSVersion = "11.12.13"
	cfg.IngestPipeline.NormalizeOnFailure = true
	cfg.SampleEvent.ECSVersion = "1.2.3"

	result, err := Edit(pkg, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Results
	assert.True(t, result.Changed, "result/changed")

	assert.True(t, result.BuildManifest.Changed, "result/build_manifest/changed")
	assert.Equal(t, "git@8.2", result.BuildManifest.ECSReferenceOld, "result/build_manifest/ecs_old")
	assert.Equal(t, cfg.BuildManifest.ECSReference, result.BuildManifest.ECSReferenceNew, "result/build_manifest/ecs_new")

	assert.True(t, result.Manifest.FormatVersionChanged, "result/manifest/changed")
	assert.Equal(t, "1.0.0", result.Manifest.FormatVersionOld, "result/manifest/format_version_old")
	assert.Equal(t, cfg.Manifest.FormatVersion, result.Manifest.FormatVersionNew, "result/manifest/format_version_old")

	assert.True(t, result.IngestPipelinesChanged(), "result/ingest_pipeline/changed")
	ipr := result.IngestPipeline["data_stream/item_usages/elasticsearch/ingest_pipeline/default.yml"]
	assert.True(t, ipr.ChangedECSVersion)
	assert.Equal(t, "8.2.0", ipr.ECSVersionOld)
	assert.Equal(t, cfg.IngestPipeline.ECSVersion, ipr.ECSVersionNew)
	assert.True(t, ipr.ChangedOnFailure, "result/ingest_pipeline/default.yml/changed_on_failure")

	// Changed Package
	pkg, err = fleetpkg.Read(dir)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cfg.BuildManifest.ECSReference,
		pkg.Build.Dependencies.ECS.Reference,
		"build_manifest.dependencies.ecs.reference")

	assert.Equal(t, cfg.Manifest.FormatVersion,
		pkg.Manifest.FormatVersion,
		"manifest.format_version")
	assert.Empty(t, pkg.Manifest.License, "manifest.license")
	assert.Empty(t, pkg.Manifest.Release, "manifest.release")

	assert.Equal(t, pkg.Manifest.Conditions.Kibana.Version, "^7.16.0 || ^8.0.0")
	manifestContents, err := os.ReadFile(filepath.Join(dir, "manifest.yml"))
	require.NoError(t, err)
	assert.NotContains(t, string(manifestContents), "kibana.version")

	pipeline := pkg.DataStreams["item_usages"].Pipelines["default.yml"]
	assert.Equal(t, cfg.IngestPipeline.ECSVersion,
		pipeline.Processors[3].Attributes["value"],
		"default.yml set processor ecs.version")

	assert.Equal(t, cfg.SampleEvent.ECSVersion,
		pkg.DataStreams["item_usages"].SampleEvent.Event["ecs"].(map[string]any)["version"],
		"sample_event.json ecs.version")

	t.Logf("%#v", result)
}

func TestIndent(t *testing.T) {
	in := `
foo:
  bar: baz
`

	expected := `
  foo:
    bar: baz
`

	assert.Equal(t, expected, indent(in, 2))
}

func TestFixDottedMapKeys(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{
			in: `
foo:
  bar.baz: hello, world!
`,
			out: `
foo:
  bar:
    baz: hello, world!
`,
		},
		{
			in: `
foo:
  bar.baz:
    complex: object
`,
			out: `
foo:
  bar:
    baz:
      complex: object
`,
		},
		// Note: this function won't merge children under the same parent.
		{
			in: `
foo:
  alpha.bravo: foo
  alpha.charlie: bar
`,
			out: `
foo:
  alpha:
    bravo: foo
  alpha:
    charlie: bar
`,
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f, err := parser.ParseBytes([]byte(tc.in), parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}

			changed, err := fixDottedMapKeys(f, "$.foo")
			if err != nil {
				t.Fatal(err)
			}
			if tc.out == "" && changed {
				t.Fatal("changed==true, but not changes were expected")
			}

			assert.Equal(t, strings.TrimSpace(tc.out), strings.TrimSpace(f.String()))
		})
	}
}

func TestAddChangelog(t *testing.T) {
	dir := t.TempDir()
	if err := cp.Copy("testdata/my_package", dir); err != nil {
		t.Fatal(err)
	}

	pkg, err := fleetpkg.Read(dir)
	if err != nil {
		t.Fatal(err)
	}

	const description = "TEST FIX"
	newVersion, err := addChangelogEntry(pkg, enhancementChange, "https://example.com", description)
	if err != nil {
		t.Fatal(err)
	}

	if err = setManifestVersion(pkg.Manifest.Path(), newVersion); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(pkg.Changelog.Path())
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, string(data), description)
}
