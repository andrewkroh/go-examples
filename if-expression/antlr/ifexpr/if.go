//nolint:revive // This is not a complete implementation so some parameters are unused.
package ifexpr

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"go.uber.org/multierr"

	"github.com/andrewkroh/go-examples/if-expression/antlr/ifexpr/parser"
)

//go:generate rm -rf parser/
//go:generate antlr -Dlanguage=Go -o parser -visitor If.g4

type Expression struct {
	tree antlr.ParseTree
}

func New(expression string) (*Expression, error) {
	if expression == "" {
		return nil, errors.New("empty expression")
	}

	inputStream := antlr.NewInputStream(expression)
	lexer := parser.NewIfLexer(inputStream)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewIfParser(tokens)
	errListener := &errorCollector{}
	p.RemoveErrorListeners()
	p.AddErrorListener(errListener)

	// This evaluates the expression and triggers the errors.
	tree := p.Program()
	if len(errListener.errs) > 0 {
		return nil, multierr.Combine(errListener.errs...)
	}

	return &Expression{tree: tree}, nil
}

type Context struct {
	vars map[string]any
}

func (x *Expression) Evaluate(context *Context) (bool, error) {
	visitor := &ifVisitor{
		vars: context.vars,
	}

	out := visitor.Visit(x.tree)
	if visitor.err != nil {
		return false, visitor.err
	}

	if b, ok := out.(bool); !ok {
		return false, fmt.Errorf("expected a bool but got %T", out)
	} else {
		return b, nil
	}
}

type ifVisitor struct {
	err  error
	vars map[string]interface{}
}

func (i *ifVisitor) Visit(tree antlr.ParseTree) interface{} {
	if i.err != nil {
		return nil
	}

	switch expr := tree.(type) {
	case *parser.ProgramContext:
		out := expr.Accept(i)
		if i.err != nil {
			return nil
		}
		return out
	case *parser.IfStatementContext:
		out := expr.Accept(i)
		if i.err != nil {
			return nil
		}
		return out
	default:
		panic(fmt.Errorf("unhandled type %T", expr))
	}
}

func (i *ifVisitor) VisitChildren(node antlr.RuleNode) interface{} {
	panic("implement me")
}

func (i *ifVisitor) VisitTerminal(node antlr.TerminalNode) interface{} {
	panic("implement me")
}

func (i *ifVisitor) VisitErrorNode(node antlr.ErrorNode) interface{} {
	panic("implement me")
}

func (i *ifVisitor) VisitProgram(ctx *parser.ProgramContext) interface{} {
	for _, s := range ctx.AllStatement() {
		return s.Accept(i)
	}
	return nil
}

func (i *ifVisitor) VisitStatement(ctx *parser.StatementContext) interface{} {
	return ctx.IfStatement().Accept(i)
}

func (i *ifVisitor) VisitIfStatement(ctx *parser.IfStatementContext) interface{} {
	return isTruthy(ctx.BoolExpression().Accept(i))
}

func isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

func (i *ifVisitor) VisitParenBoolExpression(ctx *parser.ParenBoolExpressionContext) interface{} {
	panic("implement me")
}

func (i *ifVisitor) VisitBoolExpressionPath(ctx *parser.BoolExpressionPathContext) interface{} {
	return ctx.Path().Accept(i)
}

func (i *ifVisitor) VisitBoolExpressionBoolean(ctx *parser.BoolExpressionBooleanContext) interface{} {
	return ctx.Boolean().Accept(i)
}

func (i *ifVisitor) VisitBoolExpressionNotEqual(ctx *parser.BoolExpressionNotEqualContext) interface{} {
	lhs := ctx.GetLeft().Accept(i)
	rhs := ctx.GetRight().Accept(i)
	return lhs != rhs
}

func (i *ifVisitor) VisitBoolExpressionEqual(ctx *parser.BoolExpressionEqualContext) interface{} {
	lhs := ctx.GetLeft().Accept(i)
	rhs := ctx.GetRight().Accept(i)
	return lhs == rhs
}

func (i *ifVisitor) VisitTerm(ctx *parser.TermContext) interface{} {
	if ctx.Literal() != nil {
		return ctx.Literal().Accept(i)
	}
	panic("unhandled paren expression")
}

func (i *ifVisitor) VisitLiteral(ctx *parser.LiteralContext) interface{} {
	switch {
	case ctx.String_() != nil:
		return ctx.String_().Accept(i)
	case ctx.Number() != nil:
		return ctx.Number().Accept(i)
	case ctx.Path() != nil:
		return ctx.Path().Accept(i)
	case ctx.Boolean() != nil:
		return ctx.Boolean().Accept(i)
	case ctx.Nil_() != nil:
		return ctx.Nil_().Accept(i)
	default:
		panic("unhandled literal type")
	}
}

func (i *ifVisitor) VisitPath(ctx *parser.PathContext) interface{} {
	// Lookup variable and return value.
	return i.vars[ctx.GetText()]
}

func (*ifVisitor) VisitString(ctx *parser.StringContext) interface{} {
	switch {
	case ctx.SINGLE_STRING() != nil:
		return strings.Trim(ctx.SINGLE_STRING().GetText(), "'")
	case ctx.DOUBLE_STRING() != nil:
		return strings.Trim(ctx.DOUBLE_STRING().GetText(), `"`)
	default:
		panic("unhandled string literal type")
	}
}

func (i *ifVisitor) VisitNumber(ctx *parser.NumberContext) interface{} {
	num, err := strconv.ParseFloat(ctx.DECIMAL_NUMBER().GetText(), 64)
	if err != nil {
		i.err = err
		return nil
	}
	return num
}

func (*ifVisitor) VisitBoolean(ctx *parser.BooleanContext) interface{} {
	return ctx.GetText() == "true"
}

func (*ifVisitor) VisitNil(_ *parser.NilContext) interface{} {
	return nil
}

var _ parser.IfVisitor = (*ifVisitor)(nil)

type errorCollector struct {
	errs []error
}

func (el *errorCollector) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	el.errs = append(el.errs, fmt.Errorf("line %d:%d %v", line, column, msg))
}

func (el *errorCollector) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	panic("implement me")
}

func (el *errorCollector) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	panic("implement me")
}

func (el *errorCollector) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs *antlr.ATNConfigSet) {
	panic("implement me")
}
