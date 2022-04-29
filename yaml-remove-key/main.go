package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/andrewkroh/go-examples/yaml-remove-key/filter"
)

var (
	indent     int
	filterKeys arrayFlag
	write      bool
)

func init() {
	flag.IntVar(&indent, "indent", 2, "YAML indentation")
	flag.Var(&filterKeys, "key", "Key to filter. May be used more than once and value can be comma separated.")
	flag.BoolVar(&write, "w", false, "Write modification to file.")
}

func main() {
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		if err := filterStdin(); err != nil {
			log.Fatal("Error", err)
		}

		return
	}

	for _, f := range files {
		if err := filterFile(f); err != nil {
			log.Fatal("Error", err)
		}
	}
}

func filterStdin() error {
	node, numChanges, err := filterData(os.Stdin)
	if err != nil {
		return nil
	}

	if err = outputYAML(os.Stdout, node); err != nil {
		return err
	}

	if numChanges > 0 {
		log.Printf("stdin: %d changes", numChanges)
	}
	return nil
}

func filterFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	node, numChanges, err := filterData(f)
	if err != nil {
		f.Close()
		return nil
	}
	f.Close()

	if write {
		if numChanges > 0 {
			f, err = os.Create(path)
			if err != nil {
				return fmt.Errorf("failed to open file for writing: %w", err)
			}
			defer f.Close()

			if err := outputYAML(f, node); err != nil {
				return err
			}
		}
	} else {
		if err := outputYAML(os.Stdout, node); err != nil {
			return err
		}
	}

	if numChanges > 0 {
		log.Printf("%s: %d changes", path, numChanges)
	}

	return nil
}

func filterData(r io.Reader) (*yaml.Node, int, error) {
	var node yaml.Node
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&node); err != nil {
		return nil, 0, fmt.Errorf("failed to decode document: %w", err)
	}

	numChanges := filter.Keys(&node, filterKeys...)
	return &node, numChanges, nil
}

func outputYAML(w io.Writer, node *yaml.Node) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(indent)
	defer enc.Close()

	return enc.Encode(node)
}
