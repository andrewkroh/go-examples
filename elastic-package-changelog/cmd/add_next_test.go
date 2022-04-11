package cmd

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddNext(t *testing.T) {
	inData, err := ioutil.ReadFile("../testdata/changelog.yml")
	require.NoError(t, err)

	outData := new(bytes.Buffer)

	cmd := newRootCommand()
	cmd.SetArgs([]string{"add-next", "--changelog", "-", "--pr=1111", "-d", "Hello", "--type=en", "--manifest="})
	cmd.SetIn(bytes.NewReader(inData))
	cmd.SetOut(outData)

	err = cmd.Execute()
	require.NoError(t, err)

	assert.Contains(t, outData.String(),
		"# newer versions go on top",
		"- version: 1.3.0",
		"- version: 1.2.1")
}
