package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
)

var (
	input string
	key   string
)

func init() {
	flag.StringVar(&input, "in", "", "input ndjson file")
	flag.StringVar(&key, "key", "events", "name of json array in output")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if input == "" {
		log.Fatal("-in is required")
	}

	data, err := ioutil.ReadFile(input)
	if err != nil {
		log.Fatal("Failed to read input file.", err)
	}

	// Read ndjson.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var objs []interface{}
	for {
		var obj interface{}

		err := dec.Decode(&obj)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		objs = append(objs, obj)
	}

	// Output array of objects.
	output := map[string]interface{}{
		"events": objs,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err = enc.Encode(output); err != nil {
		log.Fatal(err)
	}
}
