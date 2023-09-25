package ecsversionfact

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrewkroh/go-fleetpkg"
	"gopkg.in/yaml.v3"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "ecsversionfact",
	Description: "Gathers the ECS version associated with fields.",
	Run:         run,
}

type Fact struct {
	dirToECSVersion map[string]string
}

// ECSVersion returns the ECS version associated to a given fields.yml file.
func (f *Fact) ECSVersion(path string) string {
	return f.dirToECSVersion[filepath.Dir(path)]
}

func run(pass *analysis.Pass) (interface{}, error) {
	dirToECSVersion := map[string]string{}
	notExist := map[string]struct{}{}

	for _, f := range pass.Flat {
		if f.External != "ecs" {
			continue
		}

		dir := filepath.Dir(f.Path())
		if _, found := dirToECSVersion[dir]; found {
			continue
		}

		if _, found := notExist[dir]; found {
			continue
		}

		ecsRef, err := lookupECSReference(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				pass.Report(analysis.Diagnostic{
					Pos:      analysis.NewPos(f.FileMetadata),
					Category: pass.Analyzer.Name,
					Message:  "missing ecs version reference because build.yml not found",
				})
				notExist[dir] = struct{}{}
				continue
			}
			return nil, fmt.Errorf("failed to read ecs version: %w", err)
		}

		if ecsRef == "" {
			notExist[dir] = struct{}{}
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  "missing ecs version reference in build.yml",
			})
			continue
		}

		dirToECSVersion[dir] = ecsRef
	}

	return &Fact{dirToECSVersion: dirToECSVersion}, nil
}

func lookupECSReference(dir string) (string, error) {
	f, err := openBuildManifest(dir)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var manifest fleetpkg.BuildManifest
	dec := yaml.NewDecoder(f)
	if err = dec.Decode(&manifest); err != nil {
		return "", fmt.Errorf("failed to unmarshal %s: %w", f.Name(), err)
	}

	// Strip prefix from git@v1.2.3.
	gitRef := manifest.Dependencies.ECS.Reference
	gitRef = strings.TrimPrefix(gitRef, "git@")
	return gitRef, nil
}

// searchPaths contains relative paths from a fields.yml file to
// a package build.yml file.
var searchPaths = []string{
	// Integration data stream fields.
	"../../../_dev/build/build.yml",
	// Input package fields.
	"../_dev/build/build.yml",
	// Transform fields.
	"../../../../_dev/build/build.yml",
}

func openBuildManifest(dir string) (*os.File, error) {
	for _, searchPath := range searchPaths {
		f, err := os.Open(filepath.Join(dir, searchPath))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}

		return f, nil
	}

	return nil, fs.ErrNotExist
}
