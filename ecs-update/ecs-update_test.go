package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
	cp "github.com/otiai10/copy"
	"github.com/pmezard/go-difflib/difflib"
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
	cfg.Manifest.AddOwnerType = true
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

func TestModifyIngestPipelineSetECSVersionViaFindReplace(t *testing.T) {
	dir := t.TempDir()
	if err := cp.Copy("testdata/my_package", dir); err != nil {
		t.Fatal(err)
	}

	pkg, err := fleetpkg.Read(dir)
	if err != nil {
		t.Fatal(err)
	}

	e := &packageEditor{}
	e.config.IngestPipeline.ECSVersion = "11.12.13"

	var r IngestPipelineResult
	pipeline := pkg.DataStreams["item_usages"].Pipelines["default.yml"]
	err = e.modifyIngestPipeline(&pipeline, &r)
	require.NoError(t, err)

	before, err := os.ReadFile("testdata/my_package/data_stream/item_usages/elasticsearch/ingest_pipeline/default.yml")
	require.NoError(t, err)
	after, err := os.ReadFile(pipeline.Path())
	require.NoError(t, err)

	// Only a single one should be changed!
	expectedChange := `
***************
*** 18 ****
!       value: "8.2.0"
--- 18 ----
!       value: "11.12.13"
`[1:]

	diff, err := difflib.GetContextDiffString(difflib.ContextDiff{
		A: difflib.SplitLines(string(before)),
		B: difflib.SplitLines(string(after)),
	})
	require.NoError(t, err)
	if diff != expectedChange {
		t.Errorf("unexpected changes found in %s", filepath.Base(pipeline.Path()))
	}
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

func TestYAMLEditString(t *testing.T) {
	testCases := []struct {
		typ token.Type
		in  string
		out string
	}{
		{
			typ: token.StringType,
			in: `
owner:
  github: team
`,
			out: `
owner:
  github: team
  type: elastic
`,
		},
		{
			typ: token.DoubleQuoteType,
			in: `
owner:
  github: team
  other: foo
`,
			out: `
owner:
  github: team
  other: foo
  type: "elastic"
`,
		},
		{
			typ: token.StringType,
			in: `
owner:
  github: team
  type: elastic
`,
			out: `
owner:
  github: team
  type: elastic
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

			err = yamlEditString(f, "$.owner.type", "elastic", tc.typ)
			if err != nil {
				t.Fatal(err)
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
	assert.Contains(t, string(data), `version: "1.5.0"`, "version must be quoted")
}

func TestReplaceECSFields(t *testing.T) {
	dir := t.TempDir()
	if err := cp.Copy("testdata/my_package", dir); err != nil {
		t.Fatal(err)
	}

	pkg, err := fleetpkg.Read(dir)
	if err != nil {
		t.Fatal(err)
	}

	fieldsFile := pkg.DataStreams["item_usages"].Fields["base-fields.yml"]

	before, err := os.ReadFile(fieldsFile.Path())
	if err != nil {
		t.Fatal(err)
	}

	f, err := parser.ParseFile(fieldsFile.Path(), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	changed, err := fieldsYMLUseExternalECS(f, fieldsFile)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, changed, "expected changes")

	changed, err = fieldsYMLRemoveUnknownOrInvalidAttributes(f, fieldsFile)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, changed, "expected changes")

	expectedChange := `
@@ -4,3 +3,0 @@
-  title: Input type.
-  group: 1
-  level: extended
@@ -8,2 +5 @@
-  type: constant_keyword
-  description: Data stream type.
+  external: ecs
@@ -11,2 +7 @@
-  type: constant_keyword
-  description: Data stream dataset.
+  external: ecs
@@ -14,2 +9 @@
-  type: constant_keyword
-  description: Data stream namespace.
+  external: ecs
@@ -24,3 +18,2 @@
-- name: '@timestamp'
-  type: date
-  description: Event timestamp.
+- name: "@timestamp"
+  external: ecs
`[1:]

	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A: difflib.SplitLines(string(before)),
		B: difflib.SplitLines(f.String()),
	})
	require.NoError(t, err)
	if diff != expectedChange {
		t.Errorf("unexpected changes found in %s", filepath.Base(fieldsFile.Path()))
	}
}

func TestRemoveECSFields(t *testing.T) {
	dir := t.TempDir()
	if err := cp.Copy("testdata/my_package", dir); err != nil {
		t.Fatal(err)
	}

	pkg, err := fleetpkg.Read(dir)
	if err != nil {
		t.Fatal(err)
	}

	fieldsFile := pkg.DataStreams["item_usages"].Fields["base-fields.yml"]

	before, err := os.ReadFile(fieldsFile.Path())
	if err != nil {
		t.Fatal(err)
	}

	f, err := parser.ParseFile(fieldsFile.Path(), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	changed, err := fieldsYMLDropExternalECS(f, fieldsFile)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, changed, "expected changes")

	expectedChange := `
@@ -7,9 +6,0 @@
-- name: data_stream.type
-  type: constant_keyword
-  description: Data stream type.
-- name: data_stream.dataset
-  type: constant_keyword
-  description: Data stream dataset.
-- name: data_stream.namespace
-  type: constant_keyword
-  description: Data stream namespace.
@@ -24,5 +14,0 @@
-- name: '@timestamp'
-  type: date
-  description: Event timestamp.
-- external: ecs
-  name: tags
`[1:]

	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A: difflib.SplitLines(string(before)),
		B: difflib.SplitLines(f.String()),
	})
	require.NoError(t, err)
	if diff != expectedChange {
		t.Errorf("unexpected changes found in %s\n%s", filepath.Base(fieldsFile.Path()), diff)
	}
}

func TestCompleteRemoveECSFields(t *testing.T) {
	dir := t.TempDir()
	if err := cp.Copy("testdata/my_package", dir); err != nil {
		t.Fatal(err)
	}

	pkg, err := fleetpkg.Read(dir)
	if err != nil {
		t.Fatal(err)
	}

	fieldsFile := pkg.DataStreams["item_usages"].Fields["ecs.yml"]

	before, err := os.ReadFile(fieldsFile.Path())
	if err != nil {
		t.Fatal(err)
	}

	f, err := parser.ParseFile(fieldsFile.Path(), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	changed, err := fieldsYMLDropExternalECS(f, fieldsFile)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, changed, "expected changes")
	assert.True(t, completeRemoval(f), "expected file deletion")

	expectedChange := `
@@ -1,42 +1 @@
-- external: ecs
-  name: ecs.version
-- external: ecs
-  name: related.user
-- external: ecs
-  name: related.ip
-- external: ecs
-  name: event.kind
-- external: ecs
-  name: event.category
-- external: ecs
-  name: event.type
-- external: ecs
-  name: event.created
-- external: ecs
-  name: event.action
-- external: ecs
-  name: user.id
-- external: ecs
-  name: user.full_name
-- external: ecs
-  name: user.email
-- external: ecs
-  name: source.as.number
-- external: ecs
-  name: source.as.organization.name
-- external: ecs
-  name: source.geo.city_name
-- external: ecs
-  name: source.geo.continent_name
-- external: ecs
-  name: source.geo.country_iso_code
-- external: ecs
-  name: source.geo.country_name
-- external: ecs
-  name: source.geo.location
-- external: ecs
-  name: source.geo.region_iso_code
-- external: ecs
-  name: source.geo.region_name
-- external: ecs
-  name: source.ip
+[]
`[1:]

	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A: difflib.SplitLines(string(before)),
		B: difflib.SplitLines(f.String()),
	})
	require.NoError(t, err)
	if diff != expectedChange {
		t.Errorf("unexpected changes found in %s\n%s", filepath.Base(fieldsFile.Path()), diff)
	}
}
