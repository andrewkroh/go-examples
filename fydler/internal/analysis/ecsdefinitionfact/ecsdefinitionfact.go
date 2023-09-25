package ecsdefinitionfact

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/andrewkroh/go-ecs"
	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/analysis/ecsversionfact"
)

var Analyzer = &analysis.Analyzer{
	Name:        "ecsdefinitionfact",
	Description: "Gathers the external ECS definition for fields.",
	Run:         run,
	Requires:    []*analysis.Analyzer{ecsversionfact.Analyzer},
}

type Fact struct {
	EnrichedFlat []*fleetpkg.Field // Field data enriched with external field data (type, description, pattern).
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Only log a Diagnostic once per directory.
	unknownECSVersion := map[string]struct{}{}
	ecsVersionsFact := pass.ResultOf[ecsversionfact.Analyzer].(*ecsversionfact.Fact)
	fact := &Fact{EnrichedFlat: make([]*fleetpkg.Field, 0, len(pass.Flat))}

	for _, f := range pass.Flat {
		if f.External != "ecs" {
			fact.EnrichedFlat = append(fact.EnrichedFlat, f)
			continue
		}

		// If the ecsVersion is not found, then the ecs.Lookup() will use data
		// from the latest ECS version. The ecsversionfact will have logged a
		// diagnostic about that problem.
		ecsVersion := ecsVersionsFact.ECSVersion(f.Path())

		dir := filepath.Dir(f.Path())
		ecsField, err := ecs.Lookup(f.Name, ecsVersion)
		if err != nil {
			switch {
			case errors.Is(err, ecs.ErrFieldNotFound):
				pass.Report(analysis.Diagnostic{
					Pos:      analysis.NewPos(f.FileMetadata),
					Category: pass.Analyzer.Name,
					Message:  fmt.Sprintf("%s is declared with 'external: ecs' but this field does not exist in ECS version %q", f.Name, ecsVersion),
				})
			case errors.Is(err, ecs.ErrVersionNotFound):
				if _, found := unknownECSVersion[dir]; !found {
					unknownECSVersion[dir] = struct{}{}
					pass.Report(analysis.Diagnostic{
						Pos:      analysis.NewPos(f.FileMetadata),
						Category: pass.Analyzer.Name,
						Message:  fmt.Sprintf("%s is declared with 'external: ecs' using ECS version %q, but this version is unknown this tool", f.Name, ecsVersion),
					})
				}
			case errors.Is(err, ecs.ErrInvalidVersion):
				if _, found := unknownECSVersion[dir]; !found {
					unknownECSVersion[dir] = struct{}{}
					pass.Report(analysis.Diagnostic{
						Pos:      analysis.NewPos(f.FileMetadata),
						Category: pass.Analyzer.Name,
						Message:  fmt.Sprintf("%s is declared with 'external: ecs' using ECS version %q, but that is an invalid version (%s)", f.Name, ecsVersion, err),
					})
				}
			default:
				return nil, fmt.Errorf("failed looking up ECS definition of %q from %s:%d:%d using version %q: %w",
					f.Name, f.Path(), f.Line(), f.Column(), ecsVersion, err)
			}

			continue
		}

		// Copy-on-write.
		{
			tmp := *f
			f = &tmp
		}
		f.Type = ecsField.DataType
		f.Pattern = ecsField.Pattern
		f.Description = ecsField.Description

		fact.EnrichedFlat = append(fact.EnrichedFlat, f)
	}

	return fact, nil
}
