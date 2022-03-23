package fieldsyml

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func ReadFieldsYAML(globs ...string) ([]Field, error) {
	var matches []string
	for _, glob := range globs {
		m, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m...)
	}

	var fields []Field
	for _, file := range matches {
		tmpFields, err := readFieldsYAML(file)
		if err != nil {
			return nil, err
		}
		fields = append(fields, tmpFields...)
	}

	return fields, nil
}

func readFieldsYAML(path string) ([]Field, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var fields []Field
	if err = yaml.NewDecoder(f).Decode(&fields); err != nil {
		return nil, fmt.Errorf("failed reading from %q: %w", path, err)
	}

	// Set file path.
	for i := range fields {
		fields[i].Source = path
	}

	return fields, err
}
