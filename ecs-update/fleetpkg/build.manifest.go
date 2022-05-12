package fleetpkg

import "errors"

var buildManifestECSReferencePath = mustYAMLPath("$.dependencies.ecs.reference")

type BuildManifest struct {
	Dependencies Dependencies `json:"dependencies"`
}

type ECS struct {
	Reference string `json:"reference"`
}

type Dependencies struct {
	ECS ECS `json:"ecs"`
}

func (doc *YAMLDocument[BuildManifest]) SetBuildManifestECSReference(version string) (old string, err error) {
	nodes, _ := buildManifestECSReferencePath.Find(&doc.Node)
	if len(nodes) == 0 {
		return "", errors.New("ECS reference not found in build manifest")
	} else if len(nodes) > 1 {
		return "", errors.New("expected only one match")
	}

	old = nodes[0].Value
	nodes[0].Value = version
	return old, nil
}
