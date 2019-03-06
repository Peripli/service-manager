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
	"sort"
	"strconv"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/query/parser"
	"github.com/antlr/antlr4/runtime/Go/antlr"

	"github.com/Peripli/service-manager/pkg/util"
)

// Operator is a query operator
type Operator string

const (
	// EqualsOperator takes two operands and tests if they are equal
	EqualsOperator Operator = "eq"
	// NotEqualsOperator takes two operands and tests if they are not equal
	NotEqualsOperator Operator = "neq"
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
	// Separator is the separator between field and label queries
	Separator rune = '|'
)

// CriterionType is a type of criteria to be applied when querying
type CriterionType string

const (
	// FieldQuery denotes that the query should be executed on the entity's fields
	FieldQuery CriterionType = "fieldQuery"
	// LabelQuery denotes that the query should be executed on the entity's labels
	LabelQuery CriterionType = "labelQuery"
)

var SupportedQueryTypes = []CriterionType{FieldQuery, LabelQuery}

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
		return fmt.Errorf("multiple values %s received for single value operation %s", c.RightOp, c.Operator)
	}
	if c.Operator.IsNullable() && c.Type != FieldQuery {
		return &util.UnsupportedQueryError{Message: "nullable operations are supported only for field queries"}
	}
	if c.Operator.IsNumeric() && !isNumeric(c.RightOp[0]) {
		return &util.UnsupportedQueryError{Message: fmt.Sprintf("%s is numeric operator, but the right operand %s is not numeric", c.Operator, c.RightOp[0])}
	}
	if strings.ContainsRune(c.LeftOp, Separator) {
		parts := strings.FieldsFunc(c.LeftOp, func(r rune) bool {
			return r == Separator
		})
		possibleKey := parts[len(parts)-1]
		return &util.UnsupportedQueryError{Message: fmt.Sprintf("separator %c is not allowed in %s with left operand \"%s\". Maybe you meant \"%s\"? Make sure if the separator is present in any right operand, that it is escaped with a backslash (\\)", Separator, c.Type, c.LeftOp, possibleKey)}
	}
	for _, op := range c.RightOp {
		if strings.ContainsRune(op, '\n') {
			return fmt.Errorf("%s with key \"%s\" has value \"%s\" contaning forbidden new line character", c.Type, c.LeftOp, op)
		}
	}
	return nil
}

func MergeCriteria(c1 []Criterion, c2 []Criterion) ([]Criterion, error) {
	result := c1
	fieldQueryLeftOperands := make(map[string]int)
	labelQueryLeftOperands := make(map[string]int)

	for _, criterion := range append(c1, c2...) {
		if criterion.Type == FieldQuery {
			fieldQueryLeftOperands[criterion.LeftOp]++
		}
		if criterion.Type == LabelQuery {
			labelQueryLeftOperands[criterion.LeftOp]++
		}
	}

	for _, newCriterion := range c2 {
		leftOp := newCriterion.LeftOp
		// disallow duplicate label queries
		if count, ok := labelQueryLeftOperands[leftOp]; ok && count > 1 && newCriterion.Type == LabelQuery {
			return nil, &util.UnsupportedQueryError{Message: fmt.Sprintf("duplicate label query key: %s", newCriterion.LeftOp)}
		}
		// disallow duplicate field query keys
		if count, ok := fieldQueryLeftOperands[leftOp]; ok && count > 1 && newCriterion.Type == FieldQuery {
			return nil, &util.UnsupportedQueryError{Message: fmt.Sprintf("duplicate field query key: %s", newCriterion.LeftOp)}
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
	criteria, err := MergeCriteria(currentCriteria, newCriteria)
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

// ContextWithCriteria returns a new context with given criteria
func ContextWithCriteria(ctx context.Context, criteria []Criterion) context.Context {
	return context.WithValue(ctx, criteriaCtxKey{}, criteria)
}

// BuildCriteriaFromRequest builds criteria for the given request's query params and returns an error if the query is not valid
func BuildCriteriaFromRequest(request *web.Request) ([]Criterion, error) {
	var criteria []Criterion
	for _, queryType := range SupportedQueryTypes {
		queryValues := request.URL.Query().Get(string(queryType))
		querySegments, err := Parse(queryType, queryValues)
		if err != nil {
			return nil, err
		}
		if criteria, err = MergeCriteria(criteria, querySegments); err != nil {
			return nil, err
		}
	}
	sort.Slice(criteria, func(i, j int) bool {
		return criteria[i].LeftOp < criteria[j].LeftOp
	})
	return criteria, nil
}

// Parse parses the query expression for and builds criteria for the provided type
func Parse(criterionType CriterionType, expression string) ([]Criterion, error) {
	if expression == "" {
		return []Criterion{}, nil
	}
	parsingListener := &queryListener{criteriaType: criterionType}

	input := antlr.NewInputStream(expression)
	lexer := parser.NewQueryLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)

	p := parser.NewQueryParser(stream)
	p.RemoveErrorListeners()
	p.AddErrorListener(parsingListener)

	antlr.ParseTreeWalkerDefault.Walk(parsingListener, p.Expression())
	if parsingListener.err != nil {
		return nil, parsingListener.err
	}

	criteria := parsingListener.result
	sort.Slice(criteria, func(i, j int) bool {
		return criteria[i].LeftOp < criteria[j].LeftOp
	})
	return criteria, nil
}

func isNumeric(str string) bool {
	_, err := strconv.Atoi(str)
	if err == nil {
		return true
	}
	_, err = strconv.ParseFloat(str, 64)
	return err == nil
}
