package unknownattribute

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
			Path: "testdata/fields.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/fields.yml", Line: 2, Col: 3},
					Category: "unknownattribute",
					Message:  `message contains an unknown attribute "typo"`,
				},
			},
		},
		{
			Path: "testdata/group.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos: analysis.Pos{
						File: string("testdata/group.yml"),
						Line: int(2),
						Col:  int(3),
					},
					Category: string("unknownattribute"),
					Message:  string("cloud contains an unknown attribute \"title\""),
					Related:  []analysis.RelatedInformation(nil),
				},
				{
					Pos: analysis.Pos{
						File: string("testdata/group.yml"),
						Line: int(2),
						Col:  int(3),
					},
					Category: string("unknownattribute"),
					Message:  string("cloud contains an unknown attribute \"group\""),
					Related:  []analysis.RelatedInformation(nil),
				},
				{
					Pos: analysis.Pos{
						File: string("testdata/group.yml"),
						Line: int(2),
						Col:  int(3),
					},
					Category: string("unknownattribute"),
					Message:  string("cloud contains an unknown attribute \"footnote\""),
					Related:  []analysis.RelatedInformation(nil),
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
