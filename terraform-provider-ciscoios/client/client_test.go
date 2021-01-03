package client_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client"
	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client/file"
	"github.com/stretchr/testify/require"
)

func TestClient_ACLs(t *testing.T) {
	cmdr, err := file.NewClient(fileURI("./testdata/1_acl.txt"))
	require.NoError(t, err)
	defer cmdr.Close()

	cl, err := client.New(cmdr)
	require.NoError(t, err)
	defer cl.Close()

	acls, err := cl.ACLs()
	require.NoError(t, err)
	if assert.Len(t, acls, 2) {
		assert.Len(t, acls[0].Rules, 31)
		assert.Len(t, acls[1].Rules, 17)
	}
}

func fileURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	return "file://" + abs
}
