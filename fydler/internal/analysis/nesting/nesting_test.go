package nesting

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
			Path: "testdata/nesting.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/nesting.yml", Line: 2, Col: 3},
					Category: "nesting",
					Message:  "message is defined as a scalar type (match_only_text), but sub-fields were found",
					Related: []analysis.RelatedInformation{
						{Pos: analysis.Pos{File: "testdata/nesting.yml", Line: 4, Col: 3}, Message: "message.id is sub-field with type keyword"},
					},
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
