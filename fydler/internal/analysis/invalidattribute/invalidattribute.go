package invalidattribute

import (
	"fmt"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "invalidattribute",
	Description: "Detect invalid usages of field attributes.",
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Fields {
		// 'description' on field groups is never used by anything in Fleet.
		if f.Type == "group" && f.Description != "" {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("%s field group contains a 'description', but this is unused by Fleet and can be removed", f.Name),
			})
		}

		// It is invalid to specify a 'type' when an external definition is used.
		if f.Type != "" && f.External != "" {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("%s use 'external: %s', therefore 'type' should not be specified", f.Name, f.External),
			})
		}
	}
	return nil, nil
}
