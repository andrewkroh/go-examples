package fleetpkg

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadIntegration(t *testing.T) {
	integ, err := Load("../../../ecs-update/testdata/my_package")
	require.NoError(t, err)

	data, err := json.MarshalIndent(integ, "", "  ")
	require.NoError(t, err)
	t.Logf("%s", data)
}
