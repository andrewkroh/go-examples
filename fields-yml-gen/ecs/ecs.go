package ecs

import (
	"bytes"
	_ "embed"
	"strings"

	"gopkg.in/yaml.v3"
)

// Version is the ECS version embedded.
const Version = "8.2"

//go:generate curl -L -o ecs_flat.yml https://raw.githubusercontent.com/elastic/ecs/8.2/generated/ecs/ecs_flat.yml

var (
	//go:embed ecs_flat.yml
	ecsFlatYML string

	ecsFields map[string]Field
)

func init() {
	fields, err := readFields()
	if err != nil {
		panic(err)
	}

	ecsFields = make(map[string]Field, len(fields))
	for _, f := range fields {
		ecsFields[f.FlatName] = f
	}
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
	dec := yaml.NewDecoder(bytes.NewBufferString(ecsFlatYML))
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

func GetField(name string) *Field {
	f, found := ecsFields[name]
	if !found {
		return nil
	}

	copy := f
	return &copy
}

// GetFieldSet returns all fields whose name contains the given prefix.
func GetFieldSet(prefix string) []Field {
	// Only allow full key-name prefixes.
	if !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}

	var out []Field
	for _, f := range ecsFields {
		if strings.HasPrefix(f.FlatName, prefix) {
			out = append(out, f)
		}
	}

	return out
}
