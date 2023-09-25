package nesting

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/analysis/ecsdefinitionfact"
)

var Analyzer = &analysis.Analyzer{
	Name:        "nesting",
	Description: "Detect fields that are nested below a scalar type field.",
	Run:         run,
	Requires:    []*analysis.Analyzer{ecsdefinitionfact.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	ecsDefinitionFact := pass.ResultOf[ecsdefinitionfact.Analyzer].(*ecsdefinitionfact.Fact)

	// Build map of parent field name to child field.
	parentChildRelations := map[string][]*fleetpkg.Field{}
	for _, f := range ecsDefinitionFact.EnrichedFlat {
		if idx := strings.LastIndexByte(f.Name, '.'); idx != -1 {
			parentName := f.Name[:idx]

			slice := parentChildRelations[parentName]
			slice = append(slice, f)
			parentChildRelations[parentName] = slice
		}
	}

	for _, f := range ecsDefinitionFact.EnrichedFlat {
		// Skip non-scalar field types.
		switch f.Type {
		case "group", "object", "nested", "array":
			continue
		}

		// Check if this scalar field has any children.
		children, found := parentChildRelations[f.Name]
		if !found {
			continue
		}

		pass.Report(makeDiag(f, children))
	}

	return nil, nil
}

func makeDiag(parent *fleetpkg.Field, children []*fleetpkg.Field) analysis.Diagnostic {
	diag := analysis.Diagnostic{
		Pos:      analysis.NewPos(parent.FileMetadata),
		Category: "nesting",
		Message:  fmt.Sprintf("%s is defined as a scalar type (%s), but sub-fields were found", parent.Name, parent.Type),
		Related:  make([]analysis.RelatedInformation, 0, len(children)),
	}

	// Sort the children for determinism.
	slices.SortFunc(children, compareFieldByFileMetadata)

	for _, f := range children {
		diag.Related = append(diag.Related, analysis.RelatedInformation{
			Pos:     analysis.NewPos(f.FileMetadata),
			Message: f.Name + " is sub-field with type " + f.Type,
		})
	}
	return diag
}

func compareFieldByFileMetadata(a, b *fleetpkg.Field) int {
	if c := cmp.Compare(a.Path(), b.Path()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Line(), b.Line()); c != 0 {
		return c
	}
	return cmp.Compare(a.Column(), b.Column())
}
