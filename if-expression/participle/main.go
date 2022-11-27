//nolint:govet,revive // Struct-tags use shorthand participle format for readability.
package participle

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2"
)

// Expression = "if" Comparison .
// Comparison = Value ((("!" "=") | ("=" "=")) Comparison)? .
// Value      = "nil" | <float> | <int> | ("true" | "false") | <string> | <ident> .

var parser = participle.MustBuild[Expression]()

type Expression struct {
	Condition *Comparison `"if" @@`
}

type Comparison struct {
	Value      *Value      `@@`
	Op         Operator    `[ @( "!" "=" | "=" "=")`
	Comparison *Comparison `@@ ]`
}

type Value struct {
	Nil           bool     `  @"nil"`
	Number        *float64 `| @Float | @Int`
	Bool          *Boolean `| @("true" | "false")`
	StringLiteral *string  `| @String`
	Identifier    *string  `| @Ident`
}

// Evaluation

type Context struct {
	vars map[string]interface{}
}

func (e *Expression) Eval(ctx Context) interface{} {
	return e.Condition.Eval(ctx)
}

func (c *Comparison) Eval(ctx Context) bool {
	n := c.Value.Eval(ctx)
	if c.Comparison != nil {
		return c.Op.Eval(n, c.Comparison.Eval(ctx))
	}

	// Truthiness
	switch v := n.(type) {
	case string:
		return v != ""
	case float64:
		return v > 0
	case bool:
		return v
	default:
		return false
	}
}

type Operator string

func (o Operator) Eval(l, r interface{}) bool {
	switch o {
	case "!=":
		return l != r
	case "==":
		return l == r
	}
	panic("unsupported operator")
}

func (v *Value) Eval(ctx Context) interface{} {
	switch {
	case v.Nil:
		return nil
	case v.Bool != nil:
		return bool(*v.Bool)
	case v.StringLiteral != nil:
		return *v.StringLiteral
	case v.Number != nil:
		return *v.Number
	case v.Identifier != nil:
		return ctx.vars[*v.Identifier]
	}
	panic("invalid empty value")
}

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true"
	return nil
}

// String

func (e *Expression) String() string {
	return "if (" + e.Condition.String() + ")"
}

func (c *Comparison) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%v", c.Value)
	if c.Op != "" {
		buf.WriteString(" ")
		buf.WriteString(string(c.Op))
		buf.WriteString(" ")
		if c.Comparison.Op != "" {
			buf.WriteString("(")
		}
		buf.WriteString(c.Comparison.String())
		if c.Comparison.Op != "" {
			buf.WriteString(")")
		}
	}
	return buf.String()
}

func (v *Value) String() string {
	switch {
	case v.Nil:
		return "nil"
	case v.Bool != nil:
		return strconv.FormatBool(bool(*v.Bool))
	case v.StringLiteral != nil:
		return *v.StringLiteral
	case v.Number != nil:
		return fmt.Sprintf("%v", *v.Number)
	case v.Identifier != nil:
		return *v.Identifier
	}
	panic("invalid empty value")
}
