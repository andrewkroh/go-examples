package useecs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/fydler"
)

func Test(t *testing.T) {
	testCases := []struct {
		Path                  string
		Diags                 []analysis.Diagnostic
		IgnoreConstantKeyword bool
	}{
		{
			Path: "testdata/fields.yml",
			Diags: []analysis.Diagnostic{
				{
					Pos:      analysis.Pos{File: "testdata/fields.yml", Line: 2, Col: 3},
					Category: "useecs",
					Message:  "event.dataset exists in ECS, but the definition is not using 'external: ecs'. The ECS type is keyword, but this uses constant_keyword",
				},
			},
		},
		{
			Path:                  "testdata/fields.yml",
			IgnoreConstantKeyword: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(filepath.Base(tc.Path), func(t *testing.T) {
			ignoreConstantKeyword = tc.IgnoreConstantKeyword

			_, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, tc.Path)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Diags, diags)
		})
	}
}
