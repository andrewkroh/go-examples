package api

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"

	"github.com/andrewkroh/go-examples/ocsf/internal/ocsf"
)

// GetClassNames fetches the class names from https://schema.ocsf.io/export/classes.
func GetClassNames(ctx context.Context) ([]string, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://schema.ocsf.io/export/classes", nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	var m map[string]struct{}
	dec := json.NewDecoder(resp.Body)
	if err = dec.Decode(&m); err != nil {
		return nil, err
	}

	return slices.Collect(maps.Keys(m)), nil
}

// GetJSONSchema fetches the JSON Schema definition of the specified class.
// It uses https://schema.ocsf.io/schema/classes/$className.
func GetJSONSchema(ctx context.Context, className string) (*ocsf.JSONSchema, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://schema.ocsf.io/schema/classes/"+url.PathEscape(className), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	var s *ocsf.JSONSchema
	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()
	if err = dec.Decode(&s); err != nil {
		return nil, err
	}

	return s, nil
}

func GetAllClassJSONSchemas(ctx context.Context) (map[string]*ocsf.JSONSchema, error) {
	classes, err := GetClassNames(ctx)
	if err != nil {
		return nil, err
	}

	schemas := map[string]*ocsf.JSONSchema{}
	for _, class := range classes {
		s, err := GetJSONSchema(ctx, class)
		if err != nil {
			return nil, fmt.Errorf("failed to get ocsf schema class %q: %w", class, err)
		}
		schemas[class] = s
	}

	return schemas, nil
}
