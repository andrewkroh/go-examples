package conflict

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/andrewkroh/go-ecs"
	"github.com/andrewkroh/go-fleetpkg"
	"golang.org/x/exp/maps"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/analysis/ecsdefinitionfact"
)

var Analyzer = &analysis.Analyzer{
	Name:        "conflict",
	Description: "Detect conflicting field data types across declarations of fields with the same name.",
	Run:         run,
	Requires:    []*analysis.Analyzer{ecsdefinitionfact.Analyzer},
}

var (
	// Isolate comparison to fields from the same data stream type (e.g. logs, metrics).
	// You might want to allow conflict between a fields in logs-* and metrics-*. Setting
	// this flag will ignore those conflicts.
	dataStreamTypeIsolation      bool
	ignoreTextFamilyConflicts    bool
	ignoreKeywordFamilyConflicts bool
)

func init() {
	Analyzer.Flags.BoolVar(&dataStreamTypeIsolation, "data-stream-type-isolation", false, "Isolate comparison to like data stream types (i.e. don't compare metrics-* fields to logs-*) NOT IMPLEMENTED")
	Analyzer.Flags.BoolVar(&ignoreTextFamilyConflicts, "ignore-text-family", false, "Ignore text type family conflicts (text and match_only_text type definitions are allowed).")
	Analyzer.Flags.BoolVar(&ignoreKeywordFamilyConflicts, "ignore-keyword-family", false, "Ignore text type family conflicts (keyword, constant_keyword, and wildcard type definitions are allowed).")
}

func run(pass *analysis.Pass) (interface{}, error) {
	if err := nonExternalConflicts(pass); err != nil {
		return nil, err
	}

	if err := externalECSConflicts(pass); err != nil {
		return nil, err
	}

	return nil, nil
}

func makeDiag(conflicts []*fleetpkg.Field, dataTypes []string) *analysis.Diagnostic {
	if ignoreTextFamilyConflicts && isTextTypeFamilyConflict(dataTypes...) {
		return nil
	}
	if ignoreKeywordFamilyConflicts && isKeywordTypeFamilyConflict(dataTypes...) {
		return nil
	}

	// For determinism.
	slices.Sort(dataTypes)

	f := conflicts[0]
	diag := &analysis.Diagnostic{
		Pos:      analysis.NewPos(f.FileMetadata),
		Category: "conflict",
		Message:  fmt.Sprintf("%s has multiple data types (%s)", f.Name, strings.Join(dataTypes, ", ")),
		Related:  make([]analysis.RelatedInformation, 0, len(conflicts)),
	}
	for _, f := range conflicts {
		diag.Related = append(diag.Related, analysis.RelatedInformation{
			Pos:     analysis.NewPos(f.FileMetadata),
			Message: f.Type,
		})
	}
	return diag
}

// nonExternalConflicts reports conflicts between non-externally defined fields with
// the same name, but different data types.
func nonExternalConflicts(pass *analysis.Pass) error {
	ecsDefinitionFact := pass.ResultOf[ecsdefinitionfact.Analyzer].(*ecsdefinitionfact.Fact)

	// Sort by name and type.
	slices.SortStableFunc(ecsDefinitionFact.EnrichedFlat, compareFieldByNameAndType)

	var currentKey string
	var fields []*fleetpkg.Field
	dataTypes := map[string]struct{}{}
	flush := func() {
		// Aggregate types.
		for _, f := range fields {
			dataTypes[f.Type] = struct{}{}
		}
		if len(dataTypes) > 1 {
			if diag := makeDiag(fields, maps.Keys(dataTypes)); diag != nil {
				pass.Report(*diag)
			}
		}

		// Reset
		fields = fields[:0]
		maps.Clear(dataTypes)
	}

	for _, f := range ecsDefinitionFact.EnrichedFlat {
		// The field must have a type to be considered in conflict with another field.
		if f.Type == "" {
			continue
		}

		// Ignore externally defined fields. We'll assume that if a field is using
		// ECS that it cannot be in conflict.
		if f.External == "ecs" {
			continue
		}

		if currentKey != f.Name {
			flush()
			currentKey = f.Name
		}

		fields = append(fields, f)
	}
	flush()

	return nil
}

// externalECSConflicts reports conflicts between a field's data type and the ECS
// data type if that field exists in ECS.
func externalECSConflicts(pass *analysis.Pass) error {
	// Find conflicts with ECS.
	for _, f := range pass.Flat {
		// The field must have a type to be considered in conflict with an external source.
		if f.Type == "" {
			continue
		}

		// Skip fields that are already referencing ECS.
		if f.External == "ecs" {
			continue
		}

		ecsField, err := ecs.Lookup(f.Name, "")
		if err != nil {
			if errors.Is(err, ecs.ErrFieldNotFound) {
				continue
			}
			return err
		}

		if f.Type == ecsField.DataType {
			continue
		}
		if ignoreTextFamilyConflicts && isTextTypeFamilyConflict(f.Type, ecsField.DataType) {
			continue
		}
		if ignoreKeywordFamilyConflicts && isKeywordTypeFamilyConflict(f.Type, ecsField.DataType) {
			continue
		}
		pass.Report(analysis.Diagnostic{
			Pos:      analysis.NewPos(f.FileMetadata),
			Category: pass.Analyzer.Name,
			Message:  fmt.Sprintf("%s field declared as type %s conflicts with the ECS data type %s", f.Name, f.Type, ecsField.DataType),
		})
	}

	return nil
}

func isTextTypeFamilyConflict(dataTypes ...string) bool {
	for _, typ := range dataTypes {
		switch typ {
		case "match_only_text", "text":
		default:
			return false
		}
	}
	return true
}

func isKeywordTypeFamilyConflict(dataTypes ...string) bool {
	for _, typ := range dataTypes {
		switch typ {
		case "keyword", "constant_keyword", "wildcard":
		default:
			return false
		}
	}
	return true
}

func compareFieldByNameAndType(a, b *fleetpkg.Field) int {
	if c := cmp.Compare(a.Name, b.Name); c != 0 {
		return c
	}
	return cmp.Compare(a.Type, b.Type)
}
