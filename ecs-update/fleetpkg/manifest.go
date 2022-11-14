package fleetpkg

type Manifest struct {
	Name            string            `json:"name" yaml:"name"`
	Title           string            `json:"title" yaml:"title"`
	Version         string            `json:"version" yaml:"version"`
	Release         string            `json:"release" yaml:"release"`
	Description     string            `json:"description" yaml:"description"`
	Type            string            `json:"type" yaml:"type"`
	Icons           []Icons           `json:"icons" yaml:"icons,omitempty"`
	FormatVersion   string            `json:"format_version" yaml:"format_version"`
	License         string            `json:"license" yaml:"license,omitempty"`
	Categories      []string          `json:"categories" yaml:"categories,omitempty"`
	Conditions      Conditions        `json:"conditions" yaml:"conditions"`
	Screenshots     []Screenshots     `json:"screenshots" yaml:"screenshots,omitempty"`
	Vars            []Vars            `json:"vars" yaml:"vars,omitempty"`
	PolicyTemplates []PolicyTemplates `json:"policy_templates" yaml:"policy_templates,omitempty"`
	Owner           Owner             `json:"owner" yaml:"owner"`
}

type Icons struct {
	Src   string `json:"src"`
	Title string `json:"title"`
	Size  string `json:"size"`
	Type  string `json:"type"`
}

type Conditions struct {
	KibanaVersion string `json:"kibana.version" yaml:"kibana.version"`
}

type Screenshots struct {
	Src   string `json:"src"`
	Title string `json:"title"`
	Size  string `json:"size"`
	Type  string `json:"type"`
}

type Vars struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Title    string      `json:"title"`
	Multi    bool        `json:"multi"`
	Required bool        `json:"required"`
	ShowUser bool        `json:"show_user"`
	Default  interface{} `json:"default,omitempty"`
}

type Inputs struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description" yaml:"description,omitempty"`
	InputGroup  string `json:"input_group" yaml:"input_group,omitempty"`
}

type PolicyTemplates struct {
	Name        string        `json:"name"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	DataStreams []string      `json:"data_streams" yaml:"data_streams,omitempty"`
	Inputs      []Inputs      `json:"inputs" yaml:"inputs,omitempty"`
	Icons       []Icons       `json:"icons" yaml:"icons,omitempty"`
	Screenshots []Screenshots `json:"screenshots" yaml:"screenshots,omitempty"`
}

type Owner struct {
	Github string `json:"github"`
}
