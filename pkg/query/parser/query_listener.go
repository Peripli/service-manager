// Generated from /Users/i355594/go/src/github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query/Query.g4 by ANTLR 4.7.

package parser // Query

import "github.com/antlr/antlr4/runtime/Go/antlr"

// QueryListener is a complete listener for a parse tree produced by QueryParser.
type QueryListener interface {
	antlr.ParseTreeListener

	// EnterExpression is called when entering the expression production.
	EnterExpression(c *ExpressionContext)

	// EnterCriterions is called when entering the criterions production.
	EnterCriterions(c *CriterionsContext)

	// EnterCriterion is called when entering the criterion production.
	EnterCriterion(c *CriterionContext)

	// EnterMultivariate is called when entering the multivariate production.
	EnterMultivariate(c *MultivariateContext)

	// EnterUnivariate is called when entering the univariate production.
	EnterUnivariate(c *UnivariateContext)

	// EnterMultiValues is called when entering the multiValues production.
	EnterMultiValues(c *MultiValuesContext)

	// EnterManyValues is called when entering the manyValues production.
	EnterManyValues(c *ManyValuesContext)

	// ExitExpression is called when exiting the expression production.
	ExitExpression(c *ExpressionContext)

	// ExitCriterions is called when exiting the criterions production.
	ExitCriterions(c *CriterionsContext)

	// ExitCriterion is called when exiting the criterion production.
	ExitCriterion(c *CriterionContext)

	// ExitMultivariate is called when exiting the multivariate production.
	ExitMultivariate(c *MultivariateContext)

	// ExitUnivariate is called when exiting the univariate production.
	ExitUnivariate(c *UnivariateContext)

	// ExitMultiValues is called when exiting the multiValues production.
	ExitMultiValues(c *MultiValuesContext)

	// ExitManyValues is called when exiting the manyValues production.
	ExitManyValues(c *ManyValuesContext)
}
