package fleetpkg

type Manifest struct {
	Name            string            `json:"name"`
	Title           string            `json:"title"`
	Version         string            `json:"version"`
	Release         string            `json:"release"`
	Description     string            `json:"description"`
	Type            string            `json:"type"`
	Icons           []Icons           `json:"icons"`
	FormatVersion   string            `json:"format_version"`
	License         string            `json:"license"`
	Categories      []string          `json:"categories"`
	Conditions      Conditions        `json:"conditions"`
	Screenshots     []Screenshots     `json:"screenshots"`
	Vars            []Vars            `json:"vars"`
	PolicyTemplates []PolicyTemplates `json:"policy_templates"`
	Owner           Owner             `json:"owner"`
}

type Icons struct {
	Src   string `json:"src"`
	Title string `json:"title"`
	Size  string `json:"size"`
	Type  string `json:"type"`
}

type Conditions struct {
	KibanaVersion string `json:"kibana.version"`
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
	Description string `json:"description"`
	InputGroup  string `json:"input_group"`
}

type PolicyTemplates struct {
	Name        string        `json:"name"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	DataStreams []string      `json:"data_streams"`
	Inputs      []Inputs      `json:"inputs"`
	Icons       []Icons       `json:"icons"`
	Screenshots []Screenshots `json:"screenshots"`
}

type Owner struct {
	Github string `json:"github"`
}
