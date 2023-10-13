package painless

import (
	"errors"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	"github.com/andrewkroh/go-examples/ingest-node/internal/painless/parser"
)

func ParseExpression(e string) ([]*parser.DynamicContext, error) {
	if e == "" {
		return nil, errors.New("empty expression")
	}

	inputStream := antlr.NewInputStream(e)
	lexer := parser.NewPainlessLexer(inputStream)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewPainlessParser(tokens)
	errListener := &errorCollector{}
	p.RemoveErrorListeners()
	p.AddErrorListener(errListener)

	// This evaluates the expression and triggers the errors.
	tree := p.Noncondexpression()
	if len(errListener.errs) > 0 {
		return nil, errors.Join(errListener.errs...)
	}

	v := &painlessVisitor{}
	if maybeErr := v.Visit(tree); maybeErr != nil {
		if err, ok := maybeErr.(error); ok {
			return nil, err
		}
		return nil, fmt.Errorf("unexpected return type from Visit: got %#v", maybeErr)
	}

	return v.variables, v.err
}

type errorCollector struct {
	errs []error
}

func (ec errorCollector) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	ec.errs = append(ec.errs, fmt.Errorf("line %d:%d %v", line, column, msg))
}

func (ec errorCollector) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	panic("ReportAmbiguity")
}

func (ec errorCollector) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
	panic("ReportAttemptingFullContext")
}

func (ec errorCollector) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs *antlr.ATNConfigSet) {
	panic("ReportContextSensitivity")
}

var _ antlr.ErrorListener = (*errorCollector)(nil)

type painlessVisitor struct {
	err       error
	variables []*parser.DynamicContext
}

var _ parser.PainlessParserVisitor = (*painlessVisitor)(nil)

func (p *painlessVisitor) Visit(tree antlr.ParseTree) interface{} {
	if p.err != nil {
		return nil
	}

	//fmt.Printf("%T | %v\n", tree, tree.GetText()) TODO
	switch ctx := tree.(type) {
	case *parser.SourceContext:
		return p.VisitSource(ctx)
	case *parser.FunctionContext:
		return p.VisitFunction(ctx)
	case *parser.ParametersContext:
		return p.VisitParameters(ctx)
	case *parser.StatementContext:
		return p.VisitStatement(ctx)
	case *parser.IfContext:
		return p.VisitIf(ctx)
	case *parser.WhileContext:
		return p.VisitWhile(ctx)
	case *parser.ForContext:
		return p.VisitFor(ctx)
	case *parser.EachContext:
		return p.VisitEach(ctx)
	case *parser.IneachContext:
		return p.VisitIneach(ctx)
	case *parser.TryContext:
		return p.VisitTry(ctx)
	case *parser.DoContext:
		return p.VisitDo(ctx)
	case *parser.DeclContext:
		return p.VisitDecl(ctx)
	case *parser.ContinueContext:
		return p.VisitContinue(ctx)
	case *parser.BreakContext:
		return p.VisitBreak(ctx)
	case *parser.ReturnContext:
		return p.VisitReturn(ctx)
	case *parser.ThrowContext:
		return p.VisitThrow(ctx)
	case *parser.ExprContext:
		return p.VisitExpr(ctx)
	case *parser.TrailerContext:
		return p.VisitTrailer(ctx)
	case *parser.BlockContext:
		return p.VisitBlock(ctx)
	case *parser.EmptyContext:
		return p.VisitEmpty(ctx)
	case *parser.InitializerContext:
		return p.VisitInitializer(ctx)
	case *parser.AfterthoughtContext:
		return p.VisitAfterthought(ctx)
	case *parser.DeclarationContext:
		return p.VisitDeclaration(ctx)
	case *parser.DecltypeContext:
		return p.VisitDecltype(ctx)
	case *parser.TypeContext:
		return p.VisitType(ctx)
	case *parser.DeclvarContext:
		return p.VisitDeclvar(ctx)
	case *parser.TrapContext:
		return p.VisitTrap(ctx)
	case *parser.SingleContext:
		return p.VisitSingle(ctx)
	case *parser.CompContext:
		return p.VisitComp(ctx)
	case *parser.BoolContext:
		return p.VisitBool(ctx)
	case *parser.BinaryContext:
		return p.VisitBinary(ctx)
	case *parser.ElvisContext:
		return p.VisitElvis(ctx)
	case *parser.InstanceofContext:
		return p.VisitInstanceof(ctx)
	case *parser.NonconditionalContext:
		return p.VisitNonconditional(ctx)
	case *parser.ConditionalContext:
		return p.VisitConditional(ctx)
	case *parser.AssignmentContext:
		return p.VisitAssignment(ctx)
	case *parser.PreContext:
		return p.VisitPre(ctx)
	case *parser.AddsubContext:
		return p.VisitAddsub(ctx)
	case *parser.NotaddsubContext:
		return p.VisitNotaddsub(ctx)
	case *parser.ReadContext:
		return p.VisitRead(ctx)
	case *parser.PostContext:
		return p.VisitPost(ctx)
	case *parser.NotContext:
		return p.VisitNot(ctx)
	case *parser.CastContext:
		return p.VisitCast(ctx)
	case *parser.PrimordefcastContext:
		return p.VisitPrimordefcast(ctx)
	case *parser.RefcastContext:
		return p.VisitRefcast(ctx)
	case *parser.PrimordefcasttypeContext:
		return p.VisitPrimordefcasttype(ctx)
	case *parser.RefcasttypeContext:
		return p.VisitRefcasttype(ctx)
	case *parser.DynamicContext:
		return p.VisitDynamic(ctx)
	case *parser.NewarrayContext:
		return p.VisitNewarray(ctx)
	case *parser.PrecedenceContext:
		return p.VisitPrecedence(ctx)
	case *parser.NumericContext:
		return p.VisitNumeric(ctx)
	case *parser.TrueContext:
		return p.VisitTrue(ctx)
	case *parser.FalseContext:
		return p.VisitFalse(ctx)
	case *parser.NullContext:
		return p.VisitNull(ctx)
	case *parser.StringContext:
		return p.VisitString(ctx)
	case *parser.RegexContext:
		return p.VisitRegex(ctx)
	case *parser.ListinitContext:
		return p.VisitListinit(ctx)
	case *parser.MapinitContext:
		return p.VisitMapinit(ctx)
	case *parser.VariableContext:
		return p.VisitVariable(ctx)
	case *parser.CalllocalContext:
		return p.VisitCalllocal(ctx)
	case *parser.NewobjectContext:
		return p.VisitNewobject(ctx)
	case *parser.PostfixContext:
		return p.VisitPostfix(ctx)
	case *parser.PostdotContext:
		return p.VisitPostdot(ctx)
	case *parser.CallinvokeContext:
		return p.VisitCallinvoke(ctx)
	case *parser.FieldaccessContext:
		return p.VisitFieldaccess(ctx)
	case *parser.BraceaccessContext:
		return p.VisitBraceaccess(ctx)
	case *parser.NewstandardarrayContext:
		return p.VisitNewstandardarray(ctx)
	case *parser.NewinitializedarrayContext:
		return p.VisitNewinitializedarray(ctx)
	case *parser.ListinitializerContext:
		return p.VisitListinitializer(ctx)
	case *parser.MapinitializerContext:
		return p.VisitMapinitializer(ctx)
	case *parser.MaptokenContext:
		return p.VisitMaptoken(ctx)
	case *parser.ArgumentsContext:
		return p.VisitArguments(ctx)
	case *parser.ArgumentContext:
		return p.VisitArgument(ctx)
	case *parser.LambdaContext:
		return p.VisitLambda(ctx)
	case *parser.LamtypeContext:
		return p.VisitLamtype(ctx)
	case *parser.ClassfuncrefContext:
		return p.VisitClassfuncref(ctx)
	case *parser.ConstructorfuncrefContext:
		return p.VisitConstructorfuncref(ctx)
	case *parser.LocalfuncrefContext:
		return p.VisitLocalfuncref(ctx)
	case *antlr.TerminalNodeImpl:
		return p.VisitTerminal(ctx)
	default:
		panic(fmt.Errorf("unhandled type %T", ctx))
	}
}

func (p *painlessVisitor) VisitChildren(ctx antlr.RuleNode) interface{} {
	for _, child := range ctx.GetChildren() {
		tree, ok := child.(antlr.ParseTree)
		if !ok {
			return fmt.Errorf("unknown child node type: %T(%s)", child, child.GetPayload())
		}

		err := p.Visit(tree)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *painlessVisitor) VisitTerminal(ctx antlr.TerminalNode) interface{} {
	return nil
}

func (p *painlessVisitor) VisitErrorNode(node antlr.ErrorNode) interface{} {
	panic("VisitErrorNode")
}

func (p *painlessVisitor) VisitSource(ctx *parser.SourceContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitFunction(ctx *parser.FunctionContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitParameters(ctx *parser.ParametersContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitStatement(ctx *parser.StatementContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitIf(ctx *parser.IfContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitWhile(ctx *parser.WhileContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitFor(ctx *parser.ForContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitEach(ctx *parser.EachContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitIneach(ctx *parser.IneachContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitTry(ctx *parser.TryContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitDo(ctx *parser.DoContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitDecl(ctx *parser.DeclContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitContinue(ctx *parser.ContinueContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitBreak(ctx *parser.BreakContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitReturn(ctx *parser.ReturnContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitThrow(ctx *parser.ThrowContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitExpr(ctx *parser.ExprContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitTrailer(ctx *parser.TrailerContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitBlock(ctx *parser.BlockContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitEmpty(ctx *parser.EmptyContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitInitializer(ctx *parser.InitializerContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitAfterthought(ctx *parser.AfterthoughtContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitDeclaration(ctx *parser.DeclarationContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitDecltype(ctx *parser.DecltypeContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitType(ctx *parser.TypeContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitDeclvar(ctx *parser.DeclvarContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitTrap(ctx *parser.TrapContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitSingle(ctx *parser.SingleContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitComp(ctx *parser.CompContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitBool(ctx *parser.BoolContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitBinary(ctx *parser.BinaryContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitElvis(ctx *parser.ElvisContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitInstanceof(ctx *parser.InstanceofContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitNonconditional(ctx *parser.NonconditionalContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitConditional(ctx *parser.ConditionalContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitAssignment(ctx *parser.AssignmentContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitPre(ctx *parser.PreContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitAddsub(ctx *parser.AddsubContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitNotaddsub(ctx *parser.NotaddsubContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitRead(ctx *parser.ReadContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitPost(ctx *parser.PostContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitNot(ctx *parser.NotContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitCast(ctx *parser.CastContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitPrimordefcast(ctx *parser.PrimordefcastContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitRefcast(ctx *parser.RefcastContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitPrimordefcasttype(ctx *parser.PrimordefcasttypeContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitRefcasttype(ctx *parser.RefcasttypeContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitDynamic(ctx *parser.DynamicContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitNewarray(ctx *parser.NewarrayContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitPrecedence(ctx *parser.PrecedenceContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitNumeric(ctx *parser.NumericContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitTrue(ctx *parser.TrueContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitFalse(ctx *parser.FalseContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitNull(ctx *parser.NullContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitString(ctx *parser.StringContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitRegex(ctx *parser.RegexContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitListinit(ctx *parser.ListinitContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitMapinit(ctx *parser.MapinitContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitVariable(ctx *parser.VariableContext) interface{} {
	p.variables = append(p.variables, ctx.GetParent().(*parser.DynamicContext))
	return nil
}

func (p *painlessVisitor) VisitCalllocal(ctx *parser.CalllocalContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitNewobject(ctx *parser.NewobjectContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitPostfix(ctx *parser.PostfixContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitPostdot(ctx *parser.PostdotContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitCallinvoke(ctx *parser.CallinvokeContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitFieldaccess(ctx *parser.FieldaccessContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitBraceaccess(ctx *parser.BraceaccessContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitNewstandardarray(ctx *parser.NewstandardarrayContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitNewinitializedarray(ctx *parser.NewinitializedarrayContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitListinitializer(ctx *parser.ListinitializerContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitMapinitializer(ctx *parser.MapinitializerContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitMaptoken(ctx *parser.MaptokenContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitArguments(ctx *parser.ArgumentsContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitArgument(ctx *parser.ArgumentContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitLambda(ctx *parser.LambdaContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitLamtype(ctx *parser.LamtypeContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitClassfuncref(ctx *parser.ClassfuncrefContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitConstructorfuncref(ctx *parser.ConstructorfuncrefContext) interface{} {
	return p.VisitChildren(ctx)
}

func (p *painlessVisitor) VisitLocalfuncref(ctx *parser.LocalfuncrefContext) interface{} {
	return p.VisitChildren(ctx)
}
