// Code generated from java-escape by ANTLR 4.11.1. DO NOT EDIT.

package parser // If

import "github.com/antlr/antlr4/runtime/Go/antlr/v4"

// BaseIfListener is a complete listener for a parse tree produced by IfParser.
type BaseIfListener struct{}

var _ IfListener = &BaseIfListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseIfListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseIfListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseIfListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseIfListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterProgram is called when production program is entered.
func (s *BaseIfListener) EnterProgram(ctx *ProgramContext) {}

// ExitProgram is called when production program is exited.
func (s *BaseIfListener) ExitProgram(ctx *ProgramContext) {}

// EnterStatement is called when production statement is entered.
func (s *BaseIfListener) EnterStatement(ctx *StatementContext) {}

// ExitStatement is called when production statement is exited.
func (s *BaseIfListener) ExitStatement(ctx *StatementContext) {}

// EnterIfStatement is called when production ifStatement is entered.
func (s *BaseIfListener) EnterIfStatement(ctx *IfStatementContext) {}

// ExitIfStatement is called when production ifStatement is exited.
func (s *BaseIfListener) ExitIfStatement(ctx *IfStatementContext) {}

// EnterBoolExpressionPath is called when production BoolExpressionPath is entered.
func (s *BaseIfListener) EnterBoolExpressionPath(ctx *BoolExpressionPathContext) {}

// ExitBoolExpressionPath is called when production BoolExpressionPath is exited.
func (s *BaseIfListener) ExitBoolExpressionPath(ctx *BoolExpressionPathContext) {}

// EnterBoolExpressionBoolean is called when production BoolExpressionBoolean is entered.
func (s *BaseIfListener) EnterBoolExpressionBoolean(ctx *BoolExpressionBooleanContext) {}

// ExitBoolExpressionBoolean is called when production BoolExpressionBoolean is exited.
func (s *BaseIfListener) ExitBoolExpressionBoolean(ctx *BoolExpressionBooleanContext) {}

// EnterBoolExpressionNotEqual is called when production BoolExpressionNotEqual is entered.
func (s *BaseIfListener) EnterBoolExpressionNotEqual(ctx *BoolExpressionNotEqualContext) {}

// ExitBoolExpressionNotEqual is called when production BoolExpressionNotEqual is exited.
func (s *BaseIfListener) ExitBoolExpressionNotEqual(ctx *BoolExpressionNotEqualContext) {}

// EnterBoolExpressionEqual is called when production BoolExpressionEqual is entered.
func (s *BaseIfListener) EnterBoolExpressionEqual(ctx *BoolExpressionEqualContext) {}

// ExitBoolExpressionEqual is called when production BoolExpressionEqual is exited.
func (s *BaseIfListener) ExitBoolExpressionEqual(ctx *BoolExpressionEqualContext) {}

// EnterParenBoolExpression is called when production parenBoolExpression is entered.
func (s *BaseIfListener) EnterParenBoolExpression(ctx *ParenBoolExpressionContext) {}

// ExitParenBoolExpression is called when production parenBoolExpression is exited.
func (s *BaseIfListener) ExitParenBoolExpression(ctx *ParenBoolExpressionContext) {}

// EnterTerm is called when production term is entered.
func (s *BaseIfListener) EnterTerm(ctx *TermContext) {}

// ExitTerm is called when production term is exited.
func (s *BaseIfListener) ExitTerm(ctx *TermContext) {}

// EnterLiteral is called when production literal is entered.
func (s *BaseIfListener) EnterLiteral(ctx *LiteralContext) {}

// ExitLiteral is called when production literal is exited.
func (s *BaseIfListener) ExitLiteral(ctx *LiteralContext) {}

// EnterPath is called when production path is entered.
func (s *BaseIfListener) EnterPath(ctx *PathContext) {}

// ExitPath is called when production path is exited.
func (s *BaseIfListener) ExitPath(ctx *PathContext) {}

// EnterString is called when production string is entered.
func (s *BaseIfListener) EnterString(ctx *StringContext) {}

// ExitString is called when production string is exited.
func (s *BaseIfListener) ExitString(ctx *StringContext) {}

// EnterNumber is called when production number is entered.
func (s *BaseIfListener) EnterNumber(ctx *NumberContext) {}

// ExitNumber is called when production number is exited.
func (s *BaseIfListener) ExitNumber(ctx *NumberContext) {}

// EnterBoolean is called when production boolean is entered.
func (s *BaseIfListener) EnterBoolean(ctx *BooleanContext) {}

// ExitBoolean is called when production boolean is exited.
func (s *BaseIfListener) ExitBoolean(ctx *BooleanContext) {}

// EnterNil is called when production nil is entered.
func (s *BaseIfListener) EnterNil(ctx *NilContext) {}

// ExitNil is called when production nil is exited.
func (s *BaseIfListener) ExitNil(ctx *NilContext) {}
