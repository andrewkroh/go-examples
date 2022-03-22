package fieldsyml

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
