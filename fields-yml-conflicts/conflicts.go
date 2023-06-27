package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/andrewkroh/go-fleetpkg"
)

// Flags
var (
	warn                bool // Warn on invalid ECS field references.
	ignoreTextConflicts bool
)

func init() {
	flag.BoolVar(&warn, "w", true, "Warn on invalid external ECS field references.")
	flag.BoolVar(&ignoreTextConflicts, "i", true, "Ignore conflicts between keyword, constant_keyword, wildcard, text, and match_only_text.")
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
	Name string `json:"name,omitempty"`
}

func detectConflicts(fields []fleetpkg.Field) []Conflict {
	allFields := map[string][]fleetpkg.Field{}

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
				File: f.FileMetadata.Path(),
				Line: f.FileMetadata.Line(),
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

func detectInvalidParentDataTypes(fields []fleetpkg.Field) []Conflict {
	// Detect fields with children that are declared as scalar types.
	parentChildRelations := map[string][]fleetpkg.Field{}
	for _, f := range fields {
		if idx := strings.LastIndexByte(f.Name, '.'); idx != -1 {
			parentName := f.Name[:idx]

			slice := parentChildRelations[parentName]
			slice = append(slice, f)
			parentChildRelations[parentName] = slice
		}
	}

	var conflicts []Conflict
	for _, f := range fields {
		switch f.Type {
		case "group", "object", "nested", "array":
			continue
		}
		children, found := parentChildRelations[f.Name]
		if !found {
			continue
		}

		sources := []ConflictSource{
			{
				File: f.FileMetadata.Path(),
				Line: f.FileMetadata.Line(),
				Type: f.Type,
				Name: f.Name,
			},
		}
		for _, child := range children {
			sources = append(sources, ConflictSource{
				File: child.FileMetadata.Path(),
				Line: child.FileMetadata.Line(),
				Type: child.Type,
				Name: child.Name,
			})
		}

		conflicts = append(conflicts, Conflict{
			Name:    f.Name,
			Types:   []string{f.Type, "sub-fields that cannot exists on a " + f.Type},
			Sources: sources,
		})
	}

	return conflicts
}

func isTextTypeConflict(types []string) bool {
	for _, typ := range types {
		switch typ {
		case "keyword", "constant_keyword", "wildcard", "match_only_text", "text":
		default:
			return false
		}
	}
	return true
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

	flat, unresolved := ResolveECSReferences(flat)
	if len(unresolved) > 0 && warn {
		for _, f := range unresolved {
			log.Printf("WARN: %q in %s:%d does not exist is ECS.", f.Name, f.FileMetadata.Path(), f.FileMetadata.Line())
		}
	}

	conflicts := detectConflicts(flat)
	conflicts = append(conflicts, detectInvalidParentDataTypes(flat)...)

	if ignoreTextConflicts {
		filtered := conflicts[:0]
		for _, x := range conflicts {
			if !isTextTypeConflict(x.Types) {
				filtered = append(filtered, x)
			}
		}
		conflicts = filtered
	}

	if len(conflicts) == 0 {
		log.Println("No conflicts.")
		return
	}

	if err = outputJSON(conflicts); err != nil {
		log.Fatal(err)
	}
}
