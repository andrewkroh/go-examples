package ocsf

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestResolve(t *testing.T) {
	data, err := os.ReadFile("testdata/process_activity.json")
	if err != nil {
		t.Fatal(err)
	}

	var s *JSONSchema
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		t.Fatal(err)
	}

	s, err = Resolve(s, 0)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range Flatten("", s) {
		if f.Reference != nil && f.Type != "" {
			t.Errorf("unresolved refs should not have a type at %q", f.Name)
		}
	}
}
