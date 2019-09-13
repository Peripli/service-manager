// Code generated from /Users/i322053/goworkspace/src/github.com/Peripli/service-manager/pkg/query/Query.g4 by ANTLR 4.7.2. DO NOT EDIT.

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
	3, 24715, 42794, 33075, 47597, 16764, 15335, 30598, 22884, 3, 22, 51, 4, 
	2, 9, 2, 4, 3, 9, 3, 4, 4, 9, 4, 4, 5, 9, 5, 4, 6, 9, 6, 4, 7, 9, 7, 3, 
	2, 3, 2, 3, 2, 3, 2, 3, 2, 5, 2, 20, 10, 2, 3, 2, 3, 2, 3, 3, 3, 3, 5, 
	3, 26, 10, 3, 3, 4, 3, 4, 3, 4, 3, 4, 3, 4, 3, 4, 3, 5, 3, 5, 3, 5, 3, 
	5, 3, 5, 3, 5, 3, 6, 3, 6, 5, 6, 42, 10, 6, 3, 6, 3, 6, 3, 7, 3, 7, 3, 
	7, 5, 7, 49, 10, 7, 3, 7, 2, 2, 8, 2, 4, 6, 8, 10, 12, 2, 2, 2, 48, 2, 
	14, 3, 2, 2, 2, 4, 25, 3, 2, 2, 2, 6, 27, 3, 2, 2, 2, 8, 33, 3, 2, 2, 2, 
	10, 39, 3, 2, 2, 2, 12, 45, 3, 2, 2, 2, 14, 19, 5, 4, 3, 2, 15, 16, 7, 
	21, 2, 2, 16, 17, 7, 3, 2, 2, 17, 18, 7, 21, 2, 2, 18, 20, 5, 2, 2, 2, 
	19, 15, 3, 2, 2, 2, 19, 20, 3, 2, 2, 2, 20, 21, 3, 2, 2, 2, 21, 22, 7, 
	2, 2, 3, 22, 3, 3, 2, 2, 2, 23, 26, 5, 6, 4, 2, 24, 26, 5, 8, 5, 2, 25, 
	23, 3, 2, 2, 2, 25, 24, 3, 2, 2, 2, 26, 5, 3, 2, 2, 2, 27, 28, 7, 19, 2, 
	2, 28, 29, 7, 21, 2, 2, 29, 30, 7, 6, 2, 2, 30, 31, 7, 21, 2, 2, 31, 32, 
	5, 10, 6, 2, 32, 7, 3, 2, 2, 2, 33, 34, 7, 19, 2, 2, 34, 35, 7, 21, 2, 
	2, 35, 36, 7, 7, 2, 2, 36, 37, 7, 21, 2, 2, 37, 38, 7, 8, 2, 2, 38, 9, 
	3, 2, 2, 2, 39, 41, 7, 4, 2, 2, 40, 42, 5, 12, 7, 2, 41, 40, 3, 2, 2, 2, 
	41, 42, 3, 2, 2, 2, 42, 43, 3, 2, 2, 2, 43, 44, 7, 5, 2, 2, 44, 11, 3, 
	2, 2, 2, 45, 48, 7, 8, 2, 2, 46, 47, 7, 20, 2, 2, 47, 49, 5, 12, 7, 2, 
	48, 46, 3, 2, 2, 2, 48, 49, 3, 2, 2, 2, 49, 13, 3, 2, 2, 2, 6, 19, 25, 
	41, 48,
}
var deserializer = antlr.NewATNDeserializer(nil)
var deserializedATN = deserializer.DeserializeFromUInt16(parserATN)

var literalNames = []string{
	"", "'and'", "'('", "')'", "", "", "", "", "", "", "", "", "", "", "", 
	"", "", "", "", "' '",
}
var symbolicNames = []string{
	"", "", "", "", "MultiOp", "UniOp", "Value", "STRING", "BOOLEAN", "NUMBER", 
	"SIGN", "DATETIME", "DIGIT", "INTEGER", "TWO_DIGITS", "FOUR_DIGITS", "FIVE_DIGITS", 
	"Key", "ValueSeparator", "Whitespace", "WS",
}

var ruleNames = []string{
	"expression", "criterion", "multivariate", "univariate", "multiValues", 
	"manyValues",
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
	QueryParserEOF = antlr.TokenEOF
	QueryParserT__0 = 1
	QueryParserT__1 = 2
	QueryParserT__2 = 3
	QueryParserMultiOp = 4
	QueryParserUniOp = 5
	QueryParserValue = 6
	QueryParserSTRING = 7
	QueryParserBOOLEAN = 8
	QueryParserNUMBER = 9
	QueryParserSIGN = 10
	QueryParserDATETIME = 11
	QueryParserDIGIT = 12
	QueryParserINTEGER = 13
	QueryParserTWO_DIGITS = 14
	QueryParserFOUR_DIGITS = 15
	QueryParserFIVE_DIGITS = 16
	QueryParserKey = 17
	QueryParserValueSeparator = 18
	QueryParserWhitespace = 19
	QueryParserWS = 20
)

// QueryParser rules.
const (
	QueryParserRULE_expression = 0
	QueryParserRULE_criterion = 1
	QueryParserRULE_multivariate = 2
	QueryParserRULE_univariate = 3
	QueryParserRULE_multiValues = 4
	QueryParserRULE_manyValues = 5
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

func (s *ExpressionContext) Criterion() ICriterionContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*ICriterionContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(ICriterionContext)
}

func (s *ExpressionContext) EOF() antlr.TerminalNode {
	return s.GetToken(QueryParserEOF, 0)
}

func (s *ExpressionContext) AllWhitespace() []antlr.TerminalNode {
	return s.GetTokens(QueryParserWhitespace)
}

func (s *ExpressionContext) Whitespace(i int) antlr.TerminalNode {
	return s.GetToken(QueryParserWhitespace, i)
}

func (s *ExpressionContext) Expression() IExpressionContext {
	var t = s.GetTypedRuleContext(reflect.TypeOf((*IExpressionContext)(nil)).Elem(), 0)

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
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
		p.SetState(12)
		p.Criterion()
	}
	p.SetState(17)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)


	if _la == QueryParserWhitespace {
		{
			p.SetState(13)
			p.Match(QueryParserWhitespace)
		}
		{
			p.SetState(14)
			p.Match(QueryParserT__0)
		}
		{
			p.SetState(15)
			p.Match(QueryParserWhitespace)
		}
		{
			p.SetState(16)
			p.Expression()
		}

	}
	{
		p.SetState(19)
		p.Match(QueryParserEOF)
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
	p.EnterRule(localctx, 2, QueryParserRULE_criterion)

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

	p.SetState(23)
	p.GetErrorHandler().Sync(p)
	switch p.GetInterpreter().AdaptivePredict(p.GetTokenStream(), 1, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(21)
			p.Multivariate()
		}


	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(22)
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
	p.EnterRule(localctx, 4, QueryParserRULE_multivariate)

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
		p.SetState(25)
		p.Match(QueryParserKey)
	}
	{
		p.SetState(26)
		p.Match(QueryParserWhitespace)
	}
	{
		p.SetState(27)
		p.Match(QueryParserMultiOp)
	}
	{
		p.SetState(28)
		p.Match(QueryParserWhitespace)
	}
	{
		p.SetState(29)
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
	p.EnterRule(localctx, 6, QueryParserRULE_univariate)

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
		p.SetState(31)
		p.Match(QueryParserKey)
	}
	{
		p.SetState(32)
		p.Match(QueryParserWhitespace)
	}
	{
		p.SetState(33)
		p.Match(QueryParserUniOp)
	}
	{
		p.SetState(34)
		p.Match(QueryParserWhitespace)
	}
	{
		p.SetState(35)
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
	p.EnterRule(localctx, 8, QueryParserRULE_multiValues)
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
		p.SetState(37)
		p.Match(QueryParserT__1)
	}
	p.SetState(39)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)


	if _la == QueryParserValue {
		{
			p.SetState(38)
			p.ManyValues()
		}

	}
	{
		p.SetState(41)
		p.Match(QueryParserT__2)
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
	p.EnterRule(localctx, 10, QueryParserRULE_manyValues)
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
		p.SetState(43)
		p.Match(QueryParserValue)
	}
	p.SetState(46)
	p.GetErrorHandler().Sync(p)
	_la = p.GetTokenStream().LA(1)


	if _la == QueryParserValueSeparator {
		{
			p.SetState(44)
			p.Match(QueryParserValueSeparator)
		}
		{
			p.SetState(45)
			p.ManyValues()
		}

	}



	return localctx
}


