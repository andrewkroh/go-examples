package invalidattribute

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/fydler"
)

func Test(t *testing.T) {
	testCases := []struct {
		Path  string
		Diags []analysis.Diagnostic
	}{
		{
			Path: "testdata/group_description.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/group_description.yml", Line: 2, Col: 3},
					Category: "invalidattribute",
					Message:  "cloud field group contains a 'description', but this is unused by Fleet and can be removed",
				},
			},
		},
		{
			Path: "testdata/type_with_external.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/type_with_external.yml", Line: 2, Col: 3},
					Category: "invalidattribute",
					Message:  "message use 'external: ecs', therefore 'type' should not be specified",
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(filepath.Base(tc.Path), func(t *testing.T) {
			_, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, tc.Path)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Diags, diags)
		})
	}
}
