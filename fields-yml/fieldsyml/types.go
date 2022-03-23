package fieldsyml

import "gopkg.in/yaml.v3"

type Field struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	External    string  `json:"external,omitempty"`
	Fields      []Field `json:"fields,omitempty"`
	Description string  `json:"description,omitempty"`

	Source     string `json:"-"` // File from which field was read.
	SourceLine int    `json:"-"` // Line from which field was read.
}

func (f *Field) UnmarshalYAML(value *yaml.Node) error {
	// Prevent recursion by creating a new type that does not implement Unmarshaler.
	type notField Field
	x := (*notField)(f)

	if err := value.Decode(&x); err != nil {
		return err
	}
	f.SourceLine = value.Line
	return nil
}

type FlatField struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	External    string `json:"external,omitempty"`
	Description string `json:"description,omitempty"`

	Source     string `json:"-"` // File from which field was read.
	SourceLine int    `json:"-"` // Line from which field was read.
}
