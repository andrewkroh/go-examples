package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/andrewkroh/go-ecs"
	"github.com/andrewkroh/go-fleetpkg"
)

// Flags
var (
	format              string // Output format (list, json).
	warn                bool   // Warn on invalid ECS field references.
	ignoreTextConflicts bool
)

func init() {
	flag.StringVar(&format, "f", "list", "Output format (list or json). Defaults to list.")
	flag.BoolVar(&warn, "w", true, "Warn on invalid external ECS field references.")
	flag.BoolVar(&ignoreTextConflicts, "i", true, "Ignore conflicts between keyword, constant_keyword, wildcard, text, and match_only_text.")
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

	var resolved, unresolved []*fleetpkg.Field
nextField:
	for i := range flat {
		f := &flat[i]

		switch f.External {
		case "ecs":
			// Use the latest ECS version in the library.
			ecsField, err := ecs.Lookup(f.Name, "")
			if err != nil {
				unresolved = append(unresolved, f)
				continue nextField
			}

			f.Type = ecsField.DataType
			f.Description = ecsField.Description
			f.Pattern = ecsField.Pattern
		case "":
			if f.Type == "" {
				f.Type = "keyword"

				if warn {
					log.Printf("WARN: %q in %s:%d does have a 'type'.", f.Name, f.FileMetadata.Path(), f.FileMetadata.Line())
				}
			}
		default:
			log.Fatalf("Unexpected 'external' value %s for field %s at %s:%d", f.External, f.Name, f.Pattern, f.Line())
		}

		resolved = append(resolved, f)
	}

	if len(unresolved) > 0 && warn {
		for _, f := range unresolved {
			log.Printf("WARN: %q in %s:%d does not exist is ECS.", f.Name, f.FileMetadata.Path(), f.FileMetadata.Line())
		}
	}

	conflicts := detectConflicts(resolved)
	conflicts = append(conflicts, detectInvalidParentDataTypes(resolved)...)

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

	slices.SortFunc(conflicts, func(a, b Conflict) int {
		return cmp.Compare(a.Name, b.Name)
	})

	switch format {
	case "json":
		if err = outputJSON(conflicts); err != nil {
			log.Fatalf("Failed to output JSON: %v", err)
		}
	case "list":
		outputText(conflicts)
	default:
		log.Fatalf("Unknown output format: %q", format)
	}
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

func compareConflictSource(a, b ConflictSource) int {
	if x := cmp.Compare(a.File, b.File); x != 0 {
		return x
	}
	return cmp.Compare(a.Line, b.Line)
}

// detectConflicts detects field definitions a field with the same name
// is declared with a different field data type.
func detectConflicts(fields []*fleetpkg.Field) []Conflict {
	allFields := map[string][]*fleetpkg.Field{}

	// Aggregate fields by flat name.
	for i := range fields {
		name := fields[i].Name
		list := allFields[name]
		allFields[name] = append(list, fields[i])
	}

	// Check for conflicts.
	var conflicts []Conflict
	for k, list := range allFields {
		conflict := Conflict{
			Name: k,
		}
		types := map[string]struct{}{}
		for _, f := range list {
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

		conflict.Types = maps.Keys(types)
		slices.SortFunc(conflict.Sources, compareConflictSource)
		conflicts = append(conflicts, conflict)
	}

	return conflicts
}

func detectInvalidParentDataTypes(fields []*fleetpkg.Field) []Conflict {
	// Detect scalar fields that have declared children.
	parentChildRelations := map[string][]*fleetpkg.Field{}
	for i := range fields {
		name := fields[i].Name
		if idx := strings.LastIndexByte(name, '.'); idx != -1 {
			parentName := name[:idx]

			slice := parentChildRelations[parentName]
			slice = append(slice, fields[i])
			parentChildRelations[parentName] = slice
		}
	}

	for k, v := range parentChildRelations {
		slices.SortFunc(v, func(a, b *fleetpkg.Field) int {
			if x := cmp.Compare(a.Path(), b.Path()); x != 0 {
				return x
			}
			return cmp.Compare(a.Line(), b.Line())
		})
		parentChildRelations[k] = v
	}

	var conflicts []Conflict
	for _, f := range fields {
		// Skip non-scalar field types.
		switch f.Type {
		case "group", "object", "nested", "array":
			continue
		}

		// Check if this scalar field has any children.
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
			Types:   []string{"sub-fields that cannot exists on a " + f.Type},
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

func outputJSON(conflicts []Conflict) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(conflicts)
}

func outputText(conflicts []Conflict) {
	for _, c := range conflicts {
		fmt.Printf("%s - %s\n", c.Name, strings.Join(c.Types, ", "))
		for _, loc := range c.Sources {
			// Special-case the output format for "sub-fields" problems
			// to include the name of the fields.
			if len(c.Types) > 0 && strings.HasPrefix(c.Types[0], "sub-fields") {
				fmt.Printf("  %s:%d - %s (%s)\n", loc.File, loc.Line, loc.Name, loc.Type)
				continue
			}

			fmt.Printf("  %s:%d - %s\n", loc.File, loc.Line, loc.Type)
		}
	}
}
