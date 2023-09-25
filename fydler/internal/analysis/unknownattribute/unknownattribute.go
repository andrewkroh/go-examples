package unknownattribute

import (
	"fmt"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "unknownattribute",
	Description: "Detect unknown field attributes.",
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Fields {
		// 'description' on field groups is never used by anything in Fleet.
		if f.Type == "group" && f.Description != "" {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("%s field group contains a 'description', but this is unused by Fleet and can be removed.", f.Name),
			})
		}

		for attrName := range f.AdditionalAttributes {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("%s contains an unknown attribute %q", f.Name, attrName),
			})
		}
	}
	return nil, nil
}
