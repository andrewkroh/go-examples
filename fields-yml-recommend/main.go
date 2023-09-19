package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/andrewkroh/go-ecs"
	"github.com/andrewkroh/go-fleetpkg"
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

	flag.StringVar(&format, "f", "list", "Output format (list or json). Defaults to list.")
	flag.BoolVar(&warn, "w", true, "Warn on invalid external ECS field references.")
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

	fields, err := fleetpkg.ReadFields(flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}

	flat, err := fleetpkg.FlattenFields(fields)
	if err != nil {
		log.Fatal(err)
	}

	var recommendations []Recommendation
	for _, f := range flat {
		if f.External == "ecs" {
			// Only analyze non-ECS fields.
			continue
		}

		if ecsField, _ := ecs.Lookup(f.Name, ""); ecsField != nil {
			recommendations = append(recommendations, Recommendation{
				Name:  f.Name,
				Types: f.Type,
				File:  f.Path(),
				Line:  f.Line(),
				Notes: []string{"Use 'external: ecs' to import the ECS definition."},
			})
		}
	}

	// Duplicates definitions by name.
	{
		fieldSet := map[string][]*fleetpkg.Field{}
		for i := range flat {
			f := &flat[i]
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
						File:  f.Path(),
						Line:  f.Line(),
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

func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
