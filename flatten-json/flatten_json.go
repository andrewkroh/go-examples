package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/exp/maps"
)

var usage = `
flatten-json transforms JSON read from stdin into a flat representation.
Each key represents the JSON path to a scalar element.

Version: %s

Usage of %s:
`[1:]

// Flags
var (
	outputFormat string // Output format (defaults to kv). Options are kv, json.
	noColor      bool   // Disable color.
	noIndex      bool   // Remove JSON path index numbers (e.g. .foo[1].bar -> .foo[].bar)
)

func init() {
	flag.StringVar(&outputFormat, "o", "kv", "Output format (options are kv, json)")
	flag.BoolVar(&noColor, "no-color", false, "Disable terminal color (disabled by default when no TTY is attached or NO_COLOR is set).")
	flag.BoolVar(&noIndex, "no-index", false, "Remove JSON path index numbers (e.g. .foo[1].bar -> .foo[].bar).")
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	return info.Main.Version
}

// readNext reads the next value from the stream.
func readNext(dec *json.Decoder) (any, error) {
	var o any
	if err := dec.Decode(&o); err != nil {
		return nil, err
	}
	return o, nil
}

type pairs struct {
	kvs []keyValue
}

func (p *pairs) push(k string, v any) {
	p.kvs = append(p.kvs, keyValue{k: k, v: v})
}

func (p *pairs) MarshalJSON() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteByte('{')
	for i, p := range p.kvs {
		if i > 0 {
			buf.WriteByte(',')
		}

		if err := marshallJSON(p.k, buf); err != nil {
			return nil, err
		}

		buf.WriteByte(':')

		if err := marshallJSON(p.v, buf); err != nil {
			return nil, err
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

type keyValue struct {
	k string
	v any
}

func jsonPathFlatten(v any) *pairs {
	p := &pairs{}
	flatten(".", v, p)
	return p
}

func flatten(root string, v any, p *pairs) {
	switch t := v.(type) {
	case map[string]any:
		// Determinism.
		keys := maps.Keys(t)
		slices.Sort(keys)

		for _, k := range keys {
			v = t[k]
			flatten(join(root, k), v, p)
		}
	case []interface{}:
		for i, v := range t {
			if noIndex {
				flatten(root+"[]", v, p)
			} else {
				flatten(root+"["+strconv.Itoa(i)+"]", v, p)
			}
		}
	default:
		p.push(root, v)
	}
}

func join(a, b string) string {
	if strings.HasSuffix(a, ".") {
		return a + b
	}
	return a + "." + b
}

type formatter interface {
	Format(*pairs) ([]byte, error)
}

type keyValueFormatter struct {
	withColor bool
}

func (f *keyValueFormatter) Format(p *pairs) ([]byte, error) {
	blue := color.New(color.Bold, color.FgBlue)
	yellow := color.New(color.FgYellow)
	if !f.withColor {
		blue.DisableColor()
		yellow.DisableColor()
	}

	buf := new(bytes.Buffer)
	jsonBuf := new(bytes.Buffer)
	for _, p := range p.kvs {
		blue.Fprint(buf, p.k)
		buf.Write([]byte(" = "))

		jsonBuf.Reset()
		if err := marshallJSON(p.v, jsonBuf); err != nil {
			return nil, err
		}

		yellow.Fprintf(buf, "%s", jsonBuf.Bytes())
	}

	return buf.Bytes(), nil
}

func marshallJSON(v any, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

type jsonFormatter struct{}

func (*jsonFormatter) Format(p *pairs) ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func run(r io.Reader, f formatter, w io.Writer) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()

	for dec.More() {
		v, err := readNext(dec)
		if err != nil {
			return fmt.Errorf("failed reading json: %w", err)
		}

		pairs := jsonPathFlatten(v)

		output, err := f.Format(pairs)
		if err != nil {
			return fmt.Errorf("failed formatting: %w", err)
		}

		if _, err = w.Write(output); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, getVersion(), filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	var f formatter
	switch outputFormat {
	case "kv":
		f = &keyValueFormatter{withColor: !noColor}
	case "json":
		f = &jsonFormatter{}
	default:
		log.Fatal("invalid output format")
	}

	err := run(os.Stdin, f, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
