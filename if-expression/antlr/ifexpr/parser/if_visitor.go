// Code generated from If.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // If

import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by IfParser.
type IfVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by IfParser#program.
	VisitProgram(ctx *ProgramContext) interface{}

	// Visit a parse tree produced by IfParser#statement.
	VisitStatement(ctx *StatementContext) interface{}

	// Visit a parse tree produced by IfParser#ifStatement.
	VisitIfStatement(ctx *IfStatementContext) interface{}

	// Visit a parse tree produced by IfParser#BoolExpressionPath.
	VisitBoolExpressionPath(ctx *BoolExpressionPathContext) interface{}

	// Visit a parse tree produced by IfParser#BoolExpressionBoolean.
	VisitBoolExpressionBoolean(ctx *BoolExpressionBooleanContext) interface{}

	// Visit a parse tree produced by IfParser#BoolExpressionNotEqual.
	VisitBoolExpressionNotEqual(ctx *BoolExpressionNotEqualContext) interface{}

	// Visit a parse tree produced by IfParser#BoolExpressionEqual.
	VisitBoolExpressionEqual(ctx *BoolExpressionEqualContext) interface{}

	// Visit a parse tree produced by IfParser#parenBoolExpression.
	VisitParenBoolExpression(ctx *ParenBoolExpressionContext) interface{}

	// Visit a parse tree produced by IfParser#term.
	VisitTerm(ctx *TermContext) interface{}

	// Visit a parse tree produced by IfParser#literal.
	VisitLiteral(ctx *LiteralContext) interface{}

	// Visit a parse tree produced by IfParser#path.
	VisitPath(ctx *PathContext) interface{}

	// Visit a parse tree produced by IfParser#string.
	VisitString(ctx *StringContext) interface{}

	// Visit a parse tree produced by IfParser#number.
	VisitNumber(ctx *NumberContext) interface{}

	// Visit a parse tree produced by IfParser#boolean.
	VisitBoolean(ctx *BooleanContext) interface{}

	// Visit a parse tree produced by IfParser#nil.
	VisitNil(ctx *NilContext) interface{}
}
