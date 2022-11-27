// Code generated from java-escape by ANTLR 4.11.1. DO NOT EDIT.

package parser // If

import "github.com/antlr/antlr4/runtime/Go/antlr/v4"

// IfListener is a complete listener for a parse tree produced by IfParser.
type IfListener interface {
	antlr.ParseTreeListener

	// EnterProgram is called when entering the program production.
	EnterProgram(c *ProgramContext)

	// EnterStatement is called when entering the statement production.
	EnterStatement(c *StatementContext)

	// EnterIfStatement is called when entering the ifStatement production.
	EnterIfStatement(c *IfStatementContext)

	// EnterBoolExpressionPath is called when entering the BoolExpressionPath production.
	EnterBoolExpressionPath(c *BoolExpressionPathContext)

	// EnterBoolExpressionBoolean is called when entering the BoolExpressionBoolean production.
	EnterBoolExpressionBoolean(c *BoolExpressionBooleanContext)

	// EnterBoolExpressionNotEqual is called when entering the BoolExpressionNotEqual production.
	EnterBoolExpressionNotEqual(c *BoolExpressionNotEqualContext)

	// EnterBoolExpressionEqual is called when entering the BoolExpressionEqual production.
	EnterBoolExpressionEqual(c *BoolExpressionEqualContext)

	// EnterParenBoolExpression is called when entering the parenBoolExpression production.
	EnterParenBoolExpression(c *ParenBoolExpressionContext)

	// EnterTerm is called when entering the term production.
	EnterTerm(c *TermContext)

	// EnterLiteral is called when entering the literal production.
	EnterLiteral(c *LiteralContext)

	// EnterPath is called when entering the path production.
	EnterPath(c *PathContext)

	// EnterString is called when entering the string production.
	EnterString(c *StringContext)

	// EnterNumber is called when entering the number production.
	EnterNumber(c *NumberContext)

	// EnterBoolean is called when entering the boolean production.
	EnterBoolean(c *BooleanContext)

	// EnterNil is called when entering the nil production.
	EnterNil(c *NilContext)

	// ExitProgram is called when exiting the program production.
	ExitProgram(c *ProgramContext)

	// ExitStatement is called when exiting the statement production.
	ExitStatement(c *StatementContext)

	// ExitIfStatement is called when exiting the ifStatement production.
	ExitIfStatement(c *IfStatementContext)

	// ExitBoolExpressionPath is called when exiting the BoolExpressionPath production.
	ExitBoolExpressionPath(c *BoolExpressionPathContext)

	// ExitBoolExpressionBoolean is called when exiting the BoolExpressionBoolean production.
	ExitBoolExpressionBoolean(c *BoolExpressionBooleanContext)

	// ExitBoolExpressionNotEqual is called when exiting the BoolExpressionNotEqual production.
	ExitBoolExpressionNotEqual(c *BoolExpressionNotEqualContext)

	// ExitBoolExpressionEqual is called when exiting the BoolExpressionEqual production.
	ExitBoolExpressionEqual(c *BoolExpressionEqualContext)

	// ExitParenBoolExpression is called when exiting the parenBoolExpression production.
	ExitParenBoolExpression(c *ParenBoolExpressionContext)

	// ExitTerm is called when exiting the term production.
	ExitTerm(c *TermContext)

	// ExitLiteral is called when exiting the literal production.
	ExitLiteral(c *LiteralContext)

	// ExitPath is called when exiting the path production.
	ExitPath(c *PathContext)

	// ExitString is called when exiting the string production.
	ExitString(c *StringContext)

	// ExitNumber is called when exiting the number production.
	ExitNumber(c *NumberContext)

	// ExitBoolean is called when exiting the boolean production.
	ExitBoolean(c *BooleanContext)

	// ExitNil is called when exiting the nil production.
	ExitNil(c *NilContext)
}
