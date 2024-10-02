package ocsf

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type JSONSchema struct {
	Defs                 map[string]*JSONSchema `json:"$defs,omitempty"`
	Type                 string                 `json:"type,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Title                string                 `json:"title,omitempty"`
	Const                any                    `json:"const,omitempty"`
	Reference            *string                `json:"$ref,omitempty"`
	Enum                 []any                  `json:"enum,omitempty"`
	Items                *JSONSchema            `json:"items,omitempty"`
	Properties           map[string]*JSONSchema `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	AdditionalProperties *bool                  `json:"additionalProperties,omitempty"`
	Default              any                    `json:"default,omitempty"`
	ID                   string                 `json:"$id,omitempty"`
	Schema               string                 `json:"$schema,omitempty"`

	cyclePath []string // Metadata about the circular references.
}

type NamedField struct {
	*JSONSchema
	Name string `json:"name"`
}

// Flatten returns a flattened view of the schema where all entries in the
// returned list are leaf fields. Unresolved fields will contain a non-null
// JSONSchema.Reference.
func Flatten(name string, schema *JSONSchema) []NamedField {
	fields := flatten(name, schema, nil)
	slices.SortFunc(fields, func(a, b NamedField) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return fields
}

func flatten(name string, schema *JSONSchema, fields []NamedField) []NamedField {
	switch {
	case len(schema.Properties) > 0:
		for k, v := range schema.Properties {
			fields = flatten(name+"."+k, v, fields)
		}
	case schema.Items != nil:
		fields = flatten(name+"[]", schema.Items, fields)
	default:
		fields = append(fields, NamedField{
			Name:       name,
			JSONSchema: schema,
		})
	}
	return fields
}

// Resolve resolves references in the fields of the schema. JSON Schema allows
// circular references within the same schema. This translates into a tree of
// infinite depth. We cannot allow infinite traversal do the bounds are controlled
// via depth.
//
// The depth argument controls how many cycles deep the resolver
// is allowed to traverse. A depth of 0 means to stop traversing when as soon
// as a cycle is detected. The node containing the final circular reference is
// left unresolved.
func Resolve(schema *JSONSchema, depth int) (*JSONSchema, error) {
	r := &resolver{
		Schema:     schema,
		state:      nil,
		cycleDepth: depth,
	}
	resolved, err := r.traverse(deepCopyJSONSchema(schema))
	if err != nil {
		return nil, err
	}
	resolved.Defs = nil
	return resolved, nil
}

type resolver struct {
	Schema     *JSONSchema
	state      []string // List of nodes traversed.
	cycleDepth int      // Number of cycles to allow.
}

func (r *resolver) pushNode(node string) {
	r.state = append(r.state, node)
}

func (r *resolver) popNode() {
	if len(r.state) > 0 {
		r.state = r.state[:len(r.state)-1]
		return
	}
}

func (r *resolver) hasCycle(node string) (cycle bool, path []string) {
	var count int
	for _, n := range r.state {
		if n == node {
			count++
		}
	}
	if count > r.cycleDepth {
		nodePath := make([]string, len(r.state)+1)
		copy(nodePath, r.state)
		nodePath = append(nodePath, node)
		return true, nodePath
	}
	return false, nil
}

func (r *resolver) resolveRefs(root *JSONSchema) (*JSONSchema, error) {
	if root.Reference == nil {
		return r.traverse(root)
	}

	if cycle, path := r.hasCycle(*root.Reference); cycle {
		root.cyclePath = path
		return root, nil
	}

	ref := *root.Reference
	r.pushNode(ref)
	defer r.popNode()

	definition, err := r.dereference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to dereference %q: %w", ref, err)
	}

	// Clear the reference to avoid traversing it again.
	root.Reference = nil
	copyAttributes(definition, root)

	// Walk the schema to resolve more references.
	return r.traverse(root)
}

func (r *resolver) traverse(root *JSONSchema) (*JSONSchema, error) {
	if len(root.Properties) == 0 && root.Items == nil {
		// Nothing to traverse. Not an object and not an array.
		return root, nil
	}
	if root.Reference != nil {
		return r.resolveRefs(root)
	}
	for k, v := range root.Properties {
		s, err := r.resolveRefs(v)
		if err != nil {
			return nil, err
		}
		root.Properties[k] = s
	}
	if root.Items != nil {
		s, err := r.resolveRefs(root.Items)
		if err != nil {
			return nil, err
		}
		root.Items = s
	}
	return root, nil
}

func (r *resolver) dereference(ref string) (*JSONSchema, error) {
	typeName, found := strings.CutPrefix(ref, "#/$defs/")
	if !found {
		return nil, fmt.Errorf("unexpected $ref value %q", ref)
	}

	resolved, found := r.Schema.Defs[typeName]
	if !found {
		return nil, fmt.Errorf("definition for %q not found in $defs", ref)
	}
	return deepCopyJSONSchema(resolved), nil
}

func copyAttributes(src, dst *JSONSchema) {
	if src == nil || dst == nil {
		return
	}

	if dst.Type == "" {
		dst.Type = src.Type
	}
	if dst.Description == "" {
		dst.Description = src.Description
	}
	if dst.Title == "" {
		dst.Title = src.Title
	}
	if dst.Const == nil {
		dst.Const = src.Const
	}
	if dst.Reference == nil || *dst.Reference == "" {
		dst.Reference = src.Reference
	}
	if len(dst.Enum) == 0 {
		dst.Enum = src.Enum
	}
	if dst.Items == nil {
		dst.Items = src.Items
	}
	if dst.Properties == nil {
		dst.Properties = make(map[string]*JSONSchema, len(src.Properties))
		for k, v := range src.Properties {
			s := *v
			dst.Properties[k] = &s
		}
	}
	if len(dst.Required) == 0 {
		dst.Required = src.Required
	}
	if dst.AdditionalProperties == nil {
		dst.AdditionalProperties = src.AdditionalProperties
	}
	if dst.Default == nil {
		dst.Default = src.Default
	}
	if dst.ID == "" {
		dst.ID = src.ID
	}
	if dst.Schema == "" {
		dst.Schema = src.Schema
	}
}

func deepCopyJSONSchema(src *JSONSchema) *JSONSchema {
	clone := *src
	if src.Defs != nil {
		clone.Defs = make(map[string]*JSONSchema)
		for k, v := range src.Defs {
			clone.Defs[k] = deepCopyJSONSchema(v)
		}
	}
	if src.Properties != nil {
		clone.Properties = make(map[string]*JSONSchema)
		for k, v := range src.Properties {
			clone.Properties[k] = deepCopyJSONSchema(v)
		}
	}
	if src.Items != nil {
		clone.Items = deepCopyJSONSchema(src.Items)
	}
	return &clone
}
