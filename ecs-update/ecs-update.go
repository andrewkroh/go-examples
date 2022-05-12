package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

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
)

func init() {
	flag.StringVar(&ecsVersion, "ecs-version", "", "ECS version (e.g. 8.3.0)")
	flag.StringVar(&pullRequestNumber, "pr", "", "Pull request number")
}

func main() {
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

var versionReg = regexp.MustCompile(`(?m)(\d+\.\d+)\.\d+`)

func updatePackage(path, ecsVersion string) error {
	pkg, err := fleetpkg.ReadPackage(path)
	if err != nil {
		return err
	}

	if pkg.BuildManifest != nil {
		ver, err := semver.NewVersion(ecsVersion)
		if err != nil {
			return err
		}

		branchName := fmt.Sprintf("%d.%d", ver.Major, ver.Minor)

		_, err = pkg.BuildManifest.SetBuildManifestECSReference("git@" + branchName)
		if err != nil {
			return err
		}
	}

	for _, ds := range pkg.DataStreams {
		if ds.DefaultPipeline != nil {
			_, err := ds.DefaultPipeline.SetIngestNodePipelineECSVersion(ecsVersion)
			if err != nil {
				return err
			}
		}

		if ds.SampleEvent != nil {
			_, err := ds.SampleEvent.SetSampleEventECSVersion(ecsVersion)
			if err != nil {
				return err
			}
		}
	}

	err = WriteDocument(pkg.BuildManifest.FilePath, pkg.BuildManifest.WriteYAML)
	for _, ds := range pkg.DataStreams {
		if ds.DefaultPipeline != nil {
			err = multierr.Append(err, WriteDocument(ds.DefaultPipeline.FilePath, ds.DefaultPipeline.WriteYAML))
		}
		if ds.SampleEvent != nil {
			err = multierr.Append(err, WriteDocument(ds.SampleEvent.FilePath, func(w io.Writer) error { return ds.SampleEvent.WriteJSON(w, 4) }))
		}
	}

	if err != nil {
		return err
	}

	if err = PurgeWhitespaceChanges(path); err != nil {
		return err
	}

	if err = BuildAndUpdate(path); err != nil {
		return err
	}

	if err = AddChangelog(path, pullRequestNumber, fmt.Sprintf("Update package to ECS %s.", ecsVersion)); err != nil {
		return err
	}

	commitMessage := fmt.Sprintf(`%s - update to ECS %s

[git-generate]
ecs-update -ecs-version=8.3.0 -pr=%s packages/%s
`, pkg.Manifest.OriginalData.Name, ecsVersion,
		pullRequestNumber, pkg.Manifest.OriginalData.Name)

	if err = Commit(path, commitMessage); err != nil {
		return err
	}

	return nil
}

func WriteDocument(path string, encode func(io.Writer) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return encode(f)
}

func PurgeWhitespaceChanges(path string) error {
	return ExecutePlan(path, []string{
		// Stage non-whitespace changes.
		`git diff -U0 -w --no-color --ignore-blank-lines | git apply --cached --ignore-whitespace --unidiff-zero -`,

		// Purge whitespace
		`git checkout -- .`,
	})
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
