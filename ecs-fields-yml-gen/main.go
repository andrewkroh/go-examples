package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	fieldList string
)

func init() {
	flag.StringVar(&fieldList, "f", "fields.txt", "list of ECS fields")
}

type FlatFields map[string]FlatField

type FlatField struct {
	Name        string `yaml:"flat_name"`
	Type        string `yaml:"type"`
	Description string `yaml:"short"`
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

	data, err := ioutil.ReadFile("ecs_flat.yml")
	if err != nil {
		log.Fatal(err)
	}

	var fields FlatFields
	if err = yaml.Unmarshal(data, &fields); err != nil {
		log.Fatal(err)
	}

	list, err := readList()
	if err != nil {
		log.Fatal(err)
	}

	var out []Field
	var nonECS []Field
	for _, f := range list {
		flat, found := fields[f]
		if !found {
			nonECS = append(nonECS, Field{Name: f, Type: "keyword"})
			continue
		}
		out = append(out, Field{Name: flat.Name, External: "ecs"})
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

func readList() ([]string, error) {
	data, err := ioutil.ReadFile(fieldList)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(data))

	// Deduplicate.
	m := map[string]struct{}{}
	for _, f := range fields {
		f := strings.Trim(f, `"`)
		m[f] = struct{}{}
	}

	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out, nil
}
