package fydler

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/go-examples/fydler/internal/analysis"
	"github.com/andrewkroh/go-examples/fydler/internal/printer"
)

var (
	outputTypes      stringListFlag
	diagnosticFilter stringListFlag
)

//nolint:revive // This is a pseudo main function so allow exits.
func Main(analyzers ...*analysis.Analyzer) {
	slices.SortFunc(analyzers, compareAnalyzer)

	progname := filepath.Base(os.Args[0])
	log.SetFlags(0)
	log.SetPrefix(progname + ": ")

	parseFlags(analyzers)

	if len(flag.Args()) == 0 {
		log.Fatal("Must pass a list of fields.yml files (e.g. **/fields/*.yml)")
	}

	files := make([]string, len(flag.Args()))
	copy(files, flag.Args())

	_, diags, err := Run(analyzers, files...)
	if err != nil {
		log.Fatal(err)
	}

	if len(diagnosticFilter) > 0 {
		diags = slices.DeleteFunc(diags, func(diag analysis.Diagnostic) bool {
			return !diagnosticContains(diagnosticFilter, &diag)
		})
	}

	for _, output := range outputTypes {
		switch output {
		case "color-text":
			err = printer.ColorText(diags, os.Stdout)
		case "text":
			err = printer.Text(diags, os.Stdout)
		case "json":
			err = printer.JSON(diags, os.Stdout)
		default:
			panic("invalid output type")
		}
		if err != nil {
			log.Fatal(err)
		}
	}
}

//nolint:revive // This is used by a pseudo main function so allow exits.
func parseFlags(analyzers []*analysis.Analyzer) {
	for _, a := range analyzers {
		prefix := a.Name + "."

		a.Flags.VisitAll(func(f *flag.Flag) {
			name := prefix + f.Name
			flag.Var(f.Value, name, f.Usage)
		})
	}

	flag.Var(&outputTypes, "set-output", "Output type to use. Allowed types are color-text, text, "+
		"and json. Defaults to color-text.")
	flag.Var(&diagnosticFilter, "i", "Include only diagnostics with a path containing this value. "+
		"If specified more than once, then diagnostics that match any value are included.")

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintln(out, "fydler [flags] fields_yml_file ...")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "fylder examines fields.yml files and reports issues that it finds,")
		fmt.Fprintln(out, "such as an unknown attribute, duplicate field definition, or")
		fmt.Fprintln(out, "conflicting type definition with another package.")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "fydler is normally invoked using a shell glob pattern to match")
		fmt.Fprintln(out, "the fields.yml files of interest.")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "  fydler packages/my_package/**/fields/*.yml")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "If you want fydler to consider all packages as context to the")
		fmt.Fprintln(out, "analyzers while only having interest in the results related to a")
		fmt.Fprintln(out, "particular path then you can use the include filter (-i).")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "  fydler -i '/my_package/' packages/**/fields/*.yml")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "The included analyzers are:")
		fmt.Fprintln(out, "")

		tw := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
		for _, a := range analyzers {
			fmt.Fprintf(tw, "  %s\t%s\n", a.Name, a.Description)
		}
		tw.Flush()
		fmt.Fprintln(out, "")

		fmt.Fprintln(out, "Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	for _, output := range outputTypes {
		switch output {
		case "color-text", "text", "json":
		default:
			log.Printf("invalid output type %q", output)
			flag.Usage()
			os.Exit(1)
		}
	}
	if len(outputTypes) == 0 {
		outputTypes = []string{"color-text"}
	}
}

func Run(analyzers []*analysis.Analyzer, files ...string) (results map[*analysis.Analyzer]any, diags []analysis.Diagnostic, err error) {
	slices.Sort(files)

	analyzers, err = dependencyOrder(analyzers)
	if err != nil {
		return nil, nil, err
	}

	fields, err := fleetpkg.ReadFields(files...)
	if err != nil {
		return nil, nil, err
	}
	slices.SortFunc(fields, compareFieldByFileMetadata)

	flat, err := fleetpkg.FlattenFields(fields)
	if err != nil {
		return nil, nil, err
	}
	slices.SortFunc(flat, compareFieldByFileMetadata)

	pass := &analysis.Pass{
		Fields: toPointerSlice(fields),
		Flat:   toPointerSlice(flat),
		Report: func(d analysis.Diagnostic) {
			diags = append(diags, d)
		},
	}
	results = map[*analysis.Analyzer]any{}

	for _, a := range analyzers {
		pass.Analyzer = a
		pass.ResultOf = map[*analysis.Analyzer]any{}
		for _, required := range a.Requires {
			pass.ResultOf[required] = results[required]
		}

		result, err := a.Run(pass)
		if err != nil {
			return nil, nil, fmt.Errorf("failed running %s analyzer: %w", a.Name, err)
		}
		results[a] = result
	}

	return results, diags, nil
}

func compareFieldByFileMetadata(a, b fleetpkg.Field) int {
	return compareFileMetadata(a.FileMetadata, b.FileMetadata)
}

func compareFileMetadata(a, b fleetpkg.FileMetadata) int {
	if c := cmp.Compare(a.Path(), b.Path()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Line(), b.Line()); c != 0 {
		return c
	}
	return cmp.Compare(a.Column(), b.Column())
}

func compareAnalyzer(a, b *analysis.Analyzer) int {
	return cmp.Compare(a.Name, b.Name)
}

func toPointerSlice[T any](in []T) []*T {
	out := make([]*T, len(in))
	for i := range in {
		out[i] = &in[i]
	}
	return out
}

func diagnosticContains(contains []string, diag *analysis.Diagnostic) bool {
	for _, c := range contains {
		if strings.Contains(diag.Pos.File, c) {
			return true
		}
		for _, related := range diag.Related {
			if strings.Contains(related.Pos.File, c) {
				return true
			}
		}
	}
	return false
}
