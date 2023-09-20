package duplicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/fydler"
)

func Test(t *testing.T) {
	_, diags, err := fydler.Run([]*analysis.Analyzer{Analyzer}, "testdata/fields.yml")
	if err != nil {
		t.Fatal(err)
	}

	require.Len(t, diags, 1)

	d := diags[0]
	assert.Equal(t, "duplicate", d.Category)
	assert.Equal(t, "message is declared 2 times", d.Message)
	assert.Equal(t, "testdata/fields.yml", d.Pos.File)
	assert.Equal(t, 2, d.Pos.Line)
	assert.Equal(t, 3, d.Pos.Col)
	assert.Len(t, d.Related, 1)
}
