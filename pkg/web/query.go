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

package web

import (
	"fmt"
	"strings"
)

type Operator string

const (
	EqualsOperator    Operator = "="
	NotEqualsOperator Operator = "!="
	InOperator        Operator = "in"
	NotInOperator     Operator = "notin"
)

func (op Operator) IsMultiVariate() bool {
	return op == InOperator || op == NotInOperator
}

var operators = []Operator{EqualsOperator, NotEqualsOperator, InOperator, NotInOperator}

var supportedQueryMatchers = []string{"fieldQuery", "labelQuery"}
var allowedSeparators = []rune{';'}

type QuerySegment struct {
	LeftOp   string
	Operator Operator
	RightOp  []string
}

func newQuerySegment(leftOp string, operator Operator, rightOp []string) QuerySegment {
	return QuerySegment{LeftOp: leftOp, Operator: operator, RightOp: rightOp}
}

func (qs QuerySegment) Validate() error {
	if len(qs.RightOp) > 1 && !qs.Operator.IsMultiVariate() {
		return fmt.Errorf("multiple values received for single value operation")
	}
	return nil
}

func BuildFilterSegmentsForRequest(request *Request) ([]QuerySegment, error) {
	var result []QuerySegment
	for _, queryMatcher := range supportedQueryMatchers {
		queryValues := request.URL.Query()[queryMatcher]
		querySegments, err := process(queryValues)
		if err != nil {
			return nil, err
		}
		result = append(result, querySegments...)
	}
	return result, nil
}

func process(values []string) ([]QuerySegment, error) {
	querySegments := make([]QuerySegment, 0)
	for _, input := range values {
		rawFilterStatements := strings.FieldsFunc(input, split)
		for _, rawStatement := range rawFilterStatements {
			op, err := getRawOperation(rawStatement)
			if err != nil {
				return nil, err
			}

			querySegment := convertRawStatementToFilterStatement(rawStatement, op)
			if err := querySegment.Validate(); err != nil {
				return nil, err
			}
			querySegments = append(querySegments, querySegment)
		}
	}
	return querySegments, nil
}

func split(r rune) bool {
	for _, sep := range allowedSeparators {
		if r == sep {
			return true
		}
	}
	return false
}

func getRawOperation(rawStatement string) (Operator, error) {
	opIdx := -1
	for _, op := range operators {
		//TODO: look for Operand"+Operation+"Operands
		opIdx = strings.Index(rawStatement, string(op))
		if opIdx != -1 {
			return op, nil
		}
	}
	return "", fmt.Errorf("label query operator is missing")
}

func convertRawStatementToFilterStatement(rawStatement string, op Operator) QuerySegment {
	opIdx := strings.Index(rawStatement, string(op))
	rightOp := strings.Split(rawStatement[opIdx+len(op):], ",")

	if op.IsMultiVariate() {
		rightOp[0] = strings.Trim(rightOp[0], "[")
		rightOp[len(rightOp)-1] = strings.Trim(rightOp[len(rightOp)-1], "]")
	}

	return newQuerySegment(rawStatement[:opIdx], op, rightOp)
}
