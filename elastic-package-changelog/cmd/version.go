package cmd

import (
	"io"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var version = "DEV"

func init() {
	if version == "DEV" {
		// Fall back to Go module data when not built with goreleaser.
		if info, ok := debug.ReadBuildInfo(); ok {
			version = info.Main.Version
		}
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Output version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		io.WriteString(cmd.OutOrStdout(), name+" "+version+"\n")
	},
}
