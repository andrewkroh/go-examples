package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
)

// Flags
var (
	list   bool // Output a list of flattened key names.
	pretty bool // Output the flattened JSON as pretty.
)

func init() {
	flag.BoolVar(&list, "l", false, "Output flattened key names as a list (non-JSON).")
	flag.BoolVar(&pretty, "p", false, "Output flattened object as pretty formatted.")
}

func ReadObject(dec *json.Decoder) (map[string]interface{}, error) {
	obj := map[string]interface{}{}
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func ToJSON(m map[string]interface{}, pretty bool, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}

	return enc.Encode(m)
}

// Flatten takes a map and returns a new one where nested maps are replaced
// by dot-delimited keys.
func Flatten(m map[string]interface{}) map[string]interface{} {
	o := make(map[string]interface{})
	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			nm := Flatten(child)
			for nk, nv := range nm {
				o[k+"."+nk] = nv
			}
		default:
			o[k] = v
		}
	}
	return o
}

func KeyList(obj map[string]interface{}) []string {
	keys := make([]string, 0, len(obj))

	for k := range obj {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}

func main() {
	flag.Parse()

	dec := json.NewDecoder(os.Stdin)
	dec.UseNumber()

	for dec.More() {
		obj, err := ReadObject(dec)
		if err != nil {
			log.Fatal("Error reading input JSON:", err)
		}

		flat := Flatten(obj)

		if list {
			for _, key := range KeyList(flat) {
				fmt.Println(key)
			}
			continue
		}

		if err = ToJSON(flat, pretty, os.Stdout); err != nil {
			log.Fatal(err)
		}

		fmt.Println("---")
	}
}
