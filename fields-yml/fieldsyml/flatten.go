package fieldsyml

import (
	"sort"
	"strings"
)

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
