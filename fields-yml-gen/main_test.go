package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFields(t *testing.T) {
	t.Run("elastic-package.out", func(t *testing.T) {
		f, err := os.Open("testdata/elastic-package.out")
		require.NoError(t, err)
		defer f.Close()

		out, err := readFields(f)
		require.NoError(t, err)

		assert.Equal(t, []string{
			"event.action",
			"event.agent_id_status",
			"event.category",
			"event.created",
			"event.id",
			"event.ingested",
			"event.kind",
			"event.original",
			"event.outcome",
			"event.timezone",
			"event.type",
		}, out)
	})

	t.Run("fields.txt", func(t *testing.T) {
		f, err := os.Open("testdata/fields.txt")
		require.NoError(t, err)
		defer f.Close()

		out, err := readFields(f)
		require.NoError(t, err)

		assert.Equal(t, []string{
			"url.original",
			"network.iana_number",
			"network.packets",
			"not.in.ecs.foobar",
			"foo.bar",
		}, out)
	})
}
