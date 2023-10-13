package deps

import (
	"strings"

	"github.com/cbroglie/mustache"

	"github.com/andrewkroh/go-examples/ingest-node/internal/painless"
	"github.com/andrewkroh/go-examples/ingest-node/internal/painless/parser"
)

// mustacheVariables extracts the field key names referenced in mustache variables.
func mustacheVariables(s string) ([]string, error) {
	if s == "" {
		return nil, nil
	}

	t, err := mustache.ParseString(s)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, t := range t.Tags() {
		if t.Type() == mustache.Variable {
			keys = append(keys, t.Name())
		}
	}
	return keys, nil
}

// painlessVariables extracts the field names reference in a Painless expression.
func painlessVariables(expression string) ([]string, error) {
	vars, err := painless.ParseExpression(expression)
	if err != nil {
		return nil, err
	}

	fieldNames := make([]string, 0, len(vars))
	for _, v := range vars {
		fieldNames = append(fieldNames, normalizePainlessVariable(v))
	}
	return fieldNames, nil
}

var painlessVarReplacer = strings.NewReplacer("?.", ".")

func normalizePainlessVariable(v *parser.DynamicContext) string {
	name := painlessVarReplacer.Replace(v.GetText())
	return strings.TrimPrefix(name, "ctx.")
}
