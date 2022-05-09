package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/andrewkroh/go-examples/fields-yml-gen/ecs"
	"github.com/andrewkroh/go-examples/fields-yml/fieldsyml"
)

var usage = `
fields-yml-recommend advises you on changes to fields YAML fields. It
is recommended that you point it at all fields files within a single data stream
so that it has the full list of fields. It detects multiple issues:

- Fields that exist in ECS, but are not using an 'external: ecs' definition.
- Fields that are duplicated.

`[1:]

// Flags
var (
	format string // Output format (list, json).
	warn   bool   // Warn on invalid ECS field references.
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage+"Usage of %s:\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.StringVar(&format, "f", "json", "Output format (list or json). Defaults to list.")
	flag.BoolVar(&warn, "w", true, "Warn on invalid external ECS field references.")
}

func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

type Recommendation struct {
	Name  string   `json:"name"`
	Types string   `json:"type"`
	File  string   `json:"file"`
	Line  int      `json:"line"`
	Notes []string `json:"notes"`
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("Must pass a list of fields.yml files.")
	}

	fields, err := fieldsyml.ReadFieldsYAML(flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}

	flat, err := fieldsyml.FlattenFields(fields)
	if err != nil {
		log.Fatal(err)
	}

	var recommendations []Recommendation

	// ECS fields that could be used.
	for _, f := range flat {
		if f.External != "" {
			// Only analyze non-external fields.
			continue
		}

		if ecsField := ecs.GetField(f.Name); ecsField != nil {
			recommendations = append(recommendations, Recommendation{
				Name:  f.Name,
				Types: f.Type,
				File:  f.Source,
				Line:  f.SourceLine,
				Notes: []string{"Use 'external: ecs' to import the ECS definition."},
			})
		}
	}

	// Duplicates definitions by name.
	{
		fieldSet := map[string][]fieldsyml.FlatField{}
		for _, f := range flat {
			list := fieldSet[f.Name]
			list = append(list, f)
			fieldSet[f.Name] = list
		}

		for _, fields := range fieldSet {
			if len(fields) > 1 {
				for _, f := range fields {
					recommendations = append(recommendations, Recommendation{
						Name:  f.Name,
						Types: f.Type,
						File:  f.Source,
						Line:  f.SourceLine,
						Notes: []string{
							fmt.Sprintf("Duplicate field (%d times).", len(fields)),
						},
					})
				}
			}
		}
	}

	if len(recommendations) == 0 {
		log.Println("No recommendation.")
		return
	}

	switch format {
	case "json":
		if err = outputJSON(recommendations); err != nil {
			log.Fatal(err)
		}
		return
	case "list":
		for _, r := range recommendations {
			fmt.Printf("%s:%d - %s : %v\n", r.File, r.Line, r.Name, r.Notes)
		}
		return
	default:
		log.Fatalf("Unknown output format: %q", format)
	}
}
