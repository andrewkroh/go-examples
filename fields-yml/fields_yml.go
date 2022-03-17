package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Flags
var (
	format string // Output format (list, json).
)

func init() {
	flag.StringVar(&format, "f", "list", "Output format (list or json). Defaults to list.")
}

type Field struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	External    string  `json:"external,omitempty"`
	Fields      []Field `json:"fields,omitempty"`
	Description string  `json:"description,omitempty"`
}

type FlatField struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	External    string `json:"external,omitempty"`
	Description string `json:"description,omitempty"`
}

func ReadFieldsYAML(globs ...string) ([]Field, error) {
	var matches []string
	for _, glob := range globs {
		m, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m...)
	}

	var fields []Field
	for _, file := range matches {
		tmpFields, err := readFieldsYAML(file)
		if err != nil {
			return nil, err
		}
		fields = append(fields, tmpFields...)
	}

	return fields, nil
}

func readFieldsYAML(path string) ([]Field, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var fields []Field
	if err = yaml.NewDecoder(f).Decode(&fields); err != nil {
		return nil, err
	}

	return fields, err
}

func FlattenFields(fields []Field) ([]FlatField, error) {
	var flat []FlatField
	for _, rootField := range fields {
		tmpFlats, err := flattenField(nil, rootField)
		if err != nil {
			return nil, err
		}
		flat = append(flat, tmpFlats...)
	}

	sort.Slice(flat, func(i, j int) bool {
		return flat[i].Name < flat[j].Name
	})

	return flat, nil
}

func flattenField(key []string, f Field) ([]FlatField, error) {
	// Leaf node.
	if len(f.Fields) == 0 {
		leafName := splitName(f.Name)

		name := make([]string, len(key)+len(leafName))
		copy(name, key)
		copy(name[len(key):], leafName)

		return []FlatField{
			{
				Name:        strings.Join(name, "."),
				Type:        f.Type,
				External:    f.External,
				Description: f.Description,
			},
		}, nil
	}

	parentName := append(key, splitName(f.Name)...)
	var flat []FlatField
	for _, child := range f.Fields {
		tmpFlats, err := flattenField(parentName, child)
		if err != nil {
			return nil, err
		}
		flat = append(flat, tmpFlats...)
	}

	return flat, nil
}

func splitName(name string) []string {
	return strings.Split(name, ".")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("Must pass a list of fields.yml files.")
	}

	fields, err := ReadFieldsYAML(flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}

	flat, err := FlattenFields(fields)
	if err != nil {
		log.Fatal(err)
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
