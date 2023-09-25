package duplicate

import (
	"fmt"
	"path/filepath"

	"github.com/andrewkroh/go-fleetpkg"
	"golang.org/x/exp/maps"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name:        "duplicate",
	Description: "Detect duplicate field declarations within a directory.",
	Run:         run,
}

func run(pass *analysis.Pass) (any, error) {
	seen := map[string][]*fleetpkg.Field{}
	var currentDir string

	flush := func() {
		for name, seenFields := range seen {
			if len(seenFields) < 2 {
				continue
			}

			field := seenFields[0]
			diag := analysis.Diagnostic{
				Pos:      analysis.NewPos(field.FileMetadata),
				Category: pass.Analyzer.Name,
				Message:  fmt.Sprintf("%s is declared %d times", name, len(seenFields)),
			}

			for _, f := range seenFields[1:] {
				diag.Related = append(diag.Related, analysis.RelatedInformation{
					Pos:     analysis.NewPos(f.FileMetadata),
					Message: "additional definition",
				})
			}

			pass.Report(diag)
		}
	}
	for _, f := range pass.Flat {
		// When the directory changes flush the duplicates.
		if dir := filepath.Dir(f.Path()); currentDir != dir {
			// Reset
			flush()
			maps.Clear(seen)
			currentDir = dir
		}

		seenFields := seen[f.Name]
		seenFields = append(seenFields, f)
		seen[f.Name] = seenFields
	}

	flush()
	return nil, nil
}
