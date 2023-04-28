package fleetpkg

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Integration struct {
	Manifest    Manifest              `json:"manifest"`
	DataStreams map[string]DataStream `json:"data_streams"`
}

type DataStream struct {
	Manifest DataStreamManifest `json:"manifest"`
}

type Manifest struct {
	Name            string           `json:"name" yaml:"name"`
	Title           string           `json:"title" yaml:"title"`
	Version         string           `json:"version" yaml:"version"`
	Release         string           `json:"release" yaml:"release"`
	Description     string           `json:"description" yaml:"description"`
	Type            string           `json:"type" yaml:"type"`
	Icons           []Icons          `json:"icons" yaml:"icons,omitempty"`
	FormatVersion   string           `json:"format_version" yaml:"format_version"`
	License         string           `json:"license" yaml:"license,omitempty"`
	Categories      []string         `json:"categories" yaml:"categories,omitempty"`
	Conditions      Conditions       `json:"conditions" yaml:"conditions"`
	Screenshots     []Screenshots    `json:"screenshots" yaml:"screenshots,omitempty"`
	Source          Source           `json:"source"`
	Vars            []Var            `json:"vars" yaml:"vars,omitempty"`
	PolicyTemplates []PolicyTemplate `json:"policy_templates" yaml:"policy_templates,omitempty"`
	Owner           Owner            `json:"owner" yaml:"owner"`
}

type Source struct {
	License string `json:"license,omitempty"`
}

type Icons struct {
	Src      string `json:"src"`
	Title    string `json:"title"`
	Size     string `json:"size"`
	Type     string `json:"type"`
	DarkMode bool   `json:"dark_mode" yaml:"dark_mode"`
}

type Conditions struct {
	KibanaVersion       string `json:"kibana.version" yaml:"kibana.version"`
	ElasticSubscription string `json:"elastic.subscription" yaml:"elastic.subscription"`
	Elastic             struct {
		Subscription string `json:"subscription"`
	} `json:"elastic"`
	Kibana struct {
		Version string `json:"version"`
	} `json:"kibana"`
}

type Screenshots struct {
	Src   string `json:"src"`
	Title string `json:"title"`
	Size  string `json:"size"`
	Type  string `json:"type"`
}

type Var struct {
	Name        string   `json:"name"`
	Default     any      `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Multi       bool     `json:"multi"`
	Required    bool     `json:"required"`
	ShowUser    bool     `json:"show_user" yaml:"show_user"`
	Options     []Option `json:"options"` // List of options for 'type: select'.
}

type Option struct {
	Value string `json:"value"`
	Text  string `json:"text"`
}

type Input struct {
	Type         string `json:"type"`
	Title        string `json:"title"`
	Description  string `json:"description" yaml:"description,omitempty"`
	InputGroup   string `json:"input_group" yaml:"input_group,omitempty"`
	TemplatePath string `json:"template_path" yaml:"template_path"`
	Multi        bool   `json:"multi"`
	Vars         []Var  `json:"vars"`
}

type PolicyTemplate struct {
	Name        string        `json:"name"`
	Title       string        `json:"title"`
	Categories  []string      `json:"categories" yaml:"categories,omitempty"`
	Description string        `json:"description"`
	DataStreams []string      `json:"data_streams" yaml:"data_streams,omitempty"`
	Inputs      []Input       `json:"inputs" yaml:"inputs,omitempty"`
	Icons       []Icons       `json:"icons" yaml:"icons,omitempty"`
	Screenshots []Screenshots `json:"screenshots" yaml:"screenshots,omitempty"`
	Multiple    bool          `json:"multiple" yaml:"multiple"`
}

type Owner struct {
	Github string `json:"github"`
}

type DataStreamManifest struct {
	Dataset         string
	DatasetIsPrefix bool           `json:"dataset_is_prefix" yaml:"dataset_is_prefix"`
	ILMPolicy       string         `json:"ilm_policy" yaml:"ilm_policy"`
	Release         string         `json:"release"`
	Title           string         `json:"title"`
	Type            string         `json:"type"`
	Streams         []Stream       `json:"streams"`
	Elasticsearch   map[string]any `json:"elasticsearch"`
}

type Stream struct {
	Input        string `json:"input"`
	Description  string `json:"description"`
	Title        string `json:"title"`
	TemplatePath string `json:"template_path" yaml:"template_path"`
	Vars         []Var  `json:"vars"`
	Enabled      bool   `json:"enabled"`
}

func Load(path string) (*Integration, error) {
	type manifest struct {
		Type string `json:"type"`
	}

	var m manifest
	if err := readYAML(filepath.Join(path, "manifest.yml"), &m, false); err != nil {
		return nil, err
	}

	switch m.Type {
	case "input":
		return LoadInput(path)
	case "integration", "":
		return LoadIntegration(path)
	default:
		return nil, fmt.Errorf("unknown manifest type %q", m.Type)
	}
}

func LoadInput(path string) (*Integration, error) {
	var manifest Manifest
	if err := readYAML(filepath.Join(path, "manifest.yml"), &manifest, false); err != nil {
		return nil, err
	}

	// TODO: This is incomplete. Need to create an Input type.
	integration := &Integration{
		Manifest: manifest,
	}
	return integration, nil
}

func LoadIntegration(path string) (*Integration, error) {
	var manifest Manifest
	if err := readYAML(filepath.Join(path, "manifest.yml"), &manifest, true); err != nil {
		return nil, err
	}

	integration := &Integration{
		Manifest:    manifest,
		DataStreams: map[string]DataStream{},
	}

	dataStreams, err := filepath.Glob(filepath.Join(path, "data_stream/*/manifest.yml"))
	if err != nil {
		return nil, err
	}
	for _, f := range dataStreams {
		var dataStreamManifest DataStreamManifest
		if err := readYAML(f, &dataStreamManifest, true); err != nil {
			return nil, err
		}
		integration.DataStreams[filepath.Base(filepath.Dir(f))] = DataStream{
			Manifest: dataStreamManifest,
		}
	}

	return integration, nil
}

func readYAML(path string, v any, strict bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(strict)

	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("failed reading %s: %w", path, err)
	}
	return nil
}
