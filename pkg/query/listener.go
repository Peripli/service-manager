/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package query

import (
	"fmt"
	"strings"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query/parser"
)

type queryListener struct {
	*parser.BaseQueryListener
	err          error
	leftOp       string
	rightOp      []string
	op           string
	criteriaType CriterionType
	result       []Criterion
}

// ExitUnivariate is called when production univariate is exited.
func (s *queryListener) ExitUnivariate(ctx *parser.UnivariateContext) {
	if s.err != nil {
		return
	}
	leftOp, operator, rightOp := getCriterionFields(ctx.Key(), ctx.UniOp(), ctx.Value())
	s.leftOp = leftOp
	s.op = operator
	s.rightOp = []string{rightOp}
	s.err = s.storeCriterion()
}

// ExitManyValues is called when production manyValues is exited.
func (s *queryListener) ExitManyValues(ctx *parser.ManyValuesContext) {
	if s.err != nil {
		return
	}
	_, _, rightOp := getCriterionFields(nil, nil, ctx.Value())
	s.rightOp = append(s.rightOp, rightOp)
}

// ExitMultivariate is called when production univariate is exited.
func (s *queryListener) ExitMultivariate(ctx *parser.MultivariateContext) {
	if s.err != nil {
		return
	}

	leftOp, operator, _ := getCriterionFields(ctx.Key(), ctx.MultiOp(), nil)
	s.leftOp = leftOp
	s.op = operator
	if s.rightOp == nil {
		s.rightOp = []string{""}
	}
	s.err = s.storeCriterion()
}

func (s *queryListener) storeCriterion() error {
	operator, err := findOpByString(s.op)
	if err != nil {
		return err
	}
	criterion := NewCriterion(s.leftOp, operator, s.rightOp, s.criteriaType)
	if err = criterion.Validate(); err != nil {
		return err
	}
	s.result = append(s.result, criterion)
	s.rightOp = []string{}
	return nil
}

func (s *queryListener) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs antlr.ATNConfigSet) {
}

func (s *queryListener) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs antlr.ATNConfigSet) {
}

func (s *queryListener) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs antlr.ATNConfigSet) {
}

func (s *queryListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	s.err = &util.UnsupportedQueryError{Message: fmt.Sprintf("error while parsing %s at column %d: %s", s.criteriaType, column, msg)}
}

func getCriterionFields(key, op, right antlr.TerminalNode) (leftOp string, operator string, rightOp string) {
	if key != nil {
		leftOp = key.GetText()
	}
	if right != nil {
		rightOp = right.GetText()
		rightOp = strings.Replace(rightOp, "''", "'", -1)
		rightOp = strings.Trim(rightOp, "'")
	}
	if op != nil {
		operator = op.GetText()
	}
	return
}

func findOpByString(op string) (Operator, error) {
	for _, operator := range Operators {
		if operator.String() == op {
			return operator, nil
		}
	}
	return nil, fmt.Errorf("provided operator %s is not supported", op)
}
