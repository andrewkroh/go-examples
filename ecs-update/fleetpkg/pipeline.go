package fleetpkg

import (
	"errors"

	"gopkg.in/yaml.v3"
)

var ingestNodePipelineSetECSVersionValuePath = mustYAMLPath("$.processors[?(@.set.field == 'ecs.version')].set.value")

type IngestNodePipeline struct {
	Description string       `json:"description"`
	Processors  []*Processor `json:"processors"`
	OnFailure   []*Processor `json:"on_failure"`
}

type Processor struct {
	Type       string
	Attributes map[string]interface{}
}

func (p *Processor) UnmarshalYAML(value *yaml.Node) error {
	var procMap map[string]map[string]interface{}
	if err := value.Decode(&procMap); err != nil {
		return err
	}

	for k, v := range procMap {
		p.Type = k
		p.Attributes = v
		break
	}

	return nil
}

func (doc *YAMLDocument[IngestNodePipeline]) SetIngestNodePipelineECSVersion(version string) (old string, err error) {
	nodes, _ := ingestNodePipelineSetECSVersionValuePath.Find(&doc.Node)
	if len(nodes) == 0 {
		return "", errors.New("set processor not found in pipeline")
	} else if len(nodes) > 1 {
		return "", errors.New("expected only one match")
	}

	old = nodes[0].Value
	nodes[0].Value = version
	return old, nil
}
