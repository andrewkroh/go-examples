package fieldsyml

import (
	"github.com/andrewkroh/go-examples/fields-yml-gen/ecs"
)

// ResolveECSReferences resolve 'external: ecs' references to get their type
// and description. If there are any unresolved references then hasUnresolved
// will be true (you can iterate the returned values to find 'external: ecs'
// fields without a type).
func ResolveECSReferences(flat []FlatField) (resolved []FlatField, unresolved []FlatField) {
	out := make([]FlatField, 0, len(flat))
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
			ecsField.Source = f.Source
			ecsField.SourceLine = f.SourceLine
			out = append(out, ecsField)
		}
	}
	return out, unresolved
}

func lookupECSField(name string) []FlatField {
	if f := ecs.GetField(name); f != nil {
		flat := FlatField{
			Name:        f.FlatName,
			Type:        f.Type,
			Description: f.Description,
			External:    "ecs",
		}
		return []FlatField{flat}
	}

	// NOTE: This does not resolve groups of fields anymore.
	// https://github.com/elastic/elastic-package/pull/818
	return nil
}
