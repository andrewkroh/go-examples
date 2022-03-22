package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/andrewkroh/go-examples/fields-yml/fieldsyml"
)

// Flags
var (
	format string // Output format (list, json).
)

func init() {
	flag.StringVar(&format, "f", "list", "Output format (list or json). Defaults to list.")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("Must pass a list of fields.yml files.")
	}

	fields, err := fieldsyml.ReadFieldsYAML(flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}

	flat, err := fieldsyml.FlattenFields(fields)
	if err != nil {
		log.Fatal(err)
	}

	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err = enc.Encode(flat); err != nil {
			log.Fatal(err)
		}
		return
	case "list":
		for _, f := range flat {
			fmt.Println(f.Name)
		}
	default:
		log.Fatalf("Unknown output format: %q", format)
	}
}
