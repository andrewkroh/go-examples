package dynamicfield

import (
	"fmt"
	"strings"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "dynamicfield",
	Description: "Detect issues with wildcard fields meant to be dynamic mappings.",
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Flat {
		if !strings.Contains(f.Name, "*") {
			continue
		}

		// This is always an error. Real field names should never contain an asterisk ('*').
		// Without 'object_type' fleet creates a static mapping for a field whose literal
		// name includes '*' (e.g. 'tags.*').
		if f.Type == "object" && f.ObjectType == "" {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message: fmt.Sprintf("%s field is meant to be a dynamic mapping, but is missing an 'object_type' "+
					"so it will never be a dynamic mapping", f.Name),
			})
			continue
		}

		if f.Type == "" {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message: fmt.Sprintf("%s field is meant to be a dynamic mapping, but does not specify a 'type' "+
					"so it will never be a dynamic mapping", f.Name),
			})
			continue
		}
	}
	return nil, nil
}
