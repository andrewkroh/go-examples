package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/andrewkroh/go-examples/fields-yml-gen/ecs"
	"github.com/andrewkroh/go-examples/fields-yml/fieldsyml"
)

// Flags
var (
	format string // Output format (list, json).
	warn   bool   // Warn on invalid ECS field references.
)

func init() {
	flag.StringVar(&format, "f", "json", "Output format (list or json). Defaults to json.")
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
	for _, f := range flat {
		if f.External != "" {
			// Only analyze non-external fields.
			continue
		}

		r := Recommendation{
			Name:  f.Name,
			Types: f.Type,
			File:  f.Source,
			Line:  f.SourceLine,
		}

		if ecsField := ecs.GetField(f.Name); ecsField != nil {
			r.Notes = append(r.Notes, "Use 'external: ecs' to import the ECS definition.")
		}

		if len(r.Notes) > 0 {
			recommendations = append(recommendations, r)
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
