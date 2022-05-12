package fleetpkg

import "errors"

var sampleEventECSVersionPath = mustYAMLPath("$.ecs.version")

type SampleEvent map[string]interface{}

func (doc *YAMLDocument[BuildManifest]) SetSampleEventECSVersion(version string) (old string, err error) {
	nodes, _ := sampleEventECSVersionPath.Find(&doc.Node)
	if len(nodes) == 0 {
		return "", errors.New("ecs.version not found in sample event")
	} else if len(nodes) > 1 {
		return "", errors.New("expected only one match")
	}

	old = nodes[0].Value
	nodes[0].Value = version
	return old, nil
}
