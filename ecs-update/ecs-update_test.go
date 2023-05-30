package main

import (
	"testing"

	"github.com/andrewkroh/go-fleetpkg"
	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
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

	assert.True(t, result.Manifest.Changed, "result/manifest/changed")
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
