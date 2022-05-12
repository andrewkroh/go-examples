package fleetpkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

type YAMLDocument[T any] struct {
	FilePath     string
	Node         yaml.Node
	RawYAML      []byte
	OriginalData T
}

func ReadYAMLDocument[T Manifest | BuildManifest | IngestNodePipeline | SampleEvent](path string) (*YAMLDocument[T], error) {
	yamlData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	doc := &YAMLDocument[T]{
		FilePath: path,
		RawYAML:  yamlData,
	}

	if err := yaml.Unmarshal(yamlData, &doc.Node); err != nil {
		return nil, err
	}

	if err = doc.Node.Decode(&doc.OriginalData); err != nil {
		return nil, err
	}

	return doc, nil
}

func (doc *YAMLDocument[any]) WriteYAML(w io.Writer) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()

	// Keep document separator (---) if it existed in the original.
	if bytes.HasPrefix(doc.RawYAML, []byte("---")) {
		w.Write([]byte("---\n"))
	}

	if err := enc.Encode(&doc.Node); err != nil {
		return err
	}

	return nil
}

func (doc *YAMLDocument[any]) WriteJSON(w io.Writer, indent int) error {
	ifc, err := yamlNodeToInterface(&doc.Node)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	if indent > 0 {
		enc.SetIndent("", strings.Repeat(" ", indent))
	}
	enc.SetEscapeHTML(false)

	if err := enc.Encode(ifc); err != nil {
		return err
	}

	return nil
}

type Package struct {
	Manifest      YAMLDocument[Manifest]       `json:"manifest"`
	BuildManifest *YAMLDocument[BuildManifest] `json:"build,omitempty"` // Optional
	DataStreams   map[string]DataStream        `json:"data_streams"`
}

type DataStream struct {
	DefaultPipeline *YAMLDocument[IngestNodePipeline] `json:"default_pipeline,omitempty"` // Optional
	SampleEvent     *YAMLDocument[SampleEvent]        `json:"sample_event,omitempty"`     // Optional
}

func ReadPackage(path string) (*Package, error) {
	manifest, err := ReadYAMLDocument[Manifest](filepath.Join(path, "manifest.yml"))
	if err != nil {
		return nil, err
	}

	buildManifest, err := ReadYAMLDocument[BuildManifest](filepath.Join(path, "_dev/build/build.yml"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	dataStreamList, err := filepath.Glob(filepath.Join(path, "data_stream/*"))
	if err != nil {
		return nil, err
	}

	pkg := &Package{
		Manifest:      *manifest,
		BuildManifest: buildManifest,
		DataStreams:   make(map[string]DataStream, len(dataStreamList)),
	}

	for _, dsPath := range dataStreamList {
		dataStreamName := filepath.Base(dsPath)

		pipeline, err := ReadYAMLDocument[IngestNodePipeline](filepath.Join(dsPath, "elasticsearch/ingest_pipeline/default.yml"))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}

		sampleEvent, err := ReadYAMLDocument[SampleEvent](filepath.Join(dsPath, "sample_event.json"))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}

		pkg.DataStreams[dataStreamName] = DataStream{
			DefaultPipeline: pipeline,
			SampleEvent:     sampleEvent,
		}
	}

	return pkg, nil
}

func mustYAMLPath(path string) *yamlpath.Path {
	p, err := yamlpath.NewPath(path)
	if err != nil {
		panic(err)
	}
	return p
}

// FixYAMLMaps recursively converts maps with interface{} keys, as returned by
// yaml.Unmarshal, to maps of string keys, as expected by the json encoder
// that will be used when delivering the pipeline to Elasticsearch.
// Will return an error when something other than a string is used as a key.
func fixYAMLMaps(elem interface{}) (_ interface{}, err error) {
	switch v := elem.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			keyS, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("key '%v' is not string but %T", key, key)
			}
			if result[keyS], err = fixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
		return result, nil
	case map[string]interface{}:
		for key, value := range v {
			if v[key], err = fixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
	case []interface{}:
		for idx, value := range v {
			if v[idx], err = fixYAMLMaps(value); err != nil {
				return nil, err
			}
		}
	}
	return elem, nil
}

func yamlNodeToInterface(n *yaml.Node) (interface{}, error) {
	var obj interface{}

	if err := n.Decode(&obj); err != nil {
		return nil, err
	}

	return fixYAMLMaps(obj)
}
