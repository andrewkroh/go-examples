package fleetpkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadIntegration(t *testing.T) {
	i, err := LoadIntegration("/Users/akroh/code/elastic/integrations/packages/github")
	require.NoError(t, err)

	t.Log(i)
}
