package changelog

import (
	"gopkg.in/yaml.v3"
)

type Changelog []Release

type Release struct {
	Version string   `json:"version"`
	Changes []Change `json:"changes"`

	// HeadComment holds any comments in the lines preceding the node and
	// not separated by an empty line.
	HeadComment string `yaml:"-"`
}

type Change struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	Link        string `json:"link"`
}

func (r *Release) UnmarshalYAML(node *yaml.Node) error {
	type release Release
	tmp := (*release)(r)

	if err := node.Decode(&tmp); err != nil {
		return err
	}

	r.HeadComment = node.HeadComment
	return nil
}

func (r Release) MarshalYAML() (interface{}, error) {
	type release Release
	tmp := (release)(r)

	var node yaml.Node
	if err := node.Encode(tmp); err != nil {
		return nil, err
	}

	node.HeadComment = r.HeadComment
	return node, nil
}
