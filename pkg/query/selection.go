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
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Peripli/service-manager/pkg/util/slice"

	"github.com/Peripli/service-manager/pkg/web"
)

// Operator is a query operator
type Operator string

const (
	// EqualsOperator takes two operands and tests if they are equal
	EqualsOperator Operator = "="
	// NotEqualsOperator takes two operands and tests if they are not equal
	NotEqualsOperator Operator = "!="
	// GreaterThanOperator takes two operands and tests if the left is greater than the right
	GreaterThanOperator Operator = "gt"
	// LessThanOperator takes two operands and tests if the left is lesser than the right
	LessThanOperator Operator = "lt"
	// InOperator takes two operands and tests if the left is contained in the right
	InOperator Operator = "in"
	// NotInOperator takes two operands and tests if the left is not contained in the right
	NotInOperator Operator = "notin"
	// EqualsOrNilOperator takes two operands and tests if the left is equal to the right, or if the left is nil
	EqualsOrNilOperator Operator = "eqornil"
)

// IsMultiVariate returns true if the operator requires right operand with multiple values
func (op Operator) IsMultiVariate() bool {
	return op == InOperator || op == NotInOperator
}

// IsNullable returns true if the operator can check if the left operand is nil
func (op Operator) IsNullable() bool {
	return op == EqualsOrNilOperator
}

// IsNumeric returns true if the operator works only with numeric operands
func (op Operator) IsNumeric() bool {
	return op == LessThanOperator || op == GreaterThanOperator
}

var operators = []Operator{EqualsOperator, NotEqualsOperator, InOperator,
	NotInOperator, GreaterThanOperator, LessThanOperator, EqualsOrNilOperator}

const (
	// OpenBracket is the token that denotes the beginning of a multivariate operand
	OpenBracket string = "["
	// OpenBracket is the token that denotes the end of a multivariate operand
	CloseBracket string = "]"
)

// CriterionType is a type of criteria to be applied when querying
type CriterionType string

const (
	// FieldQuery denotes that the query should be executed on the entity's fields
	FieldQuery CriterionType = "fieldQuery"
	// LabelQuery denotes that the query should be executed on the entity's labels
	LabelQuery CriterionType = "labelQuery"
)

var supportedQueryTypes = []CriterionType{FieldQuery, LabelQuery}

// UnsupportedQueryError is an error to show that the provided query cannot be executed
type UnsupportedQueryError struct {
	Message string
}

func (uq *UnsupportedQueryError) Error() string {
	return uq.Message
}

// Criterion is a single part of a query criteria
type Criterion struct {
	// LeftOp is the left operand in the query
	LeftOp string
	// Operator is the query operator
	Operator Operator
	// RightOp is the right operand in the query which can be multivariate
	RightOp []string
	// Type is the type of the query
	Type CriterionType
}

// ByField constructs a new criterion for field querying
func ByField(operator Operator, leftOp string, rightOp ...string) Criterion {
	return newCriterion(leftOp, operator, rightOp, FieldQuery)
}

// ByLabel constructs a new criterion for label querying
func ByLabel(operator Operator, leftOp string, rightOp ...string) Criterion {
	return newCriterion(leftOp, operator, rightOp, LabelQuery)
}

func newCriterion(leftOp string, operator Operator, rightOp []string, criteriaType CriterionType) Criterion {
	return Criterion{LeftOp: leftOp, Operator: operator, RightOp: rightOp, Type: criteriaType}
}

func (c Criterion) Validate() error {
	if len(c.RightOp) > 1 && !c.Operator.IsMultiVariate() {
		return fmt.Errorf("multiple values received for single value operation")
	}
	if c.Operator.IsNullable() && c.Type != FieldQuery {
		return &UnsupportedQueryError{"nullable operations are supported only for field queries"}
	}
	if c.Operator.IsNumeric() && !isNumeric(c.RightOp[0]) {
		return &UnsupportedQueryError{Message: fmt.Sprintf("%s is numeric operator, but the right operand is not numeric", c.Operator)}
	}
	if slice.StringsAnyEquals(c.RightOp, "") {
		return &UnsupportedQueryError{Message: "right operand must have value"}
	}
	return nil
}

func mergeCriteria(c1 []Criterion, c2 []Criterion) ([]Criterion, error) {
	result := c1
	fieldQueryLeftOperands := make(map[string]bool)
	for _, criterion := range c1 {
		if criterion.Type == FieldQuery {
			fieldQueryLeftOperands[criterion.LeftOp] = true
		}
	}

	for _, newCriterion := range c2 {
		if _, ok := fieldQueryLeftOperands[newCriterion.LeftOp]; ok && newCriterion.Type == FieldQuery {
			return nil, &UnsupportedQueryError{Message: fmt.Sprintf("duplicate query key: %s", newCriterion.LeftOp)}
		}
		if err := newCriterion.Validate(); err != nil {
			return nil, err
		}
	}
	result = append(result, c2...)
	return result, nil
}

type criteriaCtxKey struct{}

// AddCriteria adds the given criteria to the context and returns an error if any of the criteria is not valid
func AddCriteria(ctx context.Context, newCriteria ...Criterion) (context.Context, error) {
	currentCriteria := CriteriaForContext(ctx)
	criteria, err := mergeCriteria(currentCriteria, newCriteria)
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, criteriaCtxKey{}, criteria), nil
}

// CriteriaForContext returns the criteria for the given context
func CriteriaForContext(ctx context.Context) []Criterion {
	currentCriteria := ctx.Value(criteriaCtxKey{})
	if currentCriteria == nil {
		return []Criterion{}
	}
	return currentCriteria.([]Criterion)
}

// BuildCriteriaFromRequest builds criteria for the given request's query params and returns an error if the query is not valid
func BuildCriteriaFromRequest(request *web.Request) ([]Criterion, error) {
	var criteria []Criterion
	for _, queryType := range supportedQueryTypes {
		queryValues := request.URL.Query().Get(string(queryType))
		querySegments, err := process(queryValues, queryType)
		if err != nil {
			return nil, err
		}
		if criteria, err = mergeCriteria(criteria, querySegments); err != nil {
			return nil, err
		}
	}
	return criteria, nil
}

func process(input string, criteriaType CriterionType) ([]Criterion, error) {
	var c []Criterion
	if input != "" {
		operator, err := getOperator(input)
		if err != nil {
			return nil, err
		}
		criterion := convertRawStatementToCriterion(input, operator, criteriaType)
		c = append(c, criterion)
	}
	return c, nil
}

func getOperator(rawStatement string) (Operator, error) {
	var opIdx int
	for _, op := range operators {
		opIdx = strings.Index(rawStatement, fmt.Sprintf(" %s ", string(op)))
		if opIdx != -1 {
			return op, nil
		}
	}
	return "", &UnsupportedQueryError{"query operator is missing"}
}

func convertRawStatementToCriterion(rawStatement string, operator Operator, criterionType CriterionType) Criterion {
	rawStatement = strings.TrimSpace(rawStatement)

	opIdx := strings.Index(rawStatement, string(operator))
	rightOp := strings.Split(rawStatement[opIdx+len(operator):], ",")

	for i := range rightOp {
		rightOp[i] = strings.TrimSpace(rightOp[i])
	}

	if operator.IsMultiVariate() {
		rightOp[0] = strings.TrimPrefix(rightOp[0], OpenBracket)
		rightOp[len(rightOp)-1] = strings.TrimSuffix(rightOp[len(rightOp)-1], CloseBracket)
	}
	return newCriterion(strings.TrimSpace(rawStatement[:opIdx]), operator, rightOp, criterionType)
}

func isNumeric(str string) bool {
	_, err := strconv.Atoi(str)
	if err == nil {
		return true
	}
	_, err = strconv.ParseFloat(str, 64)
	return err == nil
}
