package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"text/template"
	"unicode"

	"github.com/coreos/go-semver/semver"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/printer"
	"github.com/goccy/go-yaml/token"
	"github.com/mitchellh/go-wordwrap"
	"golang.org/x/exp/maps"

	"github.com/andrewkroh/go-fleetpkg"
)

var usage = `
ecs-update updates the ECS version referenced by a Fleet package. It does the
following operations:

1. Read the build manifest in _dev/build/build.yml to check the currently used
   ECS version. Set the value to the specified ECS branch or tag.
2. For each data stream, check 'ecs.version' value that the pipeline sets. If the
   pipeline does not have a set ecs.version processor then a warning is logged.
3. Normalize the on_failure processors to have a set event.kind=pipeline_error
   processor and format the error.message value consistently across all packages.
   (when -on-failure is used)
4. For each data stream, update the 'ecs.version' contained in sample_event.json.
5. Build the package to update the README.md.
6. Run the pipeline tests to regenerate the test data.
7. Add a changelog entry (unless -skip-changelog).
8. Commit the changes to the package if not currently in a rebase. The commit
   message will contain the commands used to modify the package.
`[1:]

// Flags
var (
	ecsVersion         semver.Version
	formatVersion      semver.Version
	ecsGitReference    string
	normalizeOnFailure bool
	pullRequestNumber  string
	owner              string
	sampleEvents       bool
	skipChangelog      bool
	fixDottedYAMLKeys  bool
)

var semverZero = semver.Version{}

func init() {
	flag.Var(&ecsVersion, "ecs-version", "ECS version (e.g. 8.3.0)")
	flag.Var(&formatVersion, "format-version", "Fleet package format version (empty or x.y.z)")
	flag.StringVar(&ecsGitReference, "ecs-git-ref", "", "Git reference of ECS repo. Git tags are recommended. "+
		"Defaults to release branch of the ecs-version (e.g. uses 8.3 for 8.3.0).")
	flag.StringVar(&pullRequestNumber, "pr", "", "Pull request number")
	flag.StringVar(&owner, "owner", "", "Only modify packages owned by this team.")
	flag.BoolVar(&sampleEvents, "sample-events", false, "Generate new sample events (slow).")
	flag.BoolVar(&skipChangelog, "skip-changelog", false, "Skip adding a changelog entry.")
	flag.BoolVar(&normalizeOnFailure, "on-failure", false, "Rewrite ingest pipeline on_failure handlers to set event.kind=pipeline_error and normalize the error.message value.")
	flag.BoolVar(&fixDottedYAMLKeys, "fix-dotted-yaml-keys", false, "Replace YAML keys containing dots.")
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "(devel)" {
		return "latest"
	}
	return info.Main.Version
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage+"\nVersion: %s\n\nUsage of %s:\n", getVersion(), filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if ecsVersion != semverZero {
		if ecsGitReference == "" {
			ecsGitReference = fmt.Sprintf("git@%d.%d", ecsVersion.Major, ecsVersion.Minor)
		} else if !strings.HasPrefix(ecsGitReference, "git@") {
			ecsGitReference = "git@" + ecsGitReference
		}
	}

	for _, p := range flag.Args() {
		if err := updatePackage(p); err != nil {
			log.Fatalf("Error: for %v: %v", p, err)
		}
	}
}

func updatePackage(path string) error {
	pkg, err := fleetpkg.Read(path)
	if err != nil {
		return err
	}

	if strings.Contains(strings.ToLower(pkg.Manifest.Description), "deprecated") {
		// Skip
		return nil
	}

	if owner != "" {
		if pkg.Manifest.Owner.Github != owner {
			// Skip
			return nil
		}
	}

	var editCfg EditConfig
	if ecsVersion != semverZero {
		editCfg.IngestPipeline.ECSVersion = ecsVersion.String()
		editCfg.SampleEvent.ECSVersion = ecsVersion.String()
	}
	if formatVersion != semverZero {
		editCfg.Manifest.FormatVersion = formatVersion.String()
	}
	editCfg.Manifest.FixDottedKeys = fixDottedYAMLKeys
	editCfg.BuildManifest.ECSReference = ecsGitReference
	editCfg.IngestPipeline.NormalizeOnFailure = normalizeOnFailure

	result, err := Edit(pkg, editCfg)
	if err != nil {
		return err
	}
	if !result.Changed {
		log.Printf("%s: No changes.", pkg.Manifest.Name)
		return nil
	}

	if err = BuildAndUpdate(path); err != nil {
		return err
	}

	if !skipChangelog {
		if err = AddChangelog(path, pullRequestNumber, title(headline(result))+"."); err != nil {
			return err
		}
	}

	msg, err := CommitMessage{
		Manifest:    pkg.Manifest,
		Headline:    headline(result),
		Summary:     summarize(result),
		GitGenerate: gitGenerate(filepath.Base(pkg.Path())),
	}.Build()
	if err != nil {
		return err
	}

	if err = Commit(path, msg); err != nil {
		return err
	}

	return nil
}

func title(s string) string {
	first := string(s[0])
	remainder := s[1:]
	return strings.ToUpper(first) + remainder
}

func BuildAndUpdate(path string) error {
	if sampleEvents {
		return ExecutePlan(path, []string{
			"elastic-package clean",
			"elastic-package format",
			"elastic-package build",
			"elastic-package stack up -d --services package-registry",
			"elastic-package test system -g",
			"elastic-package test pipeline -g",
			"elastic-package clean",
			"elastic-package format",
			"elastic-package build",
		})
	}
	return ExecutePlan(path, []string{
		"elastic-package format",
		`elastic-package build`,
		`elastic-package test pipeline -g`,
	})
}

func Commit(path, message string) error {
	f, err := os.CreateTemp("", filepath.Base(path))
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	if _, err = f.WriteString(message); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}

	return ExecutePlan(path, []string{
		`git add -u .`,
		`git commit -F ` + f.Name(),
	})
}

func AddChangelog(path, pr, description string) error {
	cmd := fmt.Sprintf(`elastic-package-changelog add-next --type=enhancement -d=%q `, description)
	if pr != "" {
		cmd += "--pr=" + pr
	}

	return ExecutePlan(path, []string{
		cmd,
	})
}

func ExecutePlan(pwd string, plan []string) error {
	for _, cmd := range plan {
		cmd := exec.Command("sh", "-c", cmd)
		cmd.Dir = pwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed running %q: %w", cmd, err)
		}
	}

	return nil
}

var commitTmpl = template.Must(template.New("commit").Funcs(template.FuncMap{
	"wordwrap": wordwrap.WrapString,
}).Parse(strings.TrimSpace(`
[{{ .Manifest.Name }}] - {{ .Headline }}

{{ wordwrap .Summary 80 }}

{{ .GitGenerate }}
`)))

type CommitMessage struct {
	Manifest    fleetpkg.Manifest
	Headline    string
	Summary     string
	GitGenerate string
}

func (m CommitMessage) Build() (string, error) {
	buf := new(bytes.Buffer)
	if err := commitTmpl.Execute(buf, m); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func headline(r *EditResult) string {
	switch {
	case r.BuildManifest.Changed:
		return fmt.Sprintf("change to ECS version %v", r.BuildManifest.ECSReferenceNew)
	case r.Manifest.DottedYAMLRemoved:
		return "removed dotted YAML keys from manifest"
	case r.Manifest.FormatVersionChanged:
		return fmt.Sprintf("change to format_version %v", r.Manifest.FormatVersionNew)
	case r.IngestPipelinesChanged():
		for _, ipr := range r.IngestPipeline {
			if ipr.ChangedECSVersion {
				return fmt.Sprintf("change ecs.version to %v in ingest pipeline", ipr.ECSVersionNew)
			}
		}
		for _, ipr := range r.IngestPipeline {
			if ipr.ChangedOnFailure {
				return "normalize ingest pipeline on_failure handlers"
			}
		}
		// Should never get here.
		return "change ingest pipeline"
	case r.SampleEventsChanged():
		for _, ser := range r.SampleEvent {
			if ser.Changed {
				return fmt.Sprintf("change ecs.version to %v in sample_event.json", ser.ECSVersionNew)
			}
		}
		// Should never get here.
		return "change sample_event.json"
	default:
		return "no changes"
	}
}

func summarize(r *EditResult) string {
	var sb strings.Builder

	if r.BuildManifest.Changed {
		fmt.Fprintf(&sb, "ECS version in build manifest changed from %v to %v. ",
			r.BuildManifest.ECSReferenceOld, r.BuildManifest.ECSReferenceNew)
	}
	if r.Manifest.FormatVersionChanged {
		fmt.Fprintf(&sb, "The format_version in the package manifest changed from %v to %v. ",
			r.Manifest.FormatVersionOld, r.Manifest.FormatVersionNew)
	}
	if r.IngestPipelinesChanged() {
		var newVersion string
		oldVersions := map[string]struct{}{}
		for _, ipr := range r.IngestPipeline {
			if ipr.ChangedECSVersion {
				newVersion = ipr.ECSVersionNew
				oldVersions[ipr.ECSVersionOld] = struct{}{}
			}
		}
		if len(oldVersions) > 0 {
			fmt.Fprintf(&sb, "The set ecs.version processor in pipelines was changed %v. "+
				"Previously the pipeline was setting version %v. ",
				newVersion, strings.Join(maps.Keys(oldVersions), ", "))
		}

		var onFailureChanges int
		for _, ipr := range r.IngestPipeline {
			if ipr.ChangedOnFailure {
				onFailureChanges++
			}
		}
		if onFailureChanges > 0 {
			fmt.Fprintf(&sb, "The on_failure processors in %d of the %d pipelines were "+
				"normalized to set event.kind=pipeline_failure and to have a consistent "+
				"error.message format. ",
				onFailureChanges, len(r.IngestPipeline))
		}
	}
	if r.SampleEventsChanged() {
		var newVersion string
		oldVersions := map[string]struct{}{}
		for _, ser := range r.SampleEvent {
			if ser.Changed {
				newVersion = ser.ECSVersionNew
				oldVersions[ser.ECSVersionOld] = struct{}{}
			}
		}
		if len(oldVersions) > 0 {
			fmt.Fprintf(&sb, "The ecs.version in sample_event.json files was changed to %v. "+
				"Previously sample_event.json files contained %v. ",
				newVersion, strings.Join(maps.Keys(oldVersions), ", "))
		}
	}

	return strings.TrimSpace(sb.String())
}

// gitGenerate returns the commit message containing the commands to
// recreate the commit. See https://pkg.go.dev/rsc.io/rf/git-generate.
func gitGenerate(packageName string) string {
	var sb strings.Builder
	sb.WriteString("[git-generate]\n")

	sb.WriteString("go run github.com/andrewkroh/go-examples/ecs-update@")
	sb.WriteString(getVersion())
	sb.WriteString(" ")

	if ecsVersion != (semver.Version{}) {
		sb.WriteString("-ecs-version=")
		sb.WriteString(ecsVersion.String())
		sb.WriteString(" ")
	}
	if ecsGitReference != "" {
		sb.WriteString("-ecs-git-ref=")
		sb.WriteString(ecsGitReference)
		sb.WriteString(" ")
	}
	if formatVersion != (semver.Version{}) {
		sb.WriteString("-format-version=")
		sb.WriteString(formatVersion.String())
		sb.WriteString(" ")
	}
	if pullRequestNumber != "" {
		sb.WriteString("-pr=")
		sb.WriteString(pullRequestNumber)
		sb.WriteString(" ")
	}
	if skipChangelog {
		sb.WriteString("-skip-changelog")
		sb.WriteString(" ")
	}
	if normalizeOnFailure {
		sb.WriteString("-on-failure")
		sb.WriteString(" ")
	}
	if sampleEvents {
		sb.WriteString("-sample-events")
	}

	sb.WriteString("packages/")
	sb.WriteString(packageName)
	return sb.String()
}

//
// Package editor
//

type EditConfig struct {
	BuildManifest struct {
		ECSReference string // Git reference to an ECS version.
	}
	Manifest struct {
		FormatVersion string // Package format.
		FixDottedKeys bool
	}
	IngestPipeline struct {
		ECSVersion         string // ECS version (e.g. 8.2.0).
		NormalizeOnFailure bool   // Replace or add on_failure processors to achieve a consistent error.message and event.kind=pipeline_error.
	}
	SampleEvent struct {
		ECSVersion string // ECS version (e.g. 8.2.0).
	}
}

type EditResult struct {
	Changed        bool
	BuildManifest  BuildManifestResult
	Manifest       ManifestResult
	IngestPipeline map[string]*IngestPipelineResult
	SampleEvent    map[string]*SampleEventResult
}

func (r EditResult) IngestPipelinesChanged() bool {
	for _, ipr := range r.IngestPipeline {
		if ipr.ChangedECSVersion || ipr.ChangedOnFailure {
			return true
		}
	}
	return false
}

func (r EditResult) SampleEventsChanged() bool {
	for _, spr := range r.SampleEvent {
		if spr.Changed {
			return true
		}
	}
	return false
}

type BuildManifestResult struct {
	Changed         bool
	ECSReferenceOld string
	ECSReferenceNew string
}

type ManifestResult struct {
	DottedYAMLRemoved    bool
	FormatVersionChanged bool
	FormatVersionOld     string
	FormatVersionNew     string
}

type SampleEventResult struct {
	Changed       bool
	ECSVersionOld string // ECS version (e.g. 8.2.0).
	ECSVersionNew string
}

type IngestPipelineResult struct {
	ChangedOnFailure  bool
	ChangedECSVersion bool
	ECSVersionOld     string
	ECSVersionNew     string // ECS version (e.g. 8.2.0).
}

type packageEditor struct {
	config EditConfig
	pkg    *fleetpkg.Integration
	result *EditResult
}

// Edit edits the integration package according to the provided editConfig. If
// it fails the package on disk may be in a partially edited state. Use git to
// restore the package to its previous state.
func Edit(pkg *fleetpkg.Integration, c EditConfig) (*EditResult, error) {
	e := &packageEditor{
		config: c,
		pkg:    pkg,
		result: &EditResult{},
	}

	err := errors.Join(
		e.modifyBuildManifest(),
		e.modifyManifest(),
		e.modifyIngestPipelines(),
		e.modifySampleEvents(),
	)
	if err != nil {
		return nil, err
	}

	e.result.Changed = e.result.BuildManifest.Changed ||
		e.result.Manifest.DottedYAMLRemoved ||
		e.result.Manifest.FormatVersionChanged ||
		e.result.IngestPipelinesChanged() ||
		e.result.SampleEventsChanged()

	return e.result, nil
}

func (e *packageEditor) modifyBuildManifest() error {
	if e.config.BuildManifest.ECSReference == "" {
		return nil
	}
	if e.pkg.Build == nil {
		log.Printf("WARN: %s: No build manifest in package.", e.pkg.Manifest.Name)
		return nil
	}
	if e.config.BuildManifest.ECSReference == e.pkg.Build.Dependencies.ECS.Reference {
		return nil
	}

	f, err := parser.ParseFile(e.pkg.Build.Path(), parser.ParseComments)
	if err != nil {
		return err
	}
	err = yamlEditString(f, "$.dependencies.ecs.reference",
		e.config.BuildManifest.ECSReference, token.DoubleQuoteType)
	if err != nil {
		return err
	}

	e.result.BuildManifest.Changed = true
	e.result.BuildManifest.ECSReferenceOld = e.pkg.Build.Dependencies.ECS.Reference
	e.result.BuildManifest.ECSReferenceNew = e.config.BuildManifest.ECSReference

	return os.WriteFile(e.pkg.Build.Path(), []byte(f.String()+"\n"), 0o644)
}

func (e *packageEditor) modifyManifest() error {
	f, err := parser.ParseFile(e.pkg.Manifest.Path(), parser.ParseComments)
	if err != nil {
		return err
	}

	if e.config.Manifest.FixDottedKeys {
		e.result.Manifest.DottedYAMLRemoved, err = fixDottedMapKeys(f, "$.conditions")
		if err != nil {
			return fmt.Errorf("failed to fix dotted map keys: %w", err)
		}
	}

	if e.config.Manifest.FormatVersion != "" && e.config.Manifest.FormatVersion != e.pkg.Manifest.FormatVersion {
		err = yamlEditString(f, "$.format_version",
			e.config.Manifest.FormatVersion, token.DoubleQuoteType)
		if err != nil {
			return err
		}

		if formatVersion := semver.New(e.config.Manifest.FormatVersion); formatVersion.Major >= 2 {
			// license is disallowed in >= 2.0.0
			if err := removeLicenseField(f); err != nil {
				return err
			}

			// release is disallowed in >= 2.3.0
			if formatVersion.Minor >= 3 {
				if err := removeReleaseField(f); err != nil {
					return err
				}
			}
		}

		e.result.Manifest.FormatVersionChanged = true
		e.result.Manifest.FormatVersionOld = e.pkg.Manifest.FormatVersion
		e.result.Manifest.FormatVersionNew = e.config.Manifest.FormatVersion
	}

	return os.WriteFile(e.pkg.Manifest.Path(), []byte(f.String()+"\n"), 0o644)
}

func (e *packageEditor) modifySampleEvents() error {
	if e.config.SampleEvent.ECSVersion == "" {
		return nil
	}

	e.result.SampleEvent = map[string]*SampleEventResult{}
	for _, ds := range e.pkg.DataStreams {
		if ds.SampleEvent == nil {
			continue
		}

		// Result key is the path to the file relative to the package root.
		path, _ := filepath.Rel(e.pkg.Path(), ds.SampleEvent.Path())
		r := &SampleEventResult{}
		e.result.SampleEvent[filepath.ToSlash(path)] = r

		if err := e.modifySampleEvent(ds.SampleEvent, r); err != nil {
			return err
		}
	}
	return nil
}

func (e *packageEditor) modifySampleEvent(s *fleetpkg.SampleEvent, r *SampleEventResult) error {
	ecs, ok := s.Event["ecs"].(map[string]any)
	if !ok {
		log.Printf("WARN: %s: ecs not found or is not a string.", s.Path())
		return nil
	}
	oldECSVersion, ok := ecs["version"].(string)
	if !ok {
		log.Printf("WARN: %s: ecs.version not found or is not a string.", s.Path())
		return nil
	}
	if e.config.SampleEvent.ECSVersion == oldECSVersion {
		return nil
	}

	s.Event["ecs"].(map[string]any)["version"] = e.config.SampleEvent.ECSVersion

	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")

	if err := enc.Encode(s.Event); err != nil {
		return err
	}

	r.Changed = true
	r.ECSVersionOld = oldECSVersion
	r.ECSVersionNew = e.config.SampleEvent.ECSVersion

	return os.WriteFile(s.Path(), buf.Bytes(), 0o644)
}

func (e *packageEditor) modifyIngestPipelines() error {
	e.result.IngestPipeline = map[string]*IngestPipelineResult{}

	for _, ds := range e.pkg.DataStreams {
		for _, p := range ds.Pipelines {
			// Result key is the path to the file relative to the package root.
			path, _ := filepath.Rel(e.pkg.Path(), p.Path())
			r := &IngestPipelineResult{}
			e.result.IngestPipeline[filepath.ToSlash(path)] = r

			if err := e.modifyIngestPipeline(&p, r); err != nil {
				return fmt.Errorf("failed modifying ingest pipeline at %v: %w", path, err)
			}
		}
	}
	return nil
}

func (e *packageEditor) modifyIngestPipeline(p *fleetpkg.IngestPipeline, r *IngestPipelineResult) error {
	f, err := parser.ParseFile(p.Path(), parser.ParseComments)
	if err != nil {
		return err
	}

	// Check for similar problems to https://github.com/goccy/go-yaml/issues/374.
	if len(f.Docs) != 1 {
		return fmt.Errorf("failed parsing %v: got %d docs expected 1", p.Path(), len(f.Docs))
	}

	// The set ecs.version processor should only be in the default.yml pipeline.
	if e.config.IngestPipeline.ECSVersion != "" &&
		filepath.Base(p.Path()) == "default.yml" {
		if err = e.modifyIngestPipelineSetECSVersion(f, p, r); err != nil {
			return err
		}
	}

	if e.config.IngestPipeline.NormalizeOnFailure {
		if err = e.modifyIngestPipelineOnFailure(f, p, r); err != nil {
			return err
		}
	}

	if r.ChangedECSVersion || r.ChangedOnFailure {
		print := printer.Printer{}
		d := print.PrintNode(f.Docs[0])
		d = append(d, '\n')
		return os.WriteFile(p.Path(), d, 0o644)
	}
	return nil
}

func (e *packageEditor) modifyIngestPipelineSetECSVersion(f *ast.File, p *fleetpkg.IngestPipeline, r *IngestPipelineResult) error {
	idx, oldECSVersion := findSetECSVersion(p)
	if idx < 0 {
		log.Printf("WARN: %s: No set ecs.version processor found in pipeline.", p.Path())
		return nil
	}
	if e.config.IngestPipeline.ECSVersion == oldECSVersion {
		return nil
	}

	if err := yamlEditString(f, fmt.Sprintf("$.processors[%d].set.value", idx), e.config.IngestPipeline.ECSVersion, token.DoubleQuoteType); err != nil {
		return err
	}

	r.ChangedECSVersion = true
	r.ECSVersionOld = oldECSVersion
	r.ECSVersionNew = e.config.IngestPipeline.ECSVersion
	return nil
}

func (*packageEditor) modifyIngestPipelineOnFailure(f *ast.File, p *fleetpkg.IngestPipeline, r *IngestPipelineResult) error {
	setEventKindIndex := findSetEventKindPipelineErrorProcessor(p)
	errorMessageType, errorMessageIndex := findErrorMessageProcessor(p)

	onFailureNode, err := getOnFailureNode(f)
	if err != nil {
		return err
	}
	if onFailureNode == nil {
		// Pipeline has no on_failure.
		return nil
	}

	r.ChangedOnFailure = appendOrReplaceNode(onFailureNode, setEventKindIndex, newSetEventKindPipelineErrorProcessor())

	// If the pipeline uses append then update that. Otherwise, use a set processor because
	// it complies with ECS (error.message is not an array).
	if errorMessageType == appendProcessor {
		if appendOrReplaceNode(onFailureNode, errorMessageIndex, newErrorMessageProcessor(appendProcessor)) {
			r.ChangedOnFailure = true
		}
	} else {
		if appendOrReplaceNode(onFailureNode, errorMessageIndex, newErrorMessageProcessor(setProcessor)) {
			r.ChangedOnFailure = true
		}
	}

	// This tracks if something changed in order to avoid writing
	// unnecessary whitespace changes to the YAML. go-yaml does not
	// guarantee white-space is preserved when round-tripping.
	if r.ChangedOnFailure {
		if _, err = formatScriptProcessors(f, p); err != nil {
			return err
		}

		if err = os.WriteFile(p.Path(), []byte(f.String()), 0o644); err != nil {
			return err
		}
	}
	return nil
}

//
// Package editor helper functions.
//

func yamlEditString(f *ast.File, yamlPath, value string, t token.Type) error {
	p, err := yaml.PathString(yamlPath)
	if err != nil {
		return err
	}

	n, err := p.FilterFile(f)
	if err != nil {
		return err
	}

	switch n := n.(type) {
	case *ast.StringNode:
		n.Value = value
		if n.Token != nil && t >= 0 {
			n.Token.Type = t
		}
		return nil
	default:
		return fmt.Errorf("unexpected field type %T found for %q", n, yamlPath)
	}
}

func yamlDeleteStringNodeFromMap(f *ast.File, yamlPath string) error {
	path, err := yaml.PathString(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to create yaml path: %w", err)
	}

	n, err := path.FilterFile(f)
	if err != nil {
		if yaml.IsNotFoundNodeError(err) {
			return nil
		}
		return fmt.Errorf("failed to get node: %w", err)
	}

	switch n := n.(type) {
	case *ast.StringNode:
		for _, d := range f.Docs {
			m := ast.Parent(d, n)
			if m == nil {
				continue
			}
			switch p := ast.Parent(d, m).(type) {
			case *ast.MappingNode:
				for i, e := range p.Values {
					if e == m {
						p.Values = append(p.Values[:i], p.Values[i+1:]...)
						break
					}
				}
			default:
				return fmt.Errorf("failed to get parent node: %w", err)
			}
		}
	default:
		return fmt.Errorf("unexpected license field type: %T", n)
	}
	return nil
}

func removeLicenseField(file *ast.File) error {
	return yamlDeleteStringNodeFromMap(file, "$.license")
}

func removeReleaseField(file *ast.File) error {
	return yamlDeleteStringNodeFromMap(file, "$.release")
}

func findSetECSVersion(pipeline *fleetpkg.IngestPipeline) (index int, version string) {
	for i, p := range pipeline.Processors {
		if p.Type == "set" {
			if field, ok := p.Attributes["field"].(string); ok && field == "ecs.version" {
				if version, ok := p.Attributes["value"].(string); ok {
					return i, version
				}
			}
		}
	}
	return -1, ""
}

func getOnFailureNode(f *ast.File) (*ast.SequenceNode, error) {
	path, err := yaml.PathString("$.on_failure")
	if err != nil {
		return nil, fmt.Errorf("failed to create on_failure yaml path: %w", err)
	}

	node, err := path.FilterFile(f)
	if err != nil {
		if errors.Is(err, yaml.ErrNotFoundNode) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to filter file for path %v: %w", path.String(), err)
	}

	switch v := node.(type) {
	case *ast.SequenceNode:
		return v, nil
	default:
		return nil, fmt.Errorf("unexpected on_failure type: %T", v)
	}
}

// appendOrReplaceNode will append node to seq if index exists in seq,
// otherwise it appends.
func appendOrReplaceNode(seq *ast.SequenceNode, index int, node ast.Node) bool {
	if index < 0 || index >= len(seq.Values) {
		// Append
		seq.Values = append(seq.Values, node)
		return true
	}

	if !nodeEqual(seq.Values[index], node) {
		// Replace
		seq.Values[index] = node
		return true
	}

	return false
}

// fixDottedMapKeys will locate the map specified by the YAML path and replace
// any key names that contain dots with a nested object. For example, given a
// YAML path of `$.my_map` will convert
//
//	 my_map:
//		  foo.bar: {}
//
// to
//
//	 my_map:
//		  foo:
//		    bar: {}
func fixDottedMapKeys(f *ast.File, mapPath string) (bool, error) {
	path, err := yaml.PathString(mapPath)
	if err != nil {
		return false, err
	}

	node, err := path.FilterFile(f)
	if err != nil {
		if yaml.IsNotFoundNodeError(err) {
			return false, nil
		}
	}

	var changed bool
	switch v := node.(type) {
	// For maps with a single key.
	case *ast.MappingValueNode:
		return fixDottedMapNode(v)
	// For maps with a more than one key.
	case *ast.MappingNode:
		for _, n := range v.Values {
			itemChanged, err := fixDottedMapNode(n)
			if err != nil {
				return false, err
			}

			if itemChanged {
				changed = true
			}
		}
	default:
		return false, fmt.Errorf("node found at path %s is not a map (found %T)", mapPath, node)
	}

	return changed, nil
}

func fixDottedMapNode(original *ast.MappingValueNode) (bool, error) {
	stringKey, ok := original.Key.(*ast.StringNode)
	if !ok {
		return false, fmt.Errorf("found non-string map key: %#v", original.Key)
	}

	before, after, found := strings.Cut(stringKey.Value, ".")
	if !found {
		return false, nil
	}

	node := newNode(before + ":\n  " + after + ": PLACEHOLDER")
	newMapValueNode := node.(*ast.MappingValueNode)

	// Replace the placeholder with the original value.
	// This will allow complex YAML structures to be represented correctly.
	newMapValueNode.Value.(*ast.MappingValueNode).Value = original.Value
	newMapValueNode.AddColumn(original.Start.Position.IndentNum)

	original.Key = newMapValueNode.Key
	original.Value = newMapValueNode.Value
	return true, nil
}

// newNode returns a new ast.Node created from the given YAML string.
func newNode(body string) ast.Node {
	set, err := parser.ParseBytes([]byte(body), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return set.Docs[0].Body
}

// nodeEquals compares the two nodes by marshaling the node to an any
// value and then using [reflect.DeepEqual] to compare the values.
func nodeEqual(a, b ast.Node) bool {
	var x, y any
	if err := yaml.NodeToValue(a, &x); err != nil {
		panic(err)
	}
	if err := yaml.NodeToValue(b, &y); err != nil {
		panic(err)
	}
	return reflect.DeepEqual(x, y)
}

func findSetEventKindPipelineErrorProcessor(p *fleetpkg.IngestPipeline) int {
	for i, p := range p.OnFailure {
		if p.Type != "set" {
			continue
		}
		if s, ok := p.Attributes["field"].(string); ok && s == "event.kind" {
			return i
		}
	}
	return -1
}

func newSetEventKindPipelineErrorProcessor() ast.Node {
	set, err := parser.ParseBytes([]byte(`
set:
  field: event.kind
  value: pipeline_error
`), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return set.Docs[0].Body
}

func findErrorMessageProcessor(p *fleetpkg.IngestPipeline) (processorType, int) {
	for i, p := range p.OnFailure {
		switch t := processorType(p.Type); t {
		case setProcessor, appendProcessor:
			if s, ok := p.Attributes["field"].(string); ok && s == "error.message" {
				return t, i
			}
		}
	}
	return "", -1
}

type processorType string

const (
	setProcessor    processorType = "set"
	appendProcessor processorType = "append"
)

func newErrorMessageProcessor(t processorType) ast.Node {
	set, err := parser.ParseBytes([]byte(fmt.Sprintf(`
%s:
  field: error.message
  value: >-
    Processor '{{ _ingest.on_failure_processor_type }}'
    {{#_ingest.on_failure_processor_tag}}with tag '{{ _ingest.on_failure_processor_tag }}'
    {{/_ingest.on_failure_processor_tag}}failed with message '{{ _ingest.on_failure_message }}'
`, t)), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	return set.Docs[0].Body
}

// formatScriptProcessors is a hack to convert script processor source fields
// into a YAML literal. This is ugly.
func formatScriptProcessors(f *ast.File, p *fleetpkg.IngestPipeline) (changed bool, err error) {
nextProcessor:
	for i, proc := range p.Processors {
		if proc.Type != "script" {
			continue
		}

		path, err := yaml.PathString(fmt.Sprintf("$.processors[%d].script.source", i))
		if err != nil {
			return false, err
		}

		n, err := path.FilterFile(f)
		if err != nil {
			// No source field.
			if yaml.IsNotFoundNodeError(err) {
				continue
			}
			return false, err
		}

		if n, ok := n.(*ast.StringNode); ok {
			switch n.Token.Type {
			case token.DoubleQuoteType, token.SingleQuoteType:
				continue nextProcessor
			}
			// We are only interested in processors that take the form:
			// source:
			//   some.painless.code()
			origin := n.Token.Origin
			if len(origin) > 0 && origin[0] != '\n' {
				continue nextProcessor
			}

			replacement, err := createStringLiteral(origin, n.Token.Position.IndentNum)
			if err != nil {
				return false, err
			}

			parent := ast.Parent(f.Docs[0], n)
			parent.(*ast.MappingValueNode).Value = replacement

			changed = true
		}
	}
	return changed, nil
}

func unindent(s string) string {
	c := countIndent(s)
	if c == 0 {
		return s
	}
	return trimIndent(s, c)
}

func indent(s string, spaces int) string {
	return string(indentBytes([]byte(s), []byte(strings.Repeat(" ", spaces))))
}

// MIT License
// https://github.com/kr/text/blob/702c74938df48b97370179f33ce2107bd7ff3b3e/indent.go#L15
func indentBytes(b, prefix []byte) []byte {
	var res []byte
	bol := true
	for _, c := range b {
		if bol && c != '\n' {
			res = append(res, prefix...)
		}
		res = append(res, c)
		bol = c == '\n'
	}
	return res
}

func trimIndent(s string, spaces int) string {
	old := "\n" + strings.Repeat(" ", spaces)
	s = strings.Replace(s, old, "\n", -1)
	return strings.Trim(s, "\n")
}

func countIndent(s string) int {
	if len(s) == 0 || s[0] != '\n' {
		return 0
	}

	x := s[1:]
	x = strings.TrimLeftFunc(x, unicode.IsSpace)
	indent := len(s) - len(x) - 1

	return indent
}

func createStringLiteral(s string, n int) (ast.Node, error) {
	s = unindent(s)
	s = indent(s, n)
	s = "|\n" + s

	f, err := parser.ParseBytes([]byte(s), 0)
	if err != nil {
		return nil, err
	}

	return f.Docs[0].Body, nil
}
