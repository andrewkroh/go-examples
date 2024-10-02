package main

import (
	"cmp"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"os/signal"
	"path/filepath"
	"slices"

	"github.com/andrewkroh/go-examples/ocsf/internal/api"
	"github.com/andrewkroh/go-examples/ocsf/internal/ocsf"
)

// USAGE EXAMPLES
//
// # Count unique fields for each class type
//
// ocsf-fields -f | jq 'map_values(length)'
//
// {
//  "account_change": 2576,
//  "admin_group_query": 2167
// }
//
// # Count unique fields across all class types
//
// ocsf-fields -f -c| jq 'length'
//
// 14074
//
// # Count unique fields for each type if we follow one circular reference
//
// ocsf-fields -f -d 1 | jq 'map_values(length)'
//
// {
//  "account_change": 4628,
//  "admin_group_query": 3807
// }
//
// # Get a flat list of fields and their types. Values with a null type are
// # unresolved circular references.
//
// ocsf-fields -f -c | jq 'map({(.name): .type}) | add'
//
// {
//   ".win_service.uid": "string",
//   ".win_service.version": "string"
// }

var (
	flatten    bool
	combine    bool
	cycleDepth int
)

func init() {
	flag.BoolVar(&flatten, "f", false, "Flatten schema by resolving references")
	flag.BoolVar(&combine, "c", false, "Combine fields from all schemas. Any conflicts will cause a failure.")
	flag.IntVar(&cycleDepth, "d", 0, "Number of cycles to traverse when circular references exist")
}

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	schemas, err := loadSchemas(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var output any = schemas
	if flatten {
		classesToFields, err := flattenSchemas(schemas)
		if err != nil {
			log.Fatal(err)
		}
		output = classesToFields

		if combine {
			output, err = combineClasses(classesToFields)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err = enc.Encode(output); err != nil {
		log.Fatal(err)
	}
}

func loadSchemas(ctx context.Context) (map[string]*ocsf.JSONSchema, error) {
	path := filepath.Join(os.TempDir(), "ocsf_classes.gob")

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Bootstrap the cache.
			schemas, err := api.GetAllClassJSONSchemas(ctx)
			if err != nil {
				return nil, err
			}

			f, err := os.Create(path)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			if err = gob.NewEncoder(f).Encode(schemas); err != nil {
				return nil, err
			}
			log.Printf("Saved OCSF class schemas to %s", path)
			return schemas, nil
		}
	}

	log.Printf("Loading OCSF class schemas from %s", path)
	var schemas map[string]*ocsf.JSONSchema
	if err = gob.NewDecoder(f).Decode(&schemas); err != nil {
		return nil, err
	}
	return schemas, nil
}

func flattenSchemas(schemas map[string]*ocsf.JSONSchema) (map[string][]ocsf.NamedField, error) {
	out := make(map[string][]ocsf.NamedField, len(schemas))
	for className, schema := range schemas {
		resolved, err := ocsf.Resolve(schema, cycleDepth)
		if err != nil {
			return nil, err
		}

		out[className] = ocsf.Flatten("", resolved)
	}
	return out, nil
}

func combineClasses(classesToFields map[string][]ocsf.NamedField) ([]ocsf.NamedField, error) {
	allFields := map[string]ocsf.NamedField{}
	for class, fields := range classesToFields {
		for _, field := range fields {
			if existingDef, found := allFields[field.Name]; found {
				if existingDef.Type != field.Type {
					return nil, fmt.Errorf("conflicting definition of %q found in %q", field.Name, class)
				}
			}
			allFields[field.Name] = field
		}
	}
	return slices.SortedFunc(maps.Values(allFields), func(a, b ocsf.NamedField) int {
		return cmp.Compare(a.Name, b.Name)
	}), nil
}
