package conflict

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

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
		if f.Type == "" {
			continue
		}

		if currentKey != f.Name {
			flush()
			currentKey = f.Name
		}

		fields = append(fields, f)
	}
	flush()

	return nil, nil
}

func makeDiag(conflicts []*fleetpkg.Field, dataTypes []string) *analysis.Diagnostic {
	if ignoreTextFamilyConflicts && isTextTypeFamilyConflict(dataTypes) {
		return nil
	}
	if ignoreKeywordFamilyConflicts && isKeywordTypeFamilyConflict(dataTypes) {
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

func isTextTypeFamilyConflict(dataTypes []string) bool {
	for _, typ := range dataTypes {
		switch typ {
		case "match_only_text", "text":
		default:
			return false
		}
	}
	return true
}

func isKeywordTypeFamilyConflict(dataTypes []string) bool {
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
