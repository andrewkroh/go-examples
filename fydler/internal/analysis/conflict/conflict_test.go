package conflict

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/fydler"
)

func Test(t *testing.T) {
	testCases := []struct {
		Path             string
		Diags            []analysis.Diagnostic
		IgnoreTextFam    bool
		IgnoreKeywordFam bool
	}{
		{
			Path: "testdata/conflict.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/conflict.yml", Line: 2, Col: 3},
					Category: "conflict",
					Message:  "number has multiple data types (long, short)",
					Related: []analysis.RelatedInformation{
						{Pos: analysis.Pos{File: "testdata/conflict.yml", Line: 2, Col: 3}, Message: "long"},
						{Pos: analysis.Pos{File: "testdata/conflict.yml", Line: 4, Col: 3}, Message: "short"},
					},
				},
			},
		},
		{
			Path: "testdata/keyword_conflict.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/keyword_conflict.yml", Line: 4, Col: 3},
					Category: "conflict",
					Message:  "id has multiple data types (constant_keyword, keyword, wildcard)",
					Related: []analysis.RelatedInformation{
						{Pos: analysis.Pos{File: "testdata/keyword_conflict.yml", Line: 4, Col: 3}, Message: "constant_keyword"},
						{Pos: analysis.Pos{File: "testdata/keyword_conflict.yml", Line: 2, Col: 3}, Message: "keyword"},
						{Pos: analysis.Pos{File: "testdata/keyword_conflict.yml", Line: 6, Col: 3}, Message: "wildcard"},
					},
				},
			},
		},
		{
			Path:             "testdata/keyword_conflict.yml",
			IgnoreKeywordFam: true,
		},
		{
			Path: "testdata/text_conflict.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/text_conflict.yml", Line: 4, Col: 3},
					Category: "conflict",
					Message:  "abstract has multiple data types (match_only_text, text)",
					Related: []analysis.RelatedInformation{
						{Pos: analysis.Pos{File: "testdata/text_conflict.yml", Line: 4, Col: 3}, Message: "match_only_text"},
						{Pos: analysis.Pos{File: "testdata/text_conflict.yml", Line: 2, Col: 3}, Message: "text"},
					},
				},
			},
		},
		{
			Path:          "testdata/text_conflict.yml",
			IgnoreTextFam: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(filepath.Base(tc.Path), func(t *testing.T) {
			ignoreKeywordFamilyConflicts = tc.IgnoreKeywordFam
			ignoreTextFamilyConflicts = tc.IgnoreTextFam

			_, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, tc.Path)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Diags, diags)
		})
	}
}
