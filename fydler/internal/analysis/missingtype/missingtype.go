package missingtype

import (
	"fmt"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "missingtype",
	Description: "Detect fields declared without a 'type'.",
	Run:         run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Flat {
		if f.Type == "" && f.External == "" {
			pass.Report(analysis.Diagnostic{
				Pos:      analysis.NewPos(f.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("%s is missing a 'type'", f.Name),
			})
		}
	}
	return nil, nil
}
