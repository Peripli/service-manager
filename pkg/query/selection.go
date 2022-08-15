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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query/parser"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
)

const (
	// Separator is the separator between queries of different type
	Separator string = "and"
)

// CriterionType is a type of criteria to be applied when querying
type CriterionType string

const (
	// FieldQuery denotes that the query should be executed on the entity's fields
	FieldQuery CriterionType = "fieldQuery"
	// LabelQuery denotes that the query should be executed on the entity's labels
	LabelQuery CriterionType = "labelQuery"
	// ResultQuery is used to further process result
	ResultQuery CriterionType = "resultQuery"
	// ExistQuery denotes that the query should test for the existence of any record in a given sub-query
	ExistQuery CriterionType = "existQuery"
	// InQuery denotes that the query
	Subquery CriterionType = "subQuery"
)

// OperatorType represents the type of the query operator
type OperatorType string

const (
	// UnivariateOperator denotes that the operator expects exactly one variable on the right side
	UnivariateOperator OperatorType = "univariate"
	// MultivariateOperator denotes that the operator expects more than one variable on the right side
	MultivariateOperator OperatorType = "multivariate"
)

// OrderType is the type of the order in which result is presented
type OrderType string

const (
	// AscOrder orders result in ascending order
	AscOrder OrderType = "ASC"
	// DescOrder orders result in descending order
	DescOrder OrderType = "DESC"
)

const (
	// OrderBy should be used as a left operand in Criterion
	OrderBy string = "orderBy"
	// Limit should be used as a left operand in Criterion to signify the
	Limit string = "limit"
)

var (
	// Operators returns the supported query operators
	Operators = []Operator{
		EqualsOperator, NotEqualsOperator,
		GreaterThanOperator, LessThanOperator,
		GreaterThanOrEqualOperator, LessThanOrEqualOperator,
		InOperator, NotInOperator, EqualsOrNilOperator, ContainsOperator,
	}
	// CriteriaTypes returns the supported query criteria types
	CriteriaTypes = []CriterionType{FieldQuery, LabelQuery, ExistQuery, Subquery}
)

// Operator is a query operator
type Operator interface {
	// String returns the text representation of the operator
	String() string
	// Type returns the type of the operator
	Type() OperatorType
	// IsNullable returns true if the operator allows results with null value in the RHS
	IsNullable() bool
	// IsNumeric returns true if the operator works only with numbers
	IsNumeric() bool
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
	return NewCriterion(leftOp, operator, rightOp, FieldQuery)
}

func ByNotExists(subQuery string) Criterion {
	return NewCriterion("", NotExistsSubquery, []string{subQuery}, ExistQuery)
}

func ByExists(subQuery string) Criterion {
	return NewCriterion("", ExistsSubquery, []string{subQuery}, ExistQuery)
}

func BySubquery(operator Operator, leftOp string, subQuery string) Criterion {
	return NewCriterion(leftOp, operator, []string{subQuery}, Subquery)
}

// ByLabel constructs a new criterion for label querying
func ByLabel(operator Operator, leftOp string, rightOp ...string) Criterion {
	return NewCriterion(leftOp, operator, rightOp, LabelQuery)
}

// OrderResultBy constructs a new criterion for result order
func OrderResultBy(field string, orderType OrderType) Criterion {
	return NewCriterion(OrderBy, NoOperator, []string{field, string(orderType)}, ResultQuery)
}

// LimitResultBy constructs a new criterion for limit result with
func LimitResultBy(limit int) Criterion {
	limitString := strconv.Itoa(limit)
	return NewCriterion(Limit, NoOperator, []string{limitString}, ResultQuery)
}

func NewCriterion(leftOp string, operator Operator, rightOp []string, criteriaType CriterionType) Criterion {
	return Criterion{LeftOp: leftOp, Operator: operator, RightOp: rightOp, Type: criteriaType}
}

// Validate the criterion fields
func (c Criterion) Validate() error {
	if len(c.RightOp) == 0 {
		return errors.New("missing right operand")
	}

	if c.Type == ResultQuery {
		if c.LeftOp == Limit {
			limit, err := strconv.Atoi(c.RightOp[0])
			if err != nil {
				return fmt.Errorf("could not convert string to int: %s", err.Error())
			}
			if limit < 0 {
				return &util.UnsupportedQueryError{Message: fmt.Sprintf("limit (%d) is invalid. Limit should be positive number", limit)}
			}
		}

		if c.LeftOp == OrderBy {
			if len(c.RightOp) < 2 {
				return &util.UnsupportedQueryError{Message: "order by result expects field name and order type"}
			}
		}

		return nil
	}

	if len(c.RightOp) > 1 && c.Operator.Type() == UnivariateOperator {
		return &util.UnsupportedQueryError{Message: fmt.Sprintf("multiple values %s received for single value operation %s", c.RightOp, c.Operator)}
	}
	if c.Operator.IsNullable() && c.Type != FieldQuery {
		return &util.UnsupportedQueryError{Message: "nullable operations are supported only for field queries"}
	}
	if c.Operator.IsNumeric() && !isNumeric(c.RightOp[0]) && !isDateTime(c.RightOp[0]) {
		return &util.UnsupportedQueryError{Message: fmt.Sprintf("%s is numeric operator, but the right operand %s is not numeric or datetime", c.Operator, c.RightOp[0])}
	}
	if strings.Contains(c.LeftOp, fmt.Sprintf(" %s ", Separator)) ||
		strings.Contains(c.LeftOp, fmt.Sprintf("%s ", Separator)) ||
		strings.Contains(c.LeftOp, fmt.Sprintf(" %s", Separator)) ||
		c.LeftOp == Separator {
		return &util.UnsupportedQueryError{Message: fmt.Sprintf("separator %s is not allowed in %s with left operand \"%s\".", Separator, c.Type, c.LeftOp)}
	}
	for _, op := range c.RightOp {
		if strings.ContainsRune(op, '\n') && c.Type != ExistQuery && c.Type != Subquery {
			return &util.UnsupportedQueryError{Message: fmt.Sprintf("%s with key \"%s\" has value \"%s\" contaning forbidden new line character", c.Type, c.LeftOp, op)}
		}
	}
	return nil
}

func validateCriteria(criteria []Criterion) error {
	fieldQueryLeftOperands := make(map[string]int)
	labelQueryLeftOperands := make(map[string]int)

	for _, criterion := range criteria {
		if criterion.Type == FieldQuery {
			fieldQueryLeftOperands[criterion.LeftOp]++
		}
		if criterion.Type == LabelQuery {
			labelQueryLeftOperands[criterion.LeftOp]++
		}
	}

	for _, c := range criteria {
		leftOp := c.LeftOp
		// disallow duplicate label queries
		if count, ok := labelQueryLeftOperands[leftOp]; ok && count > 1 && c.Type == LabelQuery {
			return &util.UnsupportedQueryError{Message: fmt.Sprintf("duplicate label query key: %s", leftOp)}
		}
		if err := c.Validate(); err != nil {
			return err
		}
	}
	return validateWholeCriteria(criteria...)
}

type criteriaCtxKey struct{}

// AddCriteria adds the given criteria to the context and returns an error if any of the criteria is not valid
func AddCriteria(ctx context.Context, newCriteria ...Criterion) (context.Context, error) {
	currentCriteria := CriteriaForContext(ctx)
	criteria := append(currentCriteria, newCriteria...)
	if err := validateCriteria(criteria); err != nil {
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
func ContextWithCriteria(ctx context.Context, criteria ...Criterion) (context.Context, error) {
	if err := validateCriteria(criteria); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, criteriaCtxKey{}, criteria), nil
}

// Parse parses the query expression for and builds criteria for the provided type
func Parse(criterionType CriterionType, expression string) ([]Criterion, error) {
	if expression == "" {
		return []Criterion{}, nil
	}
	parsingListener := &queryListener{criteriaType: criterionType}

	input := antlr.NewInputStream(expression)
	lexer := parser.NewQueryLexer(input)
	lexer.RemoveErrorListeners()
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewQueryParser(stream)
	p.RemoveErrorListeners()
	p.AddErrorListener(parsingListener)

	antlr.ParseTreeWalkerDefault.Walk(parsingListener, p.Expression())
	if parsingListener.err != nil {
		return nil, parsingListener.err
	}

	criteria := parsingListener.result
	if err := validateCriteria(criteria); err != nil {
		return nil, err
	}
	sort.Slice(criteria, func(i, j int) bool {
		return criteria[i].LeftOp < criteria[j].LeftOp
	})
	return criteria, nil
}

// RetrieveFromCriteria searches for the value (rightOp) of a given key (leftOp) in a set of criteria
func RetrieveFromCriteria(key string, criteria ...Criterion) string {
	for _, criterion := range criteria {
		if criterion.LeftOp == key {
			return criterion.RightOp[0]
		}
	}
	return ""
}

func isNumeric(str string) bool {
	_, err := strconv.Atoi(str)
	if err == nil {
		return true
	}
	_, err = strconv.ParseFloat(str, 64)
	return err == nil
}

func isDateTime(str string) bool {
	_, err := time.Parse(time.RFC3339, str)
	return err == nil
}

func validateWholeCriteria(criteria ...Criterion) error {
	isLimited := false
	for _, criterion := range criteria {
		if criterion.LeftOp == Limit {
			if isLimited {
				return fmt.Errorf("zero/one limit criterion expected but multiple provided")
			}
			isLimited = true
		}
	}
	return nil
}
