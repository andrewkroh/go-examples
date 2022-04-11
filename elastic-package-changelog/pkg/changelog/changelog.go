package changelog

import (
	"gopkg.in/yaml.v3"
)

type Changelog []yaml.Node

type Release struct {
	Version VersionString `json:"version"`
	Changes []Change      `json:"changes"`
}

type Change struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	Link        string `json:"link"`
}

type VersionString string

func (s VersionString) MarshalYAML() (interface{}, error) {
	var n yaml.Node
	if err := n.Encode(string(s)); err != nil {
		return nil, err
	}
	n.Style = yaml.DoubleQuotedStyle
	return n, nil
}

func NewReleaseFromNode(n yaml.Node) (*Release, error) {
	var r Release
	if err := n.Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r Release) ToYAMLNode() (*yaml.Node, error) {
	var n yaml.Node
	if err := n.Encode(r); err != nil {
		return nil, err
	}
	return &n, nil
}
