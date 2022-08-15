// Generated from /Users/i355594/go/src/github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query/Query.g4 by ANTLR 4.7.

package parser // Query

import "github.com/antlr/antlr4/runtime/Go/antlr"

// BaseQueryListener is a complete listener for a parse tree produced by QueryParser.
type BaseQueryListener struct{}

var _ QueryListener = &BaseQueryListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseQueryListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseQueryListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseQueryListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseQueryListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseQueryListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseQueryListener) ExitExpression(ctx *ExpressionContext) {}

// EnterCriterions is called when production criterions is entered.
func (s *BaseQueryListener) EnterCriterions(ctx *CriterionsContext) {}

// ExitCriterions is called when production criterions is exited.
func (s *BaseQueryListener) ExitCriterions(ctx *CriterionsContext) {}

// EnterCriterion is called when production criterion is entered.
func (s *BaseQueryListener) EnterCriterion(ctx *CriterionContext) {}

// ExitCriterion is called when production criterion is exited.
func (s *BaseQueryListener) ExitCriterion(ctx *CriterionContext) {}

// EnterMultivariate is called when production multivariate is entered.
func (s *BaseQueryListener) EnterMultivariate(ctx *MultivariateContext) {}

// ExitMultivariate is called when production multivariate is exited.
func (s *BaseQueryListener) ExitMultivariate(ctx *MultivariateContext) {}

// EnterUnivariate is called when production univariate is entered.
func (s *BaseQueryListener) EnterUnivariate(ctx *UnivariateContext) {}

// ExitUnivariate is called when production univariate is exited.
func (s *BaseQueryListener) ExitUnivariate(ctx *UnivariateContext) {}

// EnterMultiValues is called when production multiValues is entered.
func (s *BaseQueryListener) EnterMultiValues(ctx *MultiValuesContext) {}

// ExitMultiValues is called when production multiValues is exited.
func (s *BaseQueryListener) ExitMultiValues(ctx *MultiValuesContext) {}

// EnterManyValues is called when production manyValues is entered.
func (s *BaseQueryListener) EnterManyValues(ctx *ManyValuesContext) {}

// ExitManyValues is called when production manyValues is exited.
func (s *BaseQueryListener) ExitManyValues(ctx *ManyValuesContext) {}
