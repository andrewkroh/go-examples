package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/andrewkroh/go-examples/fields-yml-gen/ecs"
	"github.com/andrewkroh/go-examples/fields-yml/fieldsyml"
)

// Flags
var (
	warn bool // Warn on invalid ECS field references.
)

func init() {
	flag.BoolVar(&warn, "w", true, "Warn on invalid external ECS field references.")
}

func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

type Conflict struct {
	Name    string           `json:"name"`
	Types   []string         `json:"types"`
	Sources []ConflictSource `json:"conflicts"`
}

type ConflictSource struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Type string `json:"type"`
}

func detectConflicts(fields []fieldsyml.FlatField) []Conflict {
	allFields := map[string][]fieldsyml.FlatField{}

	// Aggregate fields.
	for _, f := range fields {
		list := allFields[f.Name]
		allFields[f.Name] = append(list, f)
	}

	// Check for conflicts.
	var conflicts []Conflict
	for k, list := range allFields {
		conflict := Conflict{
			Name: k,
		}
		types := map[string]struct{}{}
		for _, f := range list {
			if f.Type == "" {
				// Skip ECS external fields for now.
				continue
			}
			types[f.Type] = struct{}{}
			conflict.Sources = append(conflict.Sources, ConflictSource{
				Type: f.Type,
				File: f.Source,
				Line: f.SourceLine,
			})
		}

		if len(types) <= 1 {
			continue
		}

		typeList := make([]string, 0, len(types))
		for t := range types {
			typeList = append(typeList, t)
		}
		conflict.Types = typeList

		conflicts = append(conflicts, conflict)
	}

	return conflicts
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

	conflicts := detectConflicts(flat)
	if len(conflicts) == 0 {
		log.Println("No conflicts.")
		return
	}

	if err = outputJSON(conflicts); err != nil {
		log.Fatal(err)
	}
}
