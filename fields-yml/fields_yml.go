package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/andrewkroh/go-ecs"
	"github.com/andrewkroh/go-fleetpkg"
)

// Flags
var (
	format string // Output format (list, json).
	warn   bool   // Warn on invalid ECS field references.
)

func init() {
	flag.StringVar(&format, "f", "list", "Output format (list or json). Defaults to list.")
	flag.BoolVar(&warn, "w", true, "Warn on invalid external ECS field references.")
}

type fieldWithPath struct {
	fleetpkg.Field
	Source struct {
		Path   string `json:"path"`
		Line   int    `json:"line"`
		Column int    `json:"column"`
	} `json:"source"`
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

	// Convert to a struct that exports the source metadata.
	fieldsWithSource := make([]fieldWithPath, len(flat))
	for i := range flat {
		fieldsWithSource[i].Field = flat[i]
		fieldsWithSource[i].Source.Path = flat[i].FileMetadata.Path()
		fieldsWithSource[i].Source.Line = flat[i].FileMetadata.Line()
		fieldsWithSource[i].Source.Column = flat[i].FileMetadata.Column()
	}

	// Resolve ECS fields.
	for i, f := range fieldsWithSource {
		if f.External != "ecs" {
			continue
		}

		// Lookup definition in the latest ECS version from go-ecs.
		ecsField, err := ecs.Lookup(f.Name, "")
		if err != nil {
			if warn {
				log.Printf("WARN: %q in %s:%d does not exist is ECS or is not a leaf field.", f.Name, f.FileMetadata.Path(), f.FileMetadata.Line())
			}
			continue
		}

		fieldsWithSource[i].Type = ecsField.DataType
		fieldsWithSource[i].Description = ecsField.Description
		fieldsWithSource[i].Pattern = ecsField.Pattern
	}

	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err = enc.Encode(fieldsWithSource); err != nil {
			log.Fatal(err)
		}
		return
	case "list":
		for _, f := range fieldsWithSource {
			fmt.Println(f.Name)
		}
	default:
		log.Fatalf("Unknown output format: %q", format)
	}
}
