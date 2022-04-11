package changelog

import (
	"errors"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	document *yaml.Node
	version  *yaml.Node
}

func (m *Manifest) SetVersion(v string) error {
	return m.version.Encode(VersionString(v))
}

func (m *Manifest) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errors.New("expected mapping node in YAML")
	}

	m.document = value
	for i, n := range value.Content {
		if n.Value == "version" {
			if len(value.Content) > i {
				m.version = value.Content[i+1]
			}
			break
		}
	}

	if m.version == nil {
		return errors.New("version not found in manifest)")
	}

	return nil
}

func (m Manifest) MarshalYAML() (interface{}, error) {
	return m.document, nil
}
