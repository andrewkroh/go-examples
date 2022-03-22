package fieldsyml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseYAML(t *testing.T) {
	fields, err := ReadFieldsYAML("testdata/*.yml")
	require.NoError(t, err)

	flat, err := FlattenFields(fields)
	require.NoError(t, err)
	for _, f := range flat {
		t.Log(f)
	}
}
