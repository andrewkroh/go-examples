package main

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	semmver "github.com/Masterminds/semver/v3" // Masterminds
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

	assert.True(t, result.BuildManifest.ECSReferenceChanged, "result/build_manifest/changed")
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
@@ -27,10 +26,0 @@
-- external: ecs
-  name: tags
-- name: process
-  type: group
-  fields:
-    - name: io
-      type: group
-      fields:
-        - name: bytes_skipped.length
-          external: ecs
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

var updateConstraintsTests = []struct {
	name        string
	current     string
	update      string
	want        string
	wantChanged bool
}{
	{
		name:        "zero",
		current:     "0",
		update:      "0",
		want:        "0",
		wantChanged: false,
	},
	{
		name:        "normal_updated",
		current:     "^8.11.0",
		update:      "^8.12.0",
		want:        "^8.12.0",
		wantChanged: true,
	},
	{
		name:        "normal_not_updated",
		current:     "^8.11.0",
		update:      "^8.10.0",
		want:        "^8.11.0",
		wantChanged: false,
	},
	{
		name:        "multi_one_updated",
		current:     "^7.17.0 || ^8.11.0",
		update:      "^7.17.0 || ^8.12.0",
		want:        "^7.17.0 || ^8.12.0",
		wantChanged: true,
	},
	{
		name:        "multi_both_updated",
		current:     "^7.17.0 || ^8.11.0",
		update:      "^7.17.1 || ^8.12.0",
		want:        "^7.17.1 || ^8.12.0",
		wantChanged: true,
	},
	{
		name:        "multi_not_updated",
		current:     "^7.17.0 || ^8.11.0",
		update:      "^7.17.0 || ^8.10.0",
		want:        "^7.17.0 || ^8.11.0",
		wantChanged: false,
	},
	{
		name:        "multi_drop_updated",
		current:     "^7.17.0 || ^8.11.0",
		update:      "^8.12.0",
		want:        "^8.12.0",
		wantChanged: true,
	},
}

func TestUpdateConstraints(t *testing.T) {
	for _, test := range updateConstraintsTests {
		t.Run(test.name, func(t *testing.T) {
			c, err := semmver.NewConstraint(test.current)
			if err != nil {
				t.Fatalf("failed to parse current constraint: %v", err)
			}
			u, err := semmver.NewConstraint(test.update)
			if err != nil {
				t.Fatalf("failed to parse update constraint: %v", err)
			}
			got, changed, err := updateConstraints(c, u)
			if err != nil {
				t.Fatalf("unexpected error from updateConstraints(%s, %s): %v", test.current, test.update, err)
			}
			if got.String() != test.want {
				t.Errorf("unexpected constraint from updateConstraints(%s, %s): got=%s want=%s", test.current, test.update, got, test.want)
			}
			if changed != test.wantChanged {
				t.Errorf("unexpected constraint from updateConstraints(%s, %s): got=%t want=%t", test.current, test.update, changed, test.wantChanged)
			}
		})
	}
}

// TestAllElasticIntegrations is a smoke test that executes an edit on every
// package in the main branch of the elastic/integrations repo.
func TestAllElasticIntegrations(t *testing.T) {
	repoDir := cloneIntegrations(t, mirrorElasticIntegrations(t))

	allPackages, err := filepath.Glob(filepath.Join(repoDir, "packages/*"))
	if err != nil {
		t.Fatal(err)
	}

	for _, pkgDir := range allPackages {
		t.Run(filepath.Base(pkgDir), func(t *testing.T) {
			pkg, err := fleetpkg.Read(pkgDir)
			if err != nil {
				t.Fatal(err)
			}

			var cfg EditConfig
			cfg.Manifest.FixDottedKeys = true
			noImportMappings := false
			cfg.BuildManifest.ECSImportMappings = &noImportMappings
			cfg.BuildManifest.ECSReference = "git@v8.11.0"
			cfg.IngestPipeline.ECSVersion = "8.11.0"
			cfg.FieldsYML.DropECS = true

			_, err = Edit(pkg, cfg)
			if err != nil {
				t.Fatal(pkgDir, err)
			}
		})
	}
}

// mirrorElasticIntegrations creates a local mirror of the elastic/integrations
// repo inside the temp dir. If it already exists, then it will perform an update.
func mirrorElasticIntegrations(t *testing.T) string {
	t.Helper()

	const cloneURL = `https://github.com/elastic/integrations.git`
	clonePath := filepath.Join(os.TempDir(), "ecs-update-test/elastic-integrations")
	t.Log("Local elastic/integrations git mirror path:", clonePath)

	switch _, err := os.Stat(clonePath); {
	case errors.Is(err, fs.ErrNotExist):
		start := time.Now()
		cmd := exec.Command("git", "clone", "-q", "--bare", "--depth=1", cloneURL, clonePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		t.Log(time.Since(start))
	case err == nil:
		info, _ := os.Stat(filepath.Join(clonePath, "FETCH_HEAD"))
		if info != nil && info.ModTime().After(time.Now().Add(-1*time.Hour)) {
			t.Log("Skipping 'git remote update' because it was updated in the last hour.")
			break
		}

		cmd := exec.Command("git", "remote", "update")
		cmd.Dir = clonePath
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal(err)
	}

	return clonePath
}

// cloneIntegrations clones the elastic/integrations repo into a test-local
// temporary directory.
func cloneIntegrations(t *testing.T, remote string) string {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "ecs-update-test/elastic-integrations")
	update := exec.Command("git", "clone", "-q", remote, repoPath)
	update.Stdout = os.Stdout
	update.Stderr = os.Stderr
	if err := update.Run(); err != nil {
		t.Fatal(err)
	}

	if testing.Verbose() {
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = repoPath
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Testing with elastic/integrations@%v", string(out))
	}

	return repoPath
}
