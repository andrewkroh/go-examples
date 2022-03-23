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
	flag.StringVar(&format, "f", "list", "Output format (list or json). Defaults to list.")
	flag.BoolVar(&warn, "w", true, "Warn on invalid external ECS field references.")
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

	flat, hasUnresolved := fieldsyml.ResolveECSReferences(flat)
	if hasUnresolved && warn {
		for _, f := range flat {
			if f.External == "ecs" && f.Type == "" {
				log.Printf("WARN: %q in %s:%d does not exist is ECS %v.", f.Name, f.Source, f.SourceLine, ecs.Version)
			}
		}
	}

	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err = enc.Encode(flat); err != nil {
			log.Fatal(err)
		}
		return
	case "list":
		for _, f := range flat {
			fmt.Println(f.Name)
		}
	default:
		log.Fatalf("Unknown output format: %q", format)
	}
}
