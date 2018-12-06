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

package selection

import (
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"
)

type Operator string

const (
	EqualsOperator    Operator = "="
	NotEqualsOperator Operator = "!="
	InOperator        Operator = "IN"
	NotInOperator     Operator = "NOTIN"
)

func (op Operator) IsMultiVariate() bool {
	return op == InOperator || op == NotInOperator
}

var operators = []Operator{EqualsOperator, NotEqualsOperator, InOperator, NotInOperator}

type CriteriaType string

const (
	FieldQuery CriteriaType = "fieldQuery"
	LabelQuery CriteriaType = "labelQuery"
)

var supportedQueryTypes = []CriteriaType{FieldQuery, LabelQuery}
var allowedSeparators = []rune{';'}

type Criteria struct {
	LeftOp   string
	Operator Operator
	RightOp  []string
	Type     CriteriaType
}

func newCriteria(leftOp string, operator Operator, rightOp []string, criteriaType CriteriaType) Criteria {
	return Criteria{LeftOp: leftOp, Operator: operator, RightOp: rightOp, Type: criteriaType}
}

func (qs Criteria) Validate() error {
	if len(qs.RightOp) > 1 && !qs.Operator.IsMultiVariate() {
		return fmt.Errorf("multiple values received for single value operation")
	}
	return nil
}

func BuildQuerySegmentsForRequest(request *web.Request) ([]Criteria, error) {
	var result []Criteria
	for _, queryType := range supportedQueryTypes {
		queryValues := request.URL.Query().Get(string(queryType))
		querySegments, err := process(queryValues, queryType)
		if err != nil {
			return nil, err
		}
		result = append(result, querySegments...)
	}
	return result, nil
}

func process(input string, criteriaType CriteriaType) ([]Criteria, error) {
	querySegments := make([]Criteria, 0)
	//for _, input := range values {
	rawFilterStatements := strings.FieldsFunc(input, split)
	for _, rawStatement := range rawFilterStatements {
		op, err := getRawOperation(rawStatement)
		if err != nil {
			return nil, err
		}

		querySegment := convertRawStatementToFilterStatement(rawStatement, op, criteriaType)
		if err := querySegment.Validate(); err != nil {
			return nil, err
		}
		querySegments = append(querySegments, querySegment)
	}
	//}
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

func convertRawStatementToFilterStatement(rawStatement string, op Operator, criteriaType CriteriaType) Criteria {
	opIdx := strings.Index(rawStatement, string(op))
	rightOp := strings.Split(rawStatement[opIdx+len(op):], ",")

	if op.IsMultiVariate() {
		rightOp[0] = strings.TrimPrefix(strings.TrimSpace(rightOp[0]), "[")
		rightOp[len(rightOp)-1] = strings.TrimSuffix(strings.TrimSpace(rightOp[len(rightOp)-1]), "]")
	}
	return newCriteria(rawStatement[:opIdx], op, rightOp, criteriaType)
}
