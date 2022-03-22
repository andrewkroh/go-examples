package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/andrewkroh/go-examples/fields-yml-gen/ecs"
)

var inputFile string

func init() {
	flag.StringVar(&inputFile, "in", "-", "list of fields to read")
}

type Field struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type,omitempty"`
	Description string `yaml:"description,omitempty"`
	External    string `yaml:"external,omitempty"`
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	r := os.Stdin
	if inputFile != "-" {
		f, err := os.Open(inputFile)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()

		r = f
	}

	list, err := readFields(r)
	if err != nil {
		log.Fatal(err)
	}
	list = deduplicate(list)
	list = filterSpecialElasticFleetFields(list)

	var out []Field
	var nonECS []Field
	for _, f := range list {
		ecsField := ecs.GetField(f)
		if ecsField == nil {
			nonECS = append(nonECS, Field{Name: f, Type: "keyword", Description: "TODO"})
			continue
		}
		out = append(out, Field{Name: ecsField.FlatName, External: "ecs"})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	sort.Slice(nonECS, func(i, j int) bool {
		return nonECS[i].Name < nonECS[j].Name
	})

	fmt.Println("---")
	fmt.Println("# ECS fields")
	enc := yaml.NewEncoder(os.Stdout)
	if err = enc.Encode(out); err != nil {
		log.Fatal(err)
	}
	enc.Close()

	fmt.Println()
	fmt.Println("---")
	fmt.Println("# Non-ECS fields")
	enc = yaml.NewEncoder(os.Stdout)
	if err = enc.Encode(nonECS); err != nil {
		log.Fatal(err)
	}
	enc.Close()
}

var (
	elasticPackageUndefinedFieldRegex = regexp.MustCompile(`(?m)\[\d+\] field "(.*)" is undefined`)
	fieldNameListRegex                = regexp.MustCompile(`(?m)^["']?([^"']+)["']?$`)
)

func readFields(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if matches := elasticPackageUndefinedFieldRegex.FindAllStringSubmatch(string(data), -1); len(matches) > 0 {
		out := make([]string, 0, len(matches))
		for _, m := range matches {
			if len(m) != 2 {
				panic("unexpected match size")
			}

			out = append(out, m[1])
		}

		return out, nil
	}

	fields := strings.Fields(string(data))
	out := make([]string, 0, len(fields))
	for _, f := range strings.Fields(string(data)) {
		f := strings.Trim(f, `"'`)
		out = append(out, f)
	}

	return out, nil
}

func deduplicate(list []string) []string {
	// Deduplicate.
	m := map[string]struct{}{}
	for _, v := range list {
		m[v] = struct{}{}
	}

	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

func filterSpecialElasticFleetFields(list []string) []string {
	out := make([]string, 0, len(list))
	for _, f := range list {
		switch f {
		// These fields are accounted for in mappings added by Fleet.
		case "event.agent_id_status":
		case "event.ingested":
		default:
			out = append(out, f)
		}
	}
	return out
}
