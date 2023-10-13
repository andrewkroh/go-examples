package deps

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"

	ingestnode "github.com/andrewkroh/go-ingest-node"
)

func readPipeline(r io.Reader) (*Pipeline, error) {
	dec := yaml.NewDecoder(r)

	var pipeline Pipeline
	if err := dec.Decode(&pipeline); err != nil {
		return nil, err
	}

	var idx int64
	visitProcessors(pipeline.Processors, func(processor *Processor) error {
		processor.address = []int64{idx}
		idx++
		return nil
	})
	visitProcessors(pipeline.OnFailure, func(processor *Processor) error {
		processor.address = []int64{idx}
		idx++
		return nil
	})

	return &pipeline, nil
}

func visitProcessors(procs []Processor, visit func(*Processor) error) error {
	for i := range procs {
		p := &procs[i]
		if err := visit(p); err != nil {
			return err
		}

		// TODO: Visit on_failure
	}
	return nil
}

type Pipeline struct {
	Processors []Processor `yaml:"processors"`
	OnFailure  []Processor `yaml:"on_failure"`
}

type Processor struct {
	address   []int64
	raw       *yaml.Node
	container *ingestnode.ProcessorContainer
}

func (p *Processor) Address() string {
	var sb strings.Builder
	for i, idx := range p.address {
		if i > 0 {
			sb.WriteByte('.')
		}
		sb.WriteString(strconv.FormatInt(idx, 10))
	}
	return sb.String()
}

func (p *Processor) Type() string {
	t, _ := p.value()
	return t
}

func (p *Processor) Value() any {
	_, v := p.value()
	return v
}

func (p *Processor) YAML() *yaml.Node {
	return p.raw
}

func (p *Processor) value() (name string, proc any) {
	if p.container == nil {
		return "unknown", nil
	}

	v := reflect.ValueOf(p.container)
	v = v.Elem()

	fields := reflect.VisibleFields(v.Type())
	for _, f := range fields {
		fv := v.FieldByIndex(f.Index)
		if fv.IsNil() {
			continue
		}

		tag := f.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")

		return name, fv.Interface()
	}

	return "", nil
}

func (p *Processor) UnmarshalYAML(value *yaml.Node) error {
	p.raw = value
	return value.Decode(&p.container)
}

type kind int

const (
	field kind = iota
	mustache
	painless
	constantField
)

var insAndOuts = map[string]struct {
	in, out map[string]kind
}{
	"append": {
		in: map[string]kind{
			"if":    painless,
			"value": mustache,
		},
		out: map[string]kind{
			"field": field,
		},
	},
	"attachment": {
		in: map[string]kind{
			"if":    painless,
			"field": field,
		},
		out: map[string]kind{
			"target_field": field,
		},
	},
	"bytes": {
		in: map[string]kind{
			"if":    painless,
			"field": field,
		},
		out: map[string]kind{
			"target_field": field,
		},
	},
	"circle": {
		in: map[string]kind{
			"if":    painless,
			"field": field,
		},
		out: map[string]kind{
			"target_field": field,
		},
	},
	"convert": {
		in: map[string]kind{
			"if":    painless,
			"field": field,
		},
		out: map[string]kind{
			"target_field": field,
		},
	},
	"csv": {
		in: map[string]kind{
			"if":    painless,
			"field": field,
		},
		out: map[string]kind{
			"target_fields": field,
		},
	},
	"date_index_name": {
		in: map[string]kind{
			"if":    painless,
			"field": field,
		},
		out: map[string]kind{
			"_index": constantField,
		},
	},
	"dot_expander": {
		in: map[string]kind{
			"if":   painless,
			"path": field,
		},
		out: map[string]kind{
			"path": field,
		},
	},
	"set": {
		in: map[string]kind{
			"if":    painless,
			"value": mustache,
		},
		out: map[string]kind{
			"field": field,
		},
	},
}

func (p *Processor) dataFlow() (in, out []string, err error) {
	switch v := p.Value().(type) {
	case *ingestnode.ConvertProcessor:
		if v.If != nil {
			keys, err := painlessVariables(*v.If)
			if err != nil {
				return nil, nil, err
			}
			in = append(in, keys...)
		}
		in = append(in, string(v.Field))
		if v.TargetField != nil {
			out = append(out, string(*v.TargetField))
		} else {
			out = append(out, string(v.Field))
		}
	case *ingestnode.LowercaseProcessor:
		if v.If != nil {
			keys, err := painlessVariables(*v.If)
			if err != nil {
				return nil, nil, err
			}
			in = append(in, keys...)
		}
		in = append(in, string(v.Field))
		if v.TargetField != nil {
			out = append(out, string(*v.TargetField))
		} else {
			out = append(out, string(v.Field))
		}
	case *ingestnode.RenameProcessor:
		if v.If != nil {
			keys, err := painlessVariables(*v.If)
			if err != nil {
				return nil, nil, err
			}
			in = append(in, keys...)
		}
		in = append(in, string(v.Field))
		out = append(out, string(v.TargetField))
	case *ingestnode.SetProcessor:
		if v.If != nil {
			keys, err := painlessVariables(*v.If)
			if err != nil {
				return nil, nil, err
			}
			in = append(in, keys...)
		}
		if v.CopyFrom != nil {
			in = append(in, string(*v.CopyFrom))
		}
		if v.Value != nil {
			switch v := v.Value.(type) {
			case string:
				keys, err := mustacheVariables(v)
				if err != nil {
					return nil, nil, err
				}
				in = append(in, keys...)
			}
		}
		out = append(out, string(v.Field))
	default:
		return nil, nil, fmt.Errorf("unknown processor type %T", v)
	}

	// Deduplicate.
	slices.Sort(in)
	in = slices.Compact(in)
	slices.Sort(out)
	out = slices.Compact(out)

	return in, out, nil
}
