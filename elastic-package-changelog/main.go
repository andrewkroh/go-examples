package main

import (
	"os"

	"github.com/andrewkroh/go-examples/elastic-package-changelog/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
