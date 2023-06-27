package main

import (
	"bytes"
	_ "embed"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/andrewkroh/go-fleetpkg"
)

//go:generate curl -L -o ecs_flat.yml https://raw.githubusercontent.com/elastic/ecs/v8.8.0/generated/ecs/ecs_flat.yml

var (
	//go:embed ecs_flat.yml
	ecsFlatYML []byte

	ecsFields     map[string]Field
	ecsFieldsOnce sync.Once
)

func ecsFieldsFlat() map[string]Field {
	ecsFieldsOnce.Do(func() {
		fields, err := readFields()
		if err != nil {
			panic(err)
		}

		ecsFields = make(map[string]Field, len(fields))
		for _, f := range fields {
			ecsFields[f.FlatName] = f
		}
	})

	return ecsFields
}

type Field struct {
	DashedName  string        `yaml:"dashed_name"`
	Description string        `yaml:"description"`
	Example     string        `yaml:"example"`
	FlatName    string        `yaml:"flat_name"`
	IgnoreAbove int           `yaml:"ignore_above"`
	Level       string        `yaml:"level"`
	Name        string        `yaml:"name"`
	Normalize   []interface{} `yaml:"normalize"`
	Short       string        `yaml:"short"`
	Type        string        `yaml:"type"`
}

func readFields() ([]Field, error) {
	dec := yaml.NewDecoder(bytes.NewReader(ecsFlatYML))
	var fields map[string]Field
	if err := dec.Decode(&fields); err != nil {
		return nil, err
	}

	// Don't trust the map key name.
	list := make([]Field, 0, len(fields))
	for _, f := range fields {
		list = append(list, f)
	}

	return list, nil
}

func getField(name string) *Field {
	f, found := ecsFieldsFlat()[name]
	if !found {
		return nil
	}

	copy := f
	return &copy
}

// ResolveECSReferences resolve 'external: ecs' references to get their type
// and description. If there are any unresolved references then they will be
// contained in unresolved.
func ResolveECSReferences(flat []fleetpkg.Field) (resolved, unresolved []fleetpkg.Field) {
	out := make([]fleetpkg.Field, 0, len(flat))
	for _, f := range flat {
		if f.External != "ecs" {
			out = append(out, f)
			continue
		}

		fields := lookupECSField(f.Name)
		if len(fields) == 0 {
			unresolved = append(unresolved, f)
			continue
		}

		for _, ecsField := range fields {
			ecsField.FileMetadata = f.FileMetadata
			out = append(out, ecsField)
		}
	}
	return out, unresolved
}

func lookupECSField(name string) []fleetpkg.Field {
	if f := getField(name); f != nil {
		flat := fleetpkg.Field{
			Name:        f.FlatName,
			Type:        f.Type,
			Description: f.Description,
			External:    "ecs",
		}
		return []fleetpkg.Field{flat}
	}

	// NOTE: This does not resolve groups of fields anymore.
	// https://github.com/elastic/elastic-package/pull/818
	return nil
}
