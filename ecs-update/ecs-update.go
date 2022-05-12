package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"text/template"

	"github.com/coreos/go-semver/semver"
	"go.uber.org/multierr"

	"github.com/andrewkroh/go-examples/ecs-update/fleetpkg"
)

var usage = `
ecs-update updates the ECS version referenced by a Fleet package. It does the
following operations:

1. Read the build manifest in _dev/build/build.yml to check the currently used
   ECS version. Set the value to the specified ECS branch or tag.
2. For each data stream, check 'ecs.version' value that the pipeline sets. It
   must have a 'set' processor otherwise an error occurs.
3. For each data stream, update the 'ecs.version' contained in sample_event.json.
4. Build the package to update the README.md.
5. Run the pipeline tests to regenerate the test data.
6. Add a changelog entry.
7. Commit the changes to the package if not currently in a rebase. The commit
   message will contain the commands used to modify the package.
`[1:]

var (
	ecsVersion        string
	pullRequestNumber string
	owner             string
)

func init() {
	flag.StringVar(&ecsVersion, "ecs-version", "", "ECS version (e.g. 8.3.0)")
	flag.StringVar(&pullRequestNumber, "pr", "", "Pull request number")
	flag.StringVar(&owner, "owner", "", "Only modify packages owned by this team.")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage+"\nVersion: %s\n\nUsage of %s:\n", getVersion(), filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if ecsVersion == "" {
		log.Fatal("-ecs-version is required")
	}

	for _, p := range flag.Args() {
		if err := updatePackage(p, ecsVersion); err != nil {
			log.Fatal("Error:", err)
		}
	}
}

func updatePackage(path, ecsVersion string) error {
	pkg, err := fleetpkg.ReadPackage(path)
	if err != nil {
		return err
	}

	if strings.Contains(strings.ToLower(pkg.Manifest.OriginalData.Description), "deprecated") {
		// Skip
		return nil
	}

	if owner != "" {
		if pkg.Manifest.OriginalData.Owner.Github != owner {
			// Skip
			return nil
		}
	}

	var oldECSReference string
	if pkg.BuildManifest != nil {
		ver, err := semver.NewVersion(ecsVersion)
		if err != nil {
			return err
		}

		newECSReference := "git@" + fmt.Sprintf("%d.%d", ver.Major, ver.Minor)
		oldECSReference, err = pkg.BuildManifest.SetBuildManifestECSReference(newECSReference)
		if err != nil {
			return err
		}

		// No changes.
		if oldECSReference == newECSReference {
			return nil
		}
	}

	oldECSVersions := map[string]struct{}{}
	for dataStreamName, ds := range pkg.DataStreams {
		if ds.DefaultPipeline != nil {
			old, err := ds.DefaultPipeline.SetIngestNodePipelineECSVersion(ecsVersion)
			if err != nil {
				return fmt.Errorf("in %s/%s default pipeline: %w", pkg.Manifest.OriginalData.Name, dataStreamName, err)
			}
			oldECSVersions[old] = struct{}{}

			// Only update sample event if a pipeline exists.
			if ds.SampleEvent != nil {
				_, err := ds.SampleEvent.SetSampleEventECSVersion(ecsVersion)
				if err != nil {
					log.Println("WARN:", pkg.Manifest.OriginalData.Name, "/", dataStreamName, ":", err)
				}
			}
		}
	}

	if pkg.BuildManifest == nil {
		log.Println("WARN:", pkg.Manifest.OriginalData.Name, ": No build manifest found in package.")
		return nil
	}

	err = WriteDocument(pkg.BuildManifest, pkg.BuildManifest.WriteYAML)
	for _, ds := range pkg.DataStreams {
		if ds.DefaultPipeline != nil {
			err = multierr.Append(err, WriteDocument(ds.DefaultPipeline, ds.DefaultPipeline.WriteYAML))
		}
		if ds.SampleEvent != nil {
			err = multierr.Append(err, WriteDocument(ds.SampleEvent, func(w io.Writer) error { return ds.SampleEvent.WriteJSON(w, 4) }))
		}
	}

	if err != nil {
		return err
	}

	if err = BuildAndUpdate(path); err != nil {
		return err
	}

	if err = AddChangelog(path, pullRequestNumber, fmt.Sprintf("Update package to ECS %s.", ecsVersion)); err != nil {
		return err
	}

	var oldPipelineVersions []string
	for v := range oldECSVersions {
		oldPipelineVersions = append(oldPipelineVersions, v)
	}

	msg, err := CommitMessage{
		Manifest:            pkg.Manifest.OriginalData,
		ECSVersion:          ecsVersion,
		OldECSReference:     oldECSReference,
		PipelineECSVersions: oldPipelineVersions,
		PullRequestNumber:   pullRequestNumber,
	}.Build()
	if err != nil {
		return err
	}

	if err = Commit(path, msg); err != nil {
		return err
	}

	return nil
}

func WriteDocument[T any](doc *fleetpkg.YAMLDocument[T], encode func(io.Writer) error) error {
	f, err := os.Create(doc.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(doc.RawYAML)
	return err
}

func BuildAndUpdate(path string) error {
	return ExecutePlan(path, []string{
		`elastic-package build`,
		`elastic-package test pipeline -g`,
	})
}

func Commit(path, message string) error {
	f, err := ioutil.TempFile("", filepath.Base(path))
	if err != nil {
		return err
	}

	f.WriteString(message)
	f.Close()
	defer os.Remove(f.Name())

	return ExecutePlan(path, []string{
		`git add -u .`,
		`git commit -F ` + f.Name(),
	})
}

func AddChangelog(path, pr, description string) error {
	cmd := fmt.Sprintf(`elastic-package-changelog add-next --type=enhancement -d=%q`, description)
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
	"join":        strings.Join,
	"toolVersion": getVersion,
}).Parse(strings.TrimSpace(`
[{{ .Manifest.Name }}] - update ECS to {{ .ECSVersion }}

This updates the {{ .Manifest.Name }} integration to ECS {{ .ECSVersion }}.
{{ if .PipelineECSVersions -}}
It was referencing elastic/ecs {{ .OldECSReference }} and using {{ join .PipelineECSVersions ", " }} in ingest pipelines.
{{ else -}}
It was referencing elastic/ecs {{ .OldECSReference }} and no pipelines set ecs.version.
{{ end }}

[git-generate]
go run github.com/andrewkroh/go-examples/ecs-update@{{ toolVersion }} -ecs-version=8.3.0 {{ if .PullRequestNumber }}-pr={{ .PullRequestNumber }} {{ end }}packages/{{ .Manifest.Name }}
`)))

type CommitMessage struct {
	Manifest            fleetpkg.Manifest
	ECSVersion          string
	OldECSReference     string
	PipelineECSVersions []string
	PullRequestNumber   string
}

func (m CommitMessage) Build() (string, error) {
	buf := new(bytes.Buffer)
	if err := commitTmpl.Execute(buf, m); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "(devel)" {
		return "latest"
	}
	return info.Main.Version
}
