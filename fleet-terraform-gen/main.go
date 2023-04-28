package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"text/template"

	"golang.org/x/exp/maps"

	"github.com/andrewkroh/go-examples/fleet-terraform-gen/internal/fleetpkg"
	"github.com/andrewkroh/go-examples/fleet-terraform-gen/internal/terraform"
)

var usage = `
fleet-terraform-gen generates a Terraform module in the Terraform JSON syntax
that can be used to manage the installation of single Fleet data stream. It
follows a model of having a single data stream contained in a Fleet package
policy.

It extracts all of the variables definitions from a Fleet package and exposes
them as Terraform variables.

The generated module depends on another module named fleet_package_policy that
handles making the API calls to Fleet to create the policy.
`[1:]

var (
	packagePath        string
	policyTemplateName string
	dataStreamName     string
	inputType          string

	listPackages bool
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage+"\nVersion: %s\n\nUsage of %s:\n", getVersion(), filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.StringVar(&packagePath, "pkg", "", "Path to Fleet package directory. (Required)")
	flag.StringVar(&policyTemplateName, "policy-template", "", "Policy template (defaults to the first one in the manifest).")
	flag.StringVar(&dataStreamName, "data-stream", "", "Data stream to generate module for. (Required)")
	flag.StringVar(&inputType, "input", "", "Input type to select from the data stream. (Required)")
	flag.BoolVar(&listPackages, "list", false, "List data streams. Must set -pkg to the elastic/integrations packages/ directory.")
}

func main() {
	flag.Parse()

	if listPackages {
		// NOTE: This is WIP to print out the meaningful combination of
		// package policy template, data streams, and inputs.
		err := walkPackages(packagePath, func(pkg *fleetpkg.Integration, err error) error {
			if err != nil {
				log.Println(err)
				return nil
			}

			for _, pt := range pkg.Manifest.PolicyTemplates {
				for _, input := range pt.Inputs {
					columns := []string{pkg.Manifest.Name, pt.Name, input.Type}
					if len(pt.DataStreams) == 0 {
						fmt.Println(strings.Join(columns, "|"))
					} else {
						for _, ds := range pt.DataStreams {
							fmt.Println(strings.Join(append(columns, ds), "|"))
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// Verify the required variables.
	if packagePath == "" {
		log.Fatal("-pkg path is required.")
	}
	if dataStreamName == "" {
		log.Fatal("-data-stream is required.")
	}
	if inputType == "" {
		log.Fatal("-input is required.")
	}

	moduleJSON, err := generateModule(packagePath, policyTemplateName, dataStreamName, inputType)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	fmt.Printf("%s", moduleJSON)
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "(devel)" {
		return "latest"
	}
	return info.Main.Version
}

func generateModule(packagePath, policyTemplateName, dataStreamName, inputType string) ([]byte, error) {
	// Read in the package metadata.
	pkg, err := fleetpkg.Load(packagePath)
	if err != nil {
		return nil, err
	}

	var (
		manifest            = pkg.Manifest
		policyTemplate      *fleetpkg.PolicyTemplate
		policyTemplateInput *fleetpkg.Input
		dataStream          *fleetpkg.DataStream
		stream              *fleetpkg.Stream
	)

	// Policy template.
	{
		if policyTemplateName == "" {
			policyTemplate = &pkg.Manifest.PolicyTemplates[0]
			policyTemplateName = policyTemplate.Name
		} else {
			for i, pt := range pkg.Manifest.PolicyTemplates {
				if pt.Name == policyTemplateName {
					policyTemplate = &pkg.Manifest.PolicyTemplates[i]
					break
				}
			}
			if policyTemplate == nil {
				return nil, fmt.Errorf("policy template %q not found", policyTemplateName)
			}
		}

		for i, input := range policyTemplate.Inputs {
			if input.Type == inputType {
				policyTemplateInput = &policyTemplate.Inputs[i]
				break
			}
		}
		if policyTemplateInput == nil {
			return nil, fmt.Errorf("input %q was not found within policy template %q", inputType, policyTemplateName)
		}
	}
	// Data stream.
	{
		ds, found := pkg.DataStreams[dataStreamName]
		if !found {
			return nil, fmt.Errorf("data stream %q was not found in the package", dataStreamName)
		}
		dataStream = &ds
	}
	// Input type.
	{
		for i, s := range dataStream.Manifest.Streams {
			if s.Input == inputType {
				stream = &dataStream.Manifest.Streams[i]
				break
			}
		}
		if stream == nil {
			return nil, fmt.Errorf("input type %q was not found in data stream %q", inputType, dataStreamName)
		}
	}

	tfVariables := map[string]terraform.Variable{
		"fleet_agent_policy_id": {
			Type:        "string",
			Description: "Agent policy ID to add the package policy to.",
		},
		"fleet_data_stream_namespace": {
			Type:        "string",
			Description: "Namespace to use for the data stream.",
			Default:     &terraform.NullableValue{Value: "default"},
		},
		"fleet_package_version": {
			Type:        "string",
			Description: "Version of the " + pkg.Manifest.Name + " package to use.",
			Default:     &terraform.NullableValue{Value: pkg.Manifest.Version},
		},
	}

	// Iterate over all variables in the package and create Terraform variables.
	packageLevelVarAssociations, err := addVariables(pkg.Manifest.Vars, tfVariables)
	if err != nil {
		return nil, err
	}
	policyTemplateLevelVarAssociations, err := addVariables(policyTemplate.Vars, tfVariables)
	if err != nil {
		return nil, err
	}
	inputLevelVarAssociations, err := addVariables(policyTemplateInput.Vars, tfVariables)
	if err != nil {
		return nil, err
	}
	dataStreamVarAssociations, err := addVariables(stream.Vars, tfVariables)
	if err != nil {
		return nil, err
	}

	packageLevelVarExpression, err := buildVariableExpression(packageLevelVarAssociations)
	if err != nil {
		return nil, err
	}
	inputLevelVarExpression, err := buildVariableExpression(inputLevelVarAssociations)
	if err != nil {
		return nil, err
	}
	// Empirically it appears that input package policy template variables are treated
	// the same as data stream variables.
	dataStreamVarExpression, err := buildVariableExpression(dataStreamVarAssociations, policyTemplateLevelVarAssociations)
	if err != nil {
		return nil, err
	}

	// Get a list of data streams so that we can disable all the ones not being
	// used. This avoids validation errors for required variables.
	allDataStreams := maps.Keys(pkg.DataStreams)
	if len(policyTemplate.DataStreams) > 0 {
		allDataStreams = policyTemplate.DataStreams
	}

	tf := &terraform.File{
		Comment:   fmt.Sprintf("Generated by github.com/andrewkroh/go-examples/fleet-terraform-gen %v - DO NOT EDIT", getVersion()),
		Variables: tfVariables,
		Modules: map[string]terraform.Module{
			"fleet_package_policy": {
				Source: "../../fleet_package_policy",
				Params: toMap(FleetPackagePolicyModule{
					AgentPolicyID:           "${var.fleet_agent_policy_id}",
					PackagePolicyName:       manifest.Name + "-" + dataStreamName + "-${var.fleet_data_stream_namespace}",
					PackageName:             manifest.Name,
					PackageVersion:          "${var.fleet_package_version}",
					Namespace:               "${var.fleet_data_stream_namespace}",
					PolicyTemplate:          policyTemplate.Name,
					DataStream:              dataStreamName,
					InputType:               stream.Input,
					PackageVariablesJSON:    packageLevelVarExpression,
					InputVariablesJSON:      inputLevelVarExpression,
					DataStreamVariablesJSON: dataStreamVarExpression,
					AllDataStreams:          allDataStreams,
				}),
			},
		},
		Outputs: map[string]terraform.Output{
			"id": {
				Description: "Package policy ID",
				Value:       "${module.fleet_package_policy.id}",
			},
		},
	}

	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err = enc.Encode(tf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func addVariables(vars []fleetpkg.Var, m map[string]terraform.Variable) (associations map[string]string, err error) {
	associations = make(map[string]string, len(vars))
	for _, v := range vars {
		tfName, err := addVariable(v, m)
		if err != nil {
			return nil, err
		}
		associations[v.Name] = tfName
	}

	return associations, nil
}

func addVariable(v fleetpkg.Var, m map[string]terraform.Variable) (tfName string, err error) {
	tfVar := terraform.Variable{
		Description: v.Description,
	}

	if tfVar.Type, err = dataType(v); err != nil {
		return "", err
	}
	if tfVar.Type == "string" && isSensitive(v) {
		tfVar.Sensitive = terraform.Ptr(true)
	}
	if v.Required {
		tfVar.Nullable = terraform.Ptr(false)
	}
	if v.Default != nil {
		// Pass the default to Terraform.
		tfVar.Default = &terraform.NullableValue{Value: v.Default}
	} else if !v.Required {
		tfVar.Default = &terraform.NullableValue{}
	}

	// Append yaml suffix to indicate to users that they must yamlencode() the value.
	name := v.Name
	if v.Type == "yaml" {
		name += "_yaml"
	}

	// Don't allow variables shadowing. If Fleet allows it then this may need changed.
	if existing, found := m[name]; found {
		return "", fmt.Errorf("duplicate variable found [%#v, %#v]", existing, tfVar)
	}
	m[name] = tfVar
	return name, nil
}

func isSensitive(v fleetpkg.Var) bool {
	name := strings.ToLower(v.Name)
	switch {
	case v.Type == "password",
		strings.Contains(name, "token") && !strings.Contains(name, "file"),
		strings.Contains(name, "api_key"),
		strings.Contains(name, "secret"):
		return true
	default:
		return false
	}
}

func dataType(v fleetpkg.Var) (string, error) {
	var tfType string
	switch v.Type {
	case "bool":
		tfType = "bool"
	case "integer":
		tfType = "number"
	case "password", "email", "select", "text", "textarea", "time_zone", "url", "yaml":
		tfType = "string"
	default:
		// package-spec controls the allow types.
		return "", fmt.Errorf("unknown fleet variable type %q", v.Type)
	}

	if v.Multi {
		tfType = "list(" + tfType + ")"
	}
	return tfType, nil
}

type FleetPackagePolicyModule struct {
	AgentPolicyID           string   `json:"agent_policy_id"`
	PackagePolicyName       string   `json:"package_policy_name,omitempty"`
	PackageName             string   `json:"package_name"`
	PackageVersion          string   `json:"package_version"`
	Namespace               string   `json:"namespace"`
	PolicyTemplate          string   `json:"policy_template"`
	DataStream              string   `json:"data_stream"`
	InputType               string   `json:"input_type"`
	PackageVariablesJSON    string   `json:"package_variables_json,omitempty"`
	InputVariablesJSON      string   `json:"input_variables_json,omitempty"`
	DataStreamVariablesJSON string   `json:"data_stream_variables_json,omitempty"`
	AllDataStreams          []string `json:"all_data_streams"`
}

func toMap(v any) map[string]any {
	buf := new(bytes.Buffer)

	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err)
	}

	var m map[string]any
	dec := json.NewDecoder(buf)
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		panic(err)
	}

	return m
}

var variableExpressionTemplate = template.Must(template.New("jsonencode").
	Option("missingkey=error").
	Parse(`${jsonencode({
{{- range $fleetVar, $tfVar := . }}
  {{ $fleetVar }} = var.{{ $tfVar }}
{{- end }}
})}`))

func buildVariableExpression(associations ...map[string]string) (string, error) {
	allAssociations := joinMaps(associations...)

	if len(allAssociations) == 0 {
		return "", nil
	}

	buf := new(bytes.Buffer)
	if err := variableExpressionTemplate.Execute(buf, allAssociations); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func walkPackages(dir string, walk func(pkg *fleetpkg.Integration, err error) error) error {
	allPackages, err := filepath.Glob(filepath.Join(dir, "*/manifest.yml"))
	if err != nil {
		return err
	}

	for _, manifestPath := range allPackages {
		integration, err := fleetpkg.Load(filepath.Dir(manifestPath))
		if err = walk(integration, err); err != nil {
			return err
		}
	}

	return nil
}

func joinMaps(maps ...map[string]string) map[string]string {
	if len(maps) == 0 {
		return nil
	}
	if len(maps) == 1 {
		return maps[0]
	}

	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			if _, found := out[k]; found {
				panic("Multiple definitions for variable " + k)
			}
			out[k] = v
		}
	}

	return out
}
