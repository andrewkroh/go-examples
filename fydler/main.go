package main

import (
	"github.com/andrewkroh/go-examples/fydler/internal/analysis/conflict"
	"github.com/andrewkroh/go-examples/fydler/internal/analysis/duplicate"
	"github.com/andrewkroh/go-examples/fydler/internal/analysis/missingtype"
	"github.com/andrewkroh/go-examples/fydler/internal/analysis/nesting"
	"github.com/andrewkroh/go-examples/fydler/internal/analysis/unknownattribute"
	"github.com/andrewkroh/go-examples/fydler/internal/fydler"
)

func main() {
	fydler.Main(
		duplicate.Analyzer,
		missingtype.Analyzer,
		unknownattribute.Analyzer,
		conflict.Analyzer,
		nesting.Analyzer,
	)
}
