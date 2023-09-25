package ecsdefinitionfact

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/fydler"
)

func Test(t *testing.T) {
	testCases := []struct {
		Path   string
		Diags  []analysis.Diagnostic
		Fields map[string]string
	}{
		{
			Path: "../ecsversionfact/testdata/my_package/data_stream/foo/fields/fields.yml",
			Fields: map[string]string{
				"book":    "keyword",
				"message": "match_only_text",
				"labels":  "object",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(filepath.Base(tc.Path), func(t *testing.T) {
			results, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, tc.Path)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.Diags, diags)

			fact := results[Analyzer].(*Fact)
			require.Len(t, fact.EnrichedFlat, len(tc.Fields), "unexpected EnrichedFlat length")
			for _, f := range fact.EnrichedFlat {
				assert.Equal(t, tc.Fields[f.Name], f.Type)
			}
		})
	}
}
