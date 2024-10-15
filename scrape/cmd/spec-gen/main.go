package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"maps"
	"os"
	"slices"
	"strings"
	"text/template"
	"unicode"

	"github.com/fatih/camelcase"
	"github.com/mitchellh/go-wordwrap"
)

type parameter struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
	Type        string
}

func main() {
	flag.Parse()

	path := flag.Arg(0)
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	var processors map[string][]parameter
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&processors); err != nil {
		log.Fatal()
	}

	maps.DeleteFunc(processors, func(s string, parameters []parameter) bool {
		switch s {
		case "community_id", "network_direction", "fingerprint", "registered_domain", "grok", "date":
			return false
		}
		return true
	})

	for name, proc := range processors {
		proc = slices.DeleteFunc(proc, func(p parameter) bool {
			switch p.Name {
			case "description", "if", "ignore_failure", "on_failure", "tag":
				return true
			}
			return false
		})
		for i, p := range proc {
			switch p.Name {
			case "field", "target_field", "source_port", "destination_port", "iana_number", "icmp_type", "icmp_code", "transport":
				proc[i].Type = "Field"
			case "fields":
				proc[i].Type = "Fields"
			case "ignore_missing":
				proc[i].Type = "boolean"
			case "internal_networks":
				proc[i].Type = "Ip[]"
			case "seed":
				proc[i].Type = "integer"
			default:
				switch {
				case strings.Contains(p.Name, "ip") && (name == "community_id" || name == "network_direction"):
					proc[i].Type = "Field"
				case strings.Contains(p.Name, "field"):
					proc[i].Type = "Field"
				default:
					proc[i].Type = "string"
				}
			}
		}

		//slices.SortFunc(proc, func(a, b parameter) int {
		//	return cmp.Compare(a.Name, b.Name)
		//})

		if err = tmpl.Execute(os.Stdout, map[string]any{
			"ClassName":  name,
			"Parameters": proc,
		}); err != nil {
			log.Fatalf("in %s: %v", name, err)
		}
	}
}

var tmpl = template.Must(template.New("class").Funcs(template.FuncMap{
	"goIDName": goIDName,
	"wordwrap": wordwrap.WrapString,
	"prefix": func(prefix, s string) string {
		parts := strings.Split(s, "\n")
		for i, p := range parts {
			parts[i] = prefix + p
		}
		return strings.Join(parts, "\n")
	},
}).Parse(strings.ReplaceAll(`
export class {{ goIDName .ClassName}}Processor extends ProcessorBase {
{{- range $index, $parameter := .Parameters }}
  /**
{{ prefix "   * " (wordwrap $parameter.Description 76) }}
{{- if not $parameter.Required }}
{{- if and (ne $parameter.Default "") (ne $parameter.Default "<none>") }}
   * @server_default {{ $parameter.Default }}
{{- end }}
{{- end }}
   */
  {{ $parameter.Name }}{{if not $parameter.Required }}?{{end}}: {{ $parameter.Type }}
{{- end }}
}
`, "”", "`")))

// identifierFixes contains some special cases to make the identifier
// follow Go conventions.
var identifierFixes = map[string]string{
	"urldecode": "url_decode",
}

// acronyms is a set of acronyms that should be capitalized in identifiers.
var acronyms = map[string]bool{
	"csv":  true,
	"iana": true,
	"icmp": true,
	"id":   true,
	"ip":   true,
	"json": true,
	"kv":   true,
	"uri":  true,
	"url":  true,
	"os":   true,
}

// goIDName returns a Go identifier representing the configuration option name.
func goIDName(name string) string {
	if fix, found := identifierFixes[strings.ToLower(name)]; found {
		name = fix
	}

	snakeCaseParts := strings.FieldsFunc(name, func(r rune) bool {
		return unicode.IsPunct(r)
	})

	var allParts []string
	for _, p := range snakeCaseParts {
		allParts = append(allParts, camelcase.Split(p)...)
	}

	for i, p := range allParts {
		if _, isAcronym := acronyms[strings.ToLower(p)]; isAcronym {
			allParts[i] = strings.ToTitle(p)
			continue
		}

		allParts[i] = strings.Title(p)
	}

	return strings.Join(allParts, "")
}
