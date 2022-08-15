// Generated from /Users/i355594/go/src/github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query/Query.g4 by ANTLR 4.7.

package parser // Query

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/antlr/antlr4/runtime/Go/antlr"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = reflect.Copy
var _ = strconv.Itoa

var parserATN = []uint16{
	3, 24715, 42794, 33075, 47597, 16764, 15335, 30598, 22884, 3, 12, 52, 4,
	2, 9, 2, 4, 3, 9, 3, 4, 4, 9, 4, 4, 5, 9, 5, 4, 6, 9, 6, 4, 7, 9, 7, 4,
	8, 9, 8, 3, 2, 3, 2, 3, 2, 3, 3, 3, 3, 3, 3, 5, 3, 23, 10, 3, 3, 4, 3,
	4, 5, 4, 27, 10, 4, 3, 5, 3, 5, 3, 5, 3, 5, 3, 5, 3, 5, 3, 6, 3, 6, 3,
	6, 3, 6, 3, 6, 3, 6, 3, 7, 3, 7, 5, 7, 43, 10, 7, 3, 7, 3, 7, 3, 8, 3,
	8, 3, 8, 5, 8, 50, 10, 8, 3, 8, 2, 2, 9, 2, 4, 6, 8, 10, 12, 14, 2, 2,
	2, 48, 2, 16, 3, 2, 2, 2, 4, 19, 3, 2, 2, 2, 6, 26, 3, 2, 2, 2, 8, 28,
	3, 2, 2, 2, 10, 34, 3, 2, 2, 2, 12, 40, 3, 2, 2, 2, 14, 46, 3, 2, 2, 2,
	16, 17, 5, 4, 3, 2, 17, 18, 7, 2, 2, 3, 18, 3, 3, 2, 2, 2, 19, 22, 5, 6,
	4, 2, 20, 21, 7, 5, 2, 2, 21, 23, 5, 4, 3, 2, 22, 20, 3, 2, 2, 2, 22, 23,
	3, 2, 2, 2, 23, 5, 3, 2, 2, 2, 24, 27, 5, 8, 5, 2, 25, 27, 5, 10, 6, 2,
	26, 24, 3, 2, 2, 2, 26, 25, 3, 2, 2, 2, 27, 7, 3, 2, 2, 2, 28, 29, 7, 8,
	2, 2, 29, 30, 7, 11, 2, 2, 30, 31, 7, 3, 2, 2, 31, 32, 7, 11, 2, 2, 32,
	33, 5, 12, 7, 2, 33, 9, 3, 2, 2, 2, 34, 35, 7, 8, 2, 2, 35, 36, 7, 11,
	2, 2, 36, 37, 7, 4, 2, 2, 37, 38, 7, 11, 2, 2, 38, 39, 7, 6, 2, 2, 39,
	11, 3, 2, 2, 2, 40, 42, 7, 9, 2, 2, 41, 43, 5, 14, 8, 2, 42, 41, 3, 2,
	2, 2, 42, 43, 3, 2, 2, 2, 43, 44, 3, 2, 2, 2, 44, 45, 7, 10, 2, 2, 45,
	13, 3, 2, 2, 2, 46, 49, 7, 6, 2, 2, 47, 48, 7, 7, 2, 2, 48, 50, 5, 14,
	8, 2, 49, 47, 3, 2, 2, 2, 49, 50, 3, 2, 2, 2, 50, 15, 3, 2, 2, 2, 6, 22,
	26, 42, 49,
}
var deserializer = antlr.NewATNDeserializer(nil)
var deserializedATN = deserializer.DeserializeFromUInt16(parserATN)

var literalNames = []string{
	"", "", "", "", "", "", "", "'('", "')'", "' '",
}
var symbolicNames = []string{
	"", "MultiOp", "UniOp", "Concat", "Value", "ValueSeparator", "Key", "OpenBracket",
	"CloseBracket", "Whitespace", "WS",
}

var ruleNames = []string{
	"expression", "criterions", "criterion", "multivariate", "univariate",
	"multiValues", "manyValues",
}
var decisionToDFA = make([]*antlr.DFA, len(deserializedATN.DecisionToState))

func init() {
	for index, ds := range deserializedATN.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(ds, index)
	}
}

type QueryParser struct {
	*antlr.BaseParser
}

func NewQueryParser(input antlr.TokenStream) *QueryParser {
	this := new(QueryParser)

	this.BaseParser = antlr.NewBaseParser(input)

	this.Interpreter = antlr.NewParserATNSimulator(this, deserializedATN, decisionToDFA, antlr.NewPredictionContextCache())
	this.RuleNames = ruleNames
	this.LiteralNames = literalNames
	this.SymbolicNames = symbolicNames
	this.GrammarFileName = "Query.g4"

	return this
}

// QueryParser tokens.
const (
	QueryParserEOF            = antlr.TokenEOF
	QueryParserMultiOp        = 1
	QueryParserUniOp          = 2
	QueryParserConcat         = 3
	QueryParserValue          = 4
	QueryParserValueSeparator = 5
	QueryParserKey            = 6
	QueryParserOpenBracket    = 7
	QueryParserCloseBracket   = 8
	QueryParserWhitespace     = 9
	QueryParserWS             = 10
)

// QueryParser rules.
const (
	QueryParserRULE_expression   = 0
	QueryParserRULE_criterions   = 1
	QueryParserRULE_criterion    = 2
	QueryParserRULE_multivariate = 3
	QueryParserRULE_univariate   = 4
	QueryParserRULE_multiValues  = 5
	QueryParserRULE_manyValues   = 6
)

// IExpressionContext is an interface to support dynamic dispatch.
type IExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsExpressionContext differentiates from other interfaces.
	IsExpressionContext()
}

type ExpressionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpressionContext() *ExpressionContext {
	var p = new(ExpressionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = QueryParserRULE_expression
	return p
}

func (*ExpressionContext) IsExpressionContext() {}

func NewExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpressionContext {
	var p = new(ExpressionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = QueryParserRULE_expression

	return p
}

func (s *ExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpressionContext) Criterions() ICriterionsContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*ICriterionsContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(ICriterionsContext)
}

func (s *ExpressionContext) EOF() antlr.TerminalNode {
	return s.GetToken(QueryParserEOF, 0)
}

func (s *ExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.EnterExpression(s)
	}
}

func (s *ExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.ExitExpression(s)
	}
}

func (p *QueryParser) Expression() (localctx IExpressionContext) {
	localctx = NewExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, QueryParserRULE_expression)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(14)
		p.Criterions()
	}
	{
		p.SetState(15)
		p.Match(QueryParserEOF)
	}

	return localctx
}

// ICriterionsContext is an interface to support dynamic dispatch.
type ICriterionsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsCriterionsContext differentiates from other interfaces.
	IsCriterionsContext()
}

type CriterionsContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCriterionsContext() *CriterionsContext {
	var p = new(CriterionsContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = QueryParserRULE_criterions
	return p
}

func (*CriterionsContext) IsCriterionsContext() {}

func NewCriterionsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CriterionsContext {
	var p = new(CriterionsContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = QueryParserRULE_criterions

	return p
}

func (s *CriterionsContext) GetParser() antlr.Parser { return s.parser }

func (s *CriterionsContext) Criterion() ICriterionContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*ICriterionContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(ICriterionContext)
}

func (s *CriterionsContext) Concat() antlr.TerminalNode {
	return s.GetToken(QueryParserConcat, 0)
}

func (s *CriterionsContext) Criterions() ICriterionsContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*ICriterionsContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(ICriterionsContext)
}

func (s *CriterionsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CriterionsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CriterionsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.EnterCriterions(s)
	}
}

func (s *CriterionsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.ExitCriterions(s)
	}
}

func (p *QueryParser) Criterions() (localctx ICriterionsContext) {
	localctx = NewCriterionsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, QueryParserRULE_criterions)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(17)
		p.Criterion()
	}
	p.SetState(20)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == QueryParserConcat {
		{
			p.SetState(18)
			p.Match(QueryParserConcat)
		}
		{
			p.SetState(19)
			p.Criterions()
		}

	}

	return localctx
}

// ICriterionContext is an interface to support dynamic dispatch.
type ICriterionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsCriterionContext differentiates from other interfaces.
	IsCriterionContext()
}

type CriterionContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCriterionContext() *CriterionContext {
	var p = new(CriterionContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = QueryParserRULE_criterion
	return p
}

func (*CriterionContext) IsCriterionContext() {}

func NewCriterionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CriterionContext {
	var p = new(CriterionContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = QueryParserRULE_criterion

	return p
}

func (s *CriterionContext) GetParser() antlr.Parser { return s.parser }

func (s *CriterionContext) Multivariate() IMultivariateContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IMultivariateContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IMultivariateContext)
}

func (s *CriterionContext) Univariate() IUnivariateContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IUnivariateContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IUnivariateContext)
}

func (s *CriterionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CriterionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CriterionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.EnterCriterion(s)
	}
}

func (s *CriterionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.ExitCriterion(s)
	}
}

func (p *QueryParser) Criterion() (localctx ICriterionContext) {
	localctx = NewCriterionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, QueryParserRULE_criterion)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.SetState(24)
	p.GetErrorHandler().Sync(p)
	switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 1, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(22)
			p.Multivariate()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(23)
			p.Univariate()
		}

	}

	return localctx
}

// IMultivariateContext is an interface to support dynamic dispatch.
type IMultivariateContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsMultivariateContext differentiates from other interfaces.
	IsMultivariateContext()
}

type MultivariateContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyMultivariateContext() *MultivariateContext {
	var p = new(MultivariateContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = QueryParserRULE_multivariate
	return p
}

func (*MultivariateContext) IsMultivariateContext() {}

func NewMultivariateContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *MultivariateContext {
	var p = new(MultivariateContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = QueryParserRULE_multivariate

	return p
}

func (s *MultivariateContext) GetParser() antlr.Parser { return s.parser }

func (s *MultivariateContext) Key() antlr.TerminalNode {
	return s.GetToken(QueryParserKey, 0)
}

func (s *MultivariateContext) AllWhitespace() []antlr.TerminalNode {
	return s.GetTokens(QueryParserWhitespace)
}

func (s *MultivariateContext) Whitespace(i int) antlr.TerminalNode {
	return s.GetToken(QueryParserWhitespace, i)
}

func (s *MultivariateContext) MultiOp() antlr.TerminalNode {
	return s.GetToken(QueryParserMultiOp, 0)
}

func (s *MultivariateContext) MultiValues() IMultiValuesContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IMultiValuesContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IMultiValuesContext)
}

func (s *MultivariateContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MultivariateContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *MultivariateContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.EnterMultivariate(s)
	}
}

func (s *MultivariateContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.ExitMultivariate(s)
	}
}

func (p *QueryParser) Multivariate() (localctx IMultivariateContext) {
	localctx = NewMultivariateContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, QueryParserRULE_multivariate)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(26)
		p.Match(QueryParserKey)
	}
	{
		p.SetState(27)
		p.Match(QueryParserWhitespace)
	}
	{
		p.SetState(28)
		p.Match(QueryParserMultiOp)
	}
	{
		p.SetState(29)
		p.Match(QueryParserWhitespace)
	}
	{
		p.SetState(30)
		p.MultiValues()
	}

	return localctx
}

// IUnivariateContext is an interface to support dynamic dispatch.
type IUnivariateContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsUnivariateContext differentiates from other interfaces.
	IsUnivariateContext()
}

type UnivariateContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUnivariateContext() *UnivariateContext {
	var p = new(UnivariateContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = QueryParserRULE_univariate
	return p
}

func (*UnivariateContext) IsUnivariateContext() {}

func NewUnivariateContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UnivariateContext {
	var p = new(UnivariateContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = QueryParserRULE_univariate

	return p
}

func (s *UnivariateContext) GetParser() antlr.Parser { return s.parser }

func (s *UnivariateContext) Key() antlr.TerminalNode {
	return s.GetToken(QueryParserKey, 0)
}

func (s *UnivariateContext) AllWhitespace() []antlr.TerminalNode {
	return s.GetTokens(QueryParserWhitespace)
}

func (s *UnivariateContext) Whitespace(i int) antlr.TerminalNode {
	return s.GetToken(QueryParserWhitespace, i)
}

func (s *UnivariateContext) UniOp() antlr.TerminalNode {
	return s.GetToken(QueryParserUniOp, 0)
}

func (s *UnivariateContext) Value() antlr.TerminalNode {
	return s.GetToken(QueryParserValue, 0)
}

func (s *UnivariateContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UnivariateContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *UnivariateContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.EnterUnivariate(s)
	}
}

func (s *UnivariateContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.ExitUnivariate(s)
	}
}

func (p *QueryParser) Univariate() (localctx IUnivariateContext) {
	localctx = NewUnivariateContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, QueryParserRULE_univariate)

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(32)
		p.Match(QueryParserKey)
	}
	{
		p.SetState(33)
		p.Match(QueryParserWhitespace)
	}
	{
		p.SetState(34)
		p.Match(QueryParserUniOp)
	}
	{
		p.SetState(35)
		p.Match(QueryParserWhitespace)
	}
	{
		p.SetState(36)
		p.Match(QueryParserValue)
	}

	return localctx
}

// IMultiValuesContext is an interface to support dynamic dispatch.
type IMultiValuesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsMultiValuesContext differentiates from other interfaces.
	IsMultiValuesContext()
}

type MultiValuesContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyMultiValuesContext() *MultiValuesContext {
	var p = new(MultiValuesContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = QueryParserRULE_multiValues
	return p
}

func (*MultiValuesContext) IsMultiValuesContext() {}

func NewMultiValuesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *MultiValuesContext {
	var p = new(MultiValuesContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = QueryParserRULE_multiValues

	return p
}

func (s *MultiValuesContext) GetParser() antlr.Parser { return s.parser }

func (s *MultiValuesContext) OpenBracket() antlr.TerminalNode {
	return s.GetToken(QueryParserOpenBracket, 0)
}

func (s *MultiValuesContext) CloseBracket() antlr.TerminalNode {
	return s.GetToken(QueryParserCloseBracket, 0)
}

func (s *MultiValuesContext) ManyValues() IManyValuesContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IManyValuesContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IManyValuesContext)
}

func (s *MultiValuesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MultiValuesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *MultiValuesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.EnterMultiValues(s)
	}
}

func (s *MultiValuesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.ExitMultiValues(s)
	}
}

func (p *QueryParser) MultiValues() (localctx IMultiValuesContext) {
	localctx = NewMultiValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, QueryParserRULE_multiValues)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(38)
		p.Match(QueryParserOpenBracket)
	}
	p.SetState(40)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == QueryParserValue {
		{
			p.SetState(39)
			p.ManyValues()
		}

	}
	{
		p.SetState(42)
		p.Match(QueryParserCloseBracket)
	}

	return localctx
}

// IManyValuesContext is an interface to support dynamic dispatch.
type IManyValuesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// IsManyValuesContext differentiates from other interfaces.
	IsManyValuesContext()
}

type ManyValuesContext struct {
	*antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyManyValuesContext() *ManyValuesContext {
	var p = new(ManyValuesContext)
	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(nil, -1)
	p.RuleIndex = QueryParserRULE_manyValues
	return p
}

func (*ManyValuesContext) IsManyValuesContext() {}

func NewManyValuesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ManyValuesContext {
	var p = new(ManyValuesContext)

	p.BaseParserRuleContext = antlr.NewBaseParserRuleContext(parent, invokingState)

	p.parser = parser
	p.RuleIndex = QueryParserRULE_manyValues

	return p
}

func (s *ManyValuesContext) GetParser() antlr.Parser { return s.parser }

func (s *ManyValuesContext) Value() antlr.TerminalNode {
	return s.GetToken(QueryParserValue, 0)
}

func (s *ManyValuesContext) ValueSeparator() antlr.TerminalNode {
	return s.GetToken(QueryParserValueSeparator, 0)
}

func (s *ManyValuesContext) ManyValues() IManyValuesContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IManyValuesContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IManyValuesContext)
}

func (s *ManyValuesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ManyValuesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ManyValuesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.EnterManyValues(s)
	}
}

func (s *ManyValuesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(QueryListener); ok {
		listenerT.ExitManyValues(s)
	}
}

func (p *QueryParser) ManyValues() (localctx IManyValuesContext) {
	localctx = NewManyValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, QueryParserRULE_manyValues)
	var _la int

	defer func() {
		p.ExitRule()
	}()

	defer func() {
		if err := recover(); err != nil {
			if v, ok := err.(antlr.RecognitionException); ok {
				localctx.SetException(v)
				p.GetErrorHandler().ReportError(p, v)
				p.GetErrorHandler().Recover(p, v)
			} else {
				panic(err)
			}
		}
	}()

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(44)
		p.Match(QueryParserValue)
	}
	p.SetState(47)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)

	if _la == QueryParserValueSeparator {
		{
			p.SetState(45)
			p.Match(QueryParserValueSeparator)
		}
		{
			p.SetState(46)
			p.ManyValues()
		}

	}

	return localctx
}
