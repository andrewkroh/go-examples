// Code generated from If.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // If

import "github.com/antlr4-go/antlr/v4"

type BaseIfVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseIfVisitor) VisitProgram(ctx *ProgramContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitStatement(ctx *StatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitIfStatement(ctx *IfStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitBoolExpressionPath(ctx *BoolExpressionPathContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitBoolExpressionBoolean(ctx *BoolExpressionBooleanContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitBoolExpressionNotEqual(ctx *BoolExpressionNotEqualContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitBoolExpressionEqual(ctx *BoolExpressionEqualContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitParenBoolExpression(ctx *ParenBoolExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitTerm(ctx *TermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitLiteral(ctx *LiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitPath(ctx *PathContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitString(ctx *StringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitNumber(ctx *NumberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitBoolean(ctx *BooleanContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseIfVisitor) VisitNil(ctx *NilContext) interface{} {
	return v.VisitChildren(ctx)
}
