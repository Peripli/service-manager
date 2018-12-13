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

	"github.com/Peripli/service-manager/pkg/web"
)

type Operator string

const (
	EqualsOperator      Operator = "="
	NotEqualsOperator   Operator = "!="
	GreaterThanOperator Operator = "gt"
	LessThanOperator    Operator = "lt"
	InOperator          Operator = "in"
	NotInOperator       Operator = "notin"
	EqualsOrNilOperator Operator = "eqornil"
)

func (op Operator) IsMultiVariate() bool {
	return op == InOperator || op == NotInOperator
}

func (op Operator) IsNullable() bool {
	return op == EqualsOrNilOperator
}

func (op Operator) IsNumeric() bool {
	return op == LessThanOperator || op == GreaterThanOperator
}

var operators = []Operator{EqualsOperator, NotEqualsOperator, InOperator,
	NotInOperator, GreaterThanOperator, LessThanOperator, EqualsOrNilOperator}

type CriterionType string

const (
	FieldQuery CriterionType = "fieldQuery"
	LabelQuery CriterionType = "labelQuery"
)

var supportedQueryTypes = []CriterionType{FieldQuery, LabelQuery}
var allowedSeparators = []rune{';'}

type UnsupportedQuery struct {
	Message string
}

func (uq *UnsupportedQuery) Error() string {
	return uq.Message
}

type Criterion struct {
	LeftOp   string
	Operator Operator
	RightOp  []string
	Type     CriterionType
}

func ByField(operator Operator, leftOp string, rightOp ...string) Criterion {
	return newCriterion(leftOp, operator, rightOp, FieldQuery)
}

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
		return &UnsupportedQuery{"nullable operations are supported only for field queries"}
	}
	if c.Operator.IsNumeric() && !isNumeric(c.RightOp[0]) {
		return &UnsupportedQuery{Message: fmt.Sprintf("%s is numeric operator, but the right operand is not numeric", c.Operator)}
	}
	return nil
}

type criteria []Criterion

func (c *criteria) add(criteria ...Criterion) error {
	fieldQueryLeftOperands := make(map[string]bool)
	for _, criterion := range *c {
		if criterion.Type == FieldQuery {
			fieldQueryLeftOperands[criterion.LeftOp] = true
		}
	}
	for _, newCriterion := range criteria {
		if _, ok := fieldQueryLeftOperands[newCriterion.LeftOp]; ok && newCriterion.Type == FieldQuery {
			return &UnsupportedQuery{Message: fmt.Sprintf("duplicate query key: %s", newCriterion.LeftOp)}
		}
		if err := newCriterion.Validate(); err != nil {
			return err
		}
		*c = append(*c, newCriterion)
	}
	return nil
}

type criteriaCtxKey struct{}

func AddCriteria(ctx context.Context, criterion ...Criterion) (context.Context, error) {
	var currentCriteria criteria = CriteriaForContext(ctx)
	for _, c := range criterion {
		if err := currentCriteria.add(c); err != nil {
			return nil, err
		}
	}
	return context.WithValue(ctx, criteriaCtxKey{}, currentCriteria), nil
}

func CriteriaForContext(ctx context.Context) []Criterion {
	currentCriteria := ctx.Value(criteriaCtxKey{})
	if currentCriteria == nil {
		return []Criterion{}
	}
	return currentCriteria.(criteria)
}

func BuildCriteriaFromRequest(request *web.Request) ([]Criterion, error) {
	var criteria criteria
	for _, queryType := range supportedQueryTypes {
		queryValues := request.URL.Query().Get(string(queryType))
		querySegments, err := process(queryValues, queryType)
		if err != nil {
			return nil, err
		}
		if err := criteria.add(querySegments...); err != nil {
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
	return "", fmt.Errorf("query operator is missing")
}

func convertRawStatementToCriterion(rawStatement string, operator Operator, criterionType CriterionType) Criterion {
	rawStatement = strings.TrimSpace(rawStatement)

	opIdx := strings.Index(rawStatement, string(operator))
	rightOp := strings.Split(rawStatement[opIdx+len(operator):], ",")

	for i := range rightOp {
		rightOp[i] = strings.TrimSpace(rightOp[i])
	}

	if operator.IsMultiVariate() {
		rightOp[0] = strings.TrimPrefix(rightOp[0], "[")
		rightOp[len(rightOp)-1] = strings.TrimSuffix(rightOp[len(rightOp)-1], "]")
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
